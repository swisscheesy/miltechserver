package sync

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"miltechserver/api/response"
	"miltechserver/api/user_pmcs/persistence"
	"miltechserver/api/user_pmcs/shared"
)

type RepositoryImpl struct {
	store persistence.Store
}

type deltaRoot struct {
	version  int64
	kind     string
	identity uuid.UUID
}

func NewRepository(store persistence.Store) Repository {
	return &RepositoryImpl{store: store}
}

func (repository *RepositoryImpl) GetDelta(
	ctx context.Context,
	userUID string,
	after int64,
	limit int,
	byteLimit int,
) (*AccountDelta, error) {
	startedAt := time.Now()
	defer func() {
		shared.RecordDBDuration(ctx, time.Since(startedAt))
	}()

	tx, err := repository.store.DB.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelRepeatableRead,
		ReadOnly:  true,
	})
	if err != nil {
		return nil, fmt.Errorf("begin account delta snapshot: %w", err)
	}

	accountVersion, err := readAccountVersion(ctx, tx, userUID)
	if err != nil {
		return nil, rollbackDelta(tx, err)
	}
	roots, err := readDeltaRoots(ctx, tx, userUID, after, limit+1)
	if err != nil {
		return nil, rollbackDelta(tx, err)
	}

	pageRoots := roots[:min(limit, len(roots))]
	changes, err := loadAccountChanges(ctx, tx, userUID, pageRoots)
	if err != nil {
		return nil, rollbackDelta(tx, err)
	}
	rootTruncated := len(roots) > limit
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit account delta snapshot: %w", err)
	}
	included, byteTruncated, err := fitCompleteChanges(
		changes,
		after,
		accountVersion,
		rootTruncated,
		byteLimit,
	)
	if err != nil {
		return nil, err
	}
	through := after
	if len(included) > 0 {
		through = included[len(included)-1].AccountChangeVersion
	}
	return &AccountDelta{
		FromCursor:     after,
		ThroughCursor:  through,
		AccountVersion: accountVersion,
		HasMore:        rootTruncated || byteTruncated,
		Changes:        included,
	}, nil
}

func readAccountVersion(
	ctx context.Context,
	tx *sql.Tx,
	userUID string,
) (int64, error) {
	var (
		initialized bool
		version     int64
	)
	err := tx.QueryRowContext(
		ctx,
		`SELECT EXISTS(SELECT 1 FROM users WHERE uid = $1),
		        COALESCE(
		            (SELECT current_version
		             FROM user_pmcs_sync_state
		             WHERE user_uid = $1),
		            0
		        )`,
		userUID,
	).Scan(&initialized, &version)
	if err != nil {
		return 0, fmt.Errorf("read account delta version: %w", err)
	}
	if !initialized {
		return 0, shared.NewAccountNotInitialized(
			"account is not initialized",
			nil,
		)
	}
	return version, nil
}

func readDeltaRoots(
	ctx context.Context,
	tx *sql.Tx,
	userUID string,
	after int64,
	limit int,
) ([]deltaRoot, error) {
	rows, err := tx.QueryContext(
		ctx,
		`/* user_pmcs_account_delta_roots */
		 SELECT account_change_version, kind, identity
		 FROM (
		     SELECT account_change_version,
		            'checklist' AS kind,
		            id AS identity
		     FROM user_pmcs_checklists
		     WHERE owner_uid = $1 AND account_change_version > $2
		     UNION ALL
		     SELECT account_change_version,
		            'subscription' AS kind,
		            checklist_id AS identity
		     FROM user_pmcs_subscriptions
		     WHERE subscriber_uid = $1 AND account_change_version > $2
		 ) AS changed_roots
		 ORDER BY account_change_version
		 LIMIT $3`,
		userUID,
		after,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("query account delta roots: %w", err)
	}
	var roots []deltaRoot
	for rows.Next() {
		var root deltaRoot
		if err := rows.Scan(&root.version, &root.kind, &root.identity); err != nil {
			return nil, closeDeltaRows(
				rows,
				fmt.Errorf("scan account delta root: %w", err),
			)
		}
		roots = append(roots, root)
	}
	if err := rows.Err(); err != nil {
		return nil, closeDeltaRows(
			rows,
			fmt.Errorf("iterate account delta roots: %w", err),
		)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("close account delta root rows: %w", err)
	}
	return roots, nil
}

func loadAccountChanges(
	ctx context.Context,
	tx *sql.Tx,
	userUID string,
	roots []deltaRoot,
) ([]AccountChange, error) {
	checklistIDs := rootIDs(roots, ChangeKindChecklist)
	subscriptionIDs := rootIDs(roots, ChangeKindSubscription)
	checklists, revisionIDs, err := loadChecklistChanges(
		ctx,
		tx,
		userUID,
		checklistIDs,
	)
	if err != nil {
		return nil, err
	}
	subscriptions, installedIDs, err := loadSubscriptionChanges(
		ctx,
		tx,
		userUID,
		subscriptionIDs,
	)
	if err != nil {
		return nil, err
	}
	revisionIDs = append(revisionIDs, installedIDs...)
	trees, err := persistence.LoadRevisionTrees(ctx, tx, uniqueUUIDs(revisionIDs))
	if err != nil {
		return nil, err
	}

	changes := make([]AccountChange, 0, len(roots))
	for _, root := range roots {
		change := AccountChange{
			AccountChangeVersion: root.version,
			Kind:                 root.kind,
		}
		switch root.kind {
		case ChangeKindChecklist:
			loaded, found := checklists[root.identity]
			if !found {
				return nil, fmt.Errorf("account delta checklist root disappeared")
			}
			if err := attachChecklistTrees(&loaded, trees); err != nil {
				return nil, err
			}
			change.Checklist = &loaded.aggregate
		case ChangeKindSubscription:
			loaded, found := subscriptions[root.identity]
			if !found {
				return nil, fmt.Errorf("account delta subscription root disappeared")
			}
			change.Subscription = &loaded.subscription
			if loaded.installed != nil {
				tree, found := trees[loaded.installed.Revision.ID]
				if !found {
					return nil, fmt.Errorf("installed revision tree disappeared")
				}
				loaded.installed.Revision = tree
				change.Installed = loaded.installed
			}
		default:
			return nil, fmt.Errorf("unsupported account delta kind %q", root.kind)
		}
		changes = append(changes, change)
	}
	return changes, nil
}

type loadedChecklist struct {
	aggregate     shared.ChecklistAggregate
	draftID       uuid.NullUUID
	publicationID uuid.NullUUID
}

func loadChecklistChanges(
	ctx context.Context,
	tx *sql.Tx,
	userUID string,
	ids []uuid.UUID,
) (map[uuid.UUID]loadedChecklist, []uuid.UUID, error) {
	loaded := make(map[uuid.UUID]loadedChecklist, len(ids))
	if len(ids) == 0 {
		return loaded, nil, nil
	}
	rows, err := tx.QueryContext(
		ctx,
		`SELECT checklist.id, checklist.sync_version,
		        checklist.account_change_version, checklist.created_at,
		        checklist.updated_at, checklist.deleted_at,
		        draft.id, publication.id,
		        source.status, source.current_release_revision_id,
		        source.latest_release_revision_number,
		        source.first_released_at, source.updated_at, source.retired_at
		 FROM user_pmcs_checklists AS checklist
		 LEFT JOIN user_pmcs_revisions AS draft
		   ON draft.checklist_id = checklist.id AND draft.state = 'draft'
		 LEFT JOIN user_pmcs_revisions AS publication
		   ON publication.checklist_id = checklist.id
		  AND publication.state = 'published'
		 LEFT JOIN user_pmcs_community_sources AS source
		   ON source.checklist_id = checklist.id
		 WHERE checklist.owner_uid = $1 AND checklist.id = ANY($2)
		 ORDER BY checklist.id`,
		userUID,
		pq.Array(ids),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("query account delta checklists: %w", err)
	}
	var revisionIDs []uuid.UUID
	for rows.Next() {
		var (
			entry            loadedChecklist
			sourceStatus     sql.NullString
			sourceRevisionID uuid.NullUUID
			sourceLatest     sql.NullInt32
			sourceFirst      sql.NullTime
			sourceUpdated    sql.NullTime
			sourceRetired    sql.NullTime
		)
		err := rows.Scan(
			&entry.aggregate.ID, &entry.aggregate.SyncVersion,
			&entry.aggregate.AccountChangeVersion, &entry.aggregate.CreatedAt,
			&entry.aggregate.UpdatedAt, &entry.aggregate.DeletedAt,
			&entry.draftID, &entry.publicationID, &sourceStatus,
			&sourceRevisionID, &sourceLatest, &sourceFirst, &sourceUpdated,
			&sourceRetired,
		)
		if err != nil {
			return nil, nil, closeDeltaRows(
				rows,
				fmt.Errorf("scan account delta checklist: %w", err),
			)
		}
		if entry.aggregate.DeletedAt == nil {
			if entry.draftID.Valid {
				revisionIDs = append(revisionIDs, entry.draftID.UUID)
			}
			if entry.publicationID.Valid {
				revisionIDs = append(revisionIDs, entry.publicationID.UUID)
			}
			if sourceStatus.Valid {
				entry.aggregate.Community = &shared.CommunitySourceSummary{
					Status:                      sourceStatus.String,
					CurrentReleaseRevisionID:    nullableUUID(sourceRevisionID),
					LatestReleaseRevisionNumber: sourceLatest.Int32,
					FirstReleasedAt:             sourceFirst.Time,
					UpdatedAt:                   sourceUpdated.Time,
					RetiredAt:                   nullableTime(sourceRetired),
				}
			}
		}
		loaded[entry.aggregate.ID] = entry
	}
	if err := finishDeltaRows(rows, "account delta checklist"); err != nil {
		return nil, nil, err
	}
	return loaded, revisionIDs, nil
}

type loadedSubscription struct {
	subscription shared.Subscription
	installed    *shared.InstalledChecklistRelease
}

func loadSubscriptionChanges(
	ctx context.Context,
	tx *sql.Tx,
	userUID string,
	ids []uuid.UUID,
) (map[uuid.UUID]loadedSubscription, []uuid.UUID, error) {
	loaded := make(map[uuid.UUID]loadedSubscription, len(ids))
	if len(ids) == 0 {
		return loaded, nil, nil
	}
	rows, err := tx.QueryContext(
		ctx,
		`SELECT subscription.checklist_id,
		        subscription.installed_revision_id,
		        subscription.sync_version,
		        subscription.account_change_version,
		        subscription.created_at, subscription.updated_at,
		        subscription.deleted_at,
		        source.status,
		        COALESCE(owner.username, 'Deleted user'),
		        release.released_at, revision.id
		 FROM user_pmcs_subscriptions AS subscription
		 LEFT JOIN user_pmcs_community_sources AS source
		   ON source.checklist_id = subscription.checklist_id
		 LEFT JOIN user_pmcs_checklists AS checklist
		   ON checklist.id = subscription.checklist_id
		 LEFT JOIN users AS owner ON owner.uid = checklist.owner_uid
		 LEFT JOIN user_pmcs_community_releases AS release
		   ON release.checklist_id = subscription.checklist_id
		  AND release.revision_id = subscription.installed_revision_id
		 LEFT JOIN user_pmcs_revisions AS revision
		   ON revision.id = subscription.installed_revision_id
		 WHERE subscription.subscriber_uid = $1
		   AND subscription.checklist_id = ANY($2)
		 ORDER BY subscription.checklist_id`,
		userUID,
		pq.Array(ids),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("query account delta subscriptions: %w", err)
	}
	var revisionIDs []uuid.UUID
	for rows.Next() {
		var (
			entry        loadedSubscription
			sourceStatus sql.NullString
			creator      sql.NullString
			releasedAt   sql.NullTime
			revisionID   uuid.NullUUID
		)
		err := rows.Scan(
			&entry.subscription.ChecklistID,
			&entry.subscription.InstalledRevisionID,
			&entry.subscription.SyncVersion,
			&entry.subscription.AccountChangeVersion,
			&entry.subscription.CreatedAt,
			&entry.subscription.UpdatedAt,
			&entry.subscription.DeletedAt,
			&sourceStatus, &creator, &releasedAt, &revisionID,
		)
		if err != nil {
			return nil, nil, closeDeltaRows(
				rows,
				fmt.Errorf("scan account delta subscription: %w", err),
			)
		}
		if entry.subscription.DeletedAt == nil {
			if !sourceStatus.Valid || !creator.Valid ||
				!releasedAt.Valid || !revisionID.Valid {
				return nil, nil, closeDeltaRows(
					rows,
					fmt.Errorf("active subscription release is incomplete"),
				)
			}
			entry.installed = &shared.InstalledChecklistRelease{
				ChecklistID:        entry.subscription.ChecklistID,
				SourceStatus:       sourceStatus.String,
				CreatorDisplayName: creator.String,
				ReleasedAt:         releasedAt.Time,
				Revision:           shared.Revision{ID: revisionID.UUID},
			}
			revisionIDs = append(revisionIDs, revisionID.UUID)
		}
		loaded[entry.subscription.ChecklistID] = entry
	}
	if err := finishDeltaRows(rows, "account delta subscription"); err != nil {
		return nil, nil, err
	}
	return loaded, revisionIDs, nil
}

func attachChecklistTrees(
	entry *loadedChecklist,
	trees map[uuid.UUID]shared.Revision,
) error {
	if entry.aggregate.DeletedAt != nil {
		return nil
	}
	if entry.draftID.Valid {
		draft, found := trees[entry.draftID.UUID]
		if !found {
			return fmt.Errorf("account delta draft revision tree disappeared")
		}
		entry.aggregate.Draft = &draft
	}
	if entry.publicationID.Valid {
		publication, found := trees[entry.publicationID.UUID]
		if !found {
			return fmt.Errorf(
				"account delta publication revision tree disappeared",
			)
		}
		entry.aggregate.Publication = &publication
	}
	return nil
}

func fitCompleteChanges(
	changes []AccountChange,
	after int64,
	accountVersion int64,
	rootTruncated bool,
	byteLimit int,
) ([]AccountChange, bool, error) {
	for index := range changes {
		prefix := changes[:index+1]
		hasMore := rootTruncated || index+1 < len(changes)
		candidate := &AccountDelta{
			FromCursor:     after,
			ThroughCursor:  prefix[len(prefix)-1].AccountChangeVersion,
			AccountVersion: accountVersion,
			HasMore:        hasMore,
			Changes:        prefix,
		}
		encoded, err := json.Marshal(accountDeltaEnvelope(candidate))
		if err != nil {
			return nil, false, fmt.Errorf("encode account delta response: %w", err)
		}
		if index > 0 && len(encoded) > byteLimit {
			return changes[:index], true, nil
		}
	}
	return changes, false, nil
}

func accountDeltaEnvelope(delta *AccountDelta) response.StandardResponse {
	return response.StandardResponse{
		Status:  http.StatusOK,
		Message: "",
		Data:    delta,
	}
}

func rootIDs(roots []deltaRoot, kind string) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(roots))
	for _, root := range roots {
		if root.kind == kind {
			ids = append(ids, root.identity)
		}
	}
	return ids
}

func uniqueUUIDs(ids []uuid.UUID) []uuid.UUID {
	seen := make(map[uuid.UUID]struct{}, len(ids))
	unique := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		unique = append(unique, id)
	}
	return unique
}

func nullableUUID(value uuid.NullUUID) *uuid.UUID {
	if !value.Valid {
		return nil
	}
	id := value.UUID
	return &id
}

func nullableTime(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	timestamp := value.Time
	return &timestamp
}

func finishDeltaRows(rows *sql.Rows, resource string) error {
	if err := rows.Err(); err != nil {
		return closeDeltaRows(rows, fmt.Errorf("iterate %s rows: %w", resource, err))
	}
	if err := rows.Close(); err != nil {
		return fmt.Errorf("close %s rows: %w", resource, err)
	}
	return nil
}

func closeDeltaRows(rows *sql.Rows, cause error) error {
	if err := rows.Close(); err != nil {
		return errors.Join(cause, fmt.Errorf("close account delta rows: %w", err))
	}
	return cause
}

func rollbackDelta(tx *sql.Tx, cause error) error {
	if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
		return errors.Join(
			cause,
			fmt.Errorf("rollback account delta snapshot: %w", err),
		)
	}
	return cause
}
