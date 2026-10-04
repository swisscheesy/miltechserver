package items

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	. "github.com/go-jet/jet/v2/postgres"
	"log/slog"
	"miltechserver/.gen/miltech_ng/public/model"
	. "miltechserver/.gen/miltech_ng/public/table"
	"miltechserver/api/shops/shared"
	"time"
)

type MetadataIntent struct {
	ItemID, NotificationID, Niin string
	Nickname, UnitOfMeasure      *string
}
type ResolvedMetadata struct{ Nickname, UnitOfMeasure *string }

type metadataCandidate struct {
	ItemID        string  `json:"item_id"`
	Nickname      *string `json:"nickname"`
	UnitOfMeasure *string `json:"unit_of_measure"`
	Version       int64   `json:"version"`
}

type metadataAmbiguity struct {
	notificationID, niin string
	version              int64
	candidates           []metadataCandidate
}

func (*metadataAmbiguity) Error() string { return "notification item metadata is ambiguous" }
func (*metadataAmbiguity) Unwrap() error {
	return &shared.Failure{Code: "notification_item_metadata_conflict", PublicMessage: "Notification item metadata requires explicit resolution", Status: 409}
}

type metadataVersionsKey struct{}
type metadataKey struct{ notificationID, niin string }
type metadataObservation struct {
	version    int64
	candidates []metadataCandidate
}

// MetadataMutationContext starts one transaction attempt's durable observations.
// Retry callers create a fresh scope; rolled-back values are never evidence.
func MetadataMutationContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, metadataVersionsKey{}, map[metadataKey]metadataObservation{})
}
func observeMetadata(ctx context.Context, row model.ShopNotificationItemMetadata, candidates []metadataCandidate) metadataObservation {
	observation := metadataObservation{row.Version, append([]metadataCandidate{}, candidates...)}
	observations, ok := ctx.Value(metadataVersionsKey{}).(map[metadataKey]metadataObservation)
	if !ok {
		return observation
	}
	key := metadataKey{row.NotificationID, row.Niin}
	if previous, ok := observations[key]; ok {
		return previous
	}
	observations[key] = observation
	return observation
}

func sameNullable(a, b *string) bool { return a == nil && b == nil || a != nil && b != nil && *a == *b }
func sameMetadata(a, b ResolvedMetadata) bool {
	return sameNullable(a.Nickname, b.Nickname) && sameNullable(a.UnitOfMeasure, b.UnitOfMeasure)
}
func candidateMetadata(c metadataCandidate) ResolvedMetadata {
	return ResolvedMetadata{c.Nickname, c.UnitOfMeasure}
}
func candidatesDisagree(candidates []metadataCandidate) bool {
	for _, c := range candidates {
		if !sameMetadata(candidateMetadata(candidates[0]), candidateMetadata(c)) {
			return true
		}
	}
	return false
}

// The notification coordinating lock must already be held. It serializes absent
// logical keys too; a row lock alone cannot protect a first retained INSERT.
func loadMetadata(ctx context.Context, tx *sql.Tx, notificationID, niin string) (model.ShopNotificationItemMetadata, []metadataCandidate, error) {
	m := ShopNotificationItemMetadata
	var retained []model.ShopNotificationItemMetadata
	err := SELECT(m.AllColumns).FROM(m).WHERE(m.NotificationID.EQ(String(notificationID)).AND(m.Niin.EQ(String(niin)))).FOR(UPDATE()).QueryContext(ctx, tx, &retained)
	if err != nil {
		return model.ShopNotificationItemMetadata{}, nil, err
	}
	row := model.ShopNotificationItemMetadata{NotificationID: notificationID, Niin: niin, State: "resolved", Candidates: "[]"}
	if len(retained) > 0 {
		row = retained[0]
	}
	i := ShopNotificationItems
	var active []model.ShopNotificationItems
	err = SELECT(i.AllColumns).FROM(i).WHERE(i.NotificationID.EQ(String(notificationID)).AND(i.Niin.EQ(String(niin)))).ORDER_BY(i.ID.ASC()).FOR(UPDATE()).QueryContext(ctx, tx, &active)
	candidates := make([]metadataCandidate, 0, len(active))
	for _, item := range active {
		candidates = append(candidates, metadataCandidate{item.ID, item.Nickname, item.UnitOfMeasure, row.Version + 1})
	}
	if err == nil {
		observeMetadata(ctx, row, candidates)
	}
	return row, candidates, err
}

func writeMetadata(ctx context.Context, tx *sql.Tx, row model.ShopNotificationItemMetadata, candidates []metadataCandidate) error {
	var history []metadataCandidate
	if err := json.Unmarshal([]byte(row.Candidates), &history); err != nil {
		return err
	}
	for _, candidate := range candidates {
		found := false
		for _, prior := range history {
			if prior.ItemID == candidate.ItemID && sameMetadata(candidateMetadata(prior), candidateMetadata(candidate)) {
				found = true
				break
			}
		}
		if !found {
			history = append(history, candidate)
		}
	}
	encoded, err := json.Marshal(history)
	if err != nil {
		return err
	}
	row.Candidates = string(encoded)
	row.Version++
	m := ShopNotificationItemMetadata
	if row.Version == 1 {
		_, err = m.INSERT(m.AllColumns).MODEL(row).ExecContext(ctx, tx)
	} else {
		_, err = m.UPDATE(m.MutableColumns).MODEL(row).WHERE(m.NotificationID.EQ(String(row.NotificationID)).AND(m.Niin.EQ(String(row.Niin)))).ExecContext(ctx, tx)
	}
	return err
}

// ResolveRetainedMetadata never changes quantity or creates an active item.
// Presence must reach here before any enrichment defaults are applied.
func ResolveRetainedMetadata(ctx context.Context, tx *sql.Tx, intent MetadataIntent) (ResolvedMetadata, error) {
	row, active, err := loadMetadata(ctx, tx, intent.NotificationID, intent.Niin)
	if err != nil {
		return ResolvedMetadata{}, err
	}
	var physical []model.ShopNotificationItems
	i := ShopNotificationItems
	err = SELECT(i.AllColumns).FROM(i).WHERE(i.ID.EQ(String(intent.ItemID)).AND(i.NotificationID.EQ(String(intent.NotificationID)))).QueryContext(ctx, tx, &physical)
	if err != nil {
		return ResolvedMetadata{}, err
	}
	resolved := ResolvedMetadata{intent.Nickname, intent.UnitOfMeasure}
	existing := len(physical) > 0
	if existing {
		// A changed NIIN still preserves omissions from this physical row only.
		if resolved.Nickname == nil {
			resolved.Nickname = physical[0].Nickname
		}
		if resolved.UnitOfMeasure == nil {
			resolved.UnitOfMeasure = physical[0].UnitOfMeasure
		}
	} else if intent.Nickname == nil || intent.UnitOfMeasure == nil {
		if row.State == "ambiguous" || candidatesDisagree(active) {
			observed := observeMetadata(ctx, row, active)
			return resolved, &metadataAmbiguity{intent.NotificationID, intent.Niin, observed.version, observed.candidates}
		}
		fallback := ResolvedMetadata{row.Nickname, row.UnitOfMeasure}
		if len(active) > 0 {
			fallback = candidateMetadata(active[0])
		}
		if resolved.Nickname == nil {
			resolved.Nickname = fallback.Nickname
		}
		if resolved.UnitOfMeasure == nil {
			resolved.UnitOfMeasure = fallback.UnitOfMeasure
		}
	}
	resulting := []metadataCandidate{}
	for _, c := range active {
		if c.ItemID != intent.ItemID {
			resulting = append(resulting, c)
		}
	}
	desired := metadataCandidate{intent.ItemID, resolved.Nickname, resolved.UnitOfMeasure, row.Version + 1}
	resulting = append(resulting, desired)
	// Only an explicit operation establishing agreement may clear old ambiguity.
	explicitResolution := existing && (intent.Nickname != nil || intent.UnitOfMeasure != nil) || intent.Nickname != nil && intent.UnitOfMeasure != nil
	if candidatesDisagree(resulting) || row.State == "ambiguous" && !explicitResolution {
		row.State = "ambiguous"
	} else {
		row.State = "resolved"
		row.Nickname = resolved.Nickname
		row.UnitOfMeasure = resolved.UnitOfMeasure
		if explicitResolution {
			// Only an accepted explicit operation may supersede previously
			// observed ambiguity. Deletion and quantity-only writes cannot.
			row.ResolutionVersion = row.Version + 1
		}
	}
	return resolved, writeMetadata(ctx, tx, row, append(active, desired))
}

// RetainItemMetadata runs before deletion, so even the last conflicting
// physical row leaves its raw evidence and sticky ambiguity behind.
func RetainItemMetadata(ctx context.Context, tx *sql.Tx, item model.ShopNotificationItems) error {
	row, active, err := loadMetadata(ctx, tx, item.NotificationID, item.Niin)
	if err != nil {
		return err
	}
	evidence := append(active, metadataCandidate{item.ID, item.Nickname, item.UnitOfMeasure, row.Version + 1})
	if row.State == "ambiguous" || candidatesDisagree(evidence) {
		row.State = "ambiguous"
	} else {
		row.Nickname = item.Nickname
		row.UnitOfMeasure = item.UnitOfMeasure
	}
	return writeMetadata(ctx, tx, row, evidence)
}

// PersistMetadataAmbiguity records integrity evidence after the business
// transaction rolled back. It cannot commit failed business writes or receipts.
// Fresh authorization and explicit-resolution provenance prevent stale evidence from undoing
// an intervening accepted explicit resolution. The caller's lifetime bounds it.
func PersistMetadataAmbiguity(ctx context.Context, db *sql.DB, userID string, businessErr error) {
	var observed *metadataAmbiguity
	if !errors.As(businessErr, &observed) || !candidatesDisagree(observed.candidates) {
		return
	}
	evidenceCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	err := shared.WithNotificationMutation(evidenceCtx, db, func(tx *sql.Tx) error {
		if _, _, err := shared.LockNotificationMutation(evidenceCtx, tx, userID, observed.notificationID); err != nil {
			return err
		}
		row, active, err := loadMetadata(evidenceCtx, tx, observed.notificationID, observed.niin)
		if err != nil {
			return err
		}
		if row.ResolutionVersion > observed.version {
			return nil
		}
		row.State = "ambiguous"
		return writeMetadata(evidenceCtx, tx, row, append(observed.candidates, active...))
	})
	if err != nil {
		slog.Warn("Notification metadata ambiguity evidence unavailable", "notification_id", observed.notificationID, "actor_id", userID, "failure_category", "integrity_evidence_unavailable")
	}
}
