package owned

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/binary"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"miltechserver/api/user_pmcs/persistence"
	"miltechserver/api/user_pmcs/shared"
)

type Repository interface {
	Get(
		ctx context.Context,
		ownerUID string,
		checklistID uuid.UUID,
	) (*shared.ChecklistAggregate, error)
	Create(
		ctx context.Context,
		ownerUID string,
		checklistID uuid.UUID,
		draft shared.PreparedRevision,
		precondition shared.Precondition,
	) (*MutationResult, error)
	PutDraft(
		ctx context.Context,
		ownerUID string,
		checklistID uuid.UUID,
		draft shared.PreparedRevision,
		precondition shared.Precondition,
	) (*MutationResult, error)
	DeleteDraft(
		ctx context.Context,
		ownerUID string,
		checklistID uuid.UUID,
		revisionID uuid.UUID,
		precondition shared.Precondition,
	) (*MutationResult, error)
	Publish(
		ctx context.Context,
		ownerUID string,
		checklistID uuid.UUID,
		revision shared.PreparedRevision,
		precondition shared.Precondition,
	) (*MutationResult, error)
	GetRevision(
		ctx context.Context,
		ownerUID string,
		checklistID uuid.UUID,
		revisionID uuid.UUID,
	) (*HistoricalRevisionResult, error)
}

type MutationResult struct {
	Aggregate  shared.ChecklistAggregate
	Created    bool
	Idempotent bool
}

type HistoricalRevision struct {
	ID             uuid.UUID           `json:"id"`
	RevisionNumber *int32              `json:"revision_number,omitempty"`
	Name           string              `json:"name"`
	Description    string              `json:"description"`
	Models         []shared.ModelValue `json:"models"`
	Sections       []shared.Section    `json:"sections"`
	CreatedAt      time.Time           `json:"created_at"`
	UpdatedAt      time.Time           `json:"updated_at"`
	PublishedAt    *time.Time          `json:"published_at,omitempty"`
}

type HistoricalRevisionResult struct {
	Revision    HistoricalRevision
	ContentHash [sha256.Size]byte
}

type lockedChecklist struct {
	id          uuid.UUID
	ownerUID    *string
	syncVersion int64
	deletedAt   *time.Time
}

const (
	// The verified server baseline has max_locks_per_transaction=64. Capping
	// one submitted tree at half that value bounds shared lock-table pressure
	// and leaves capacity for relation locks and concurrent writers.
	preparedTreeAdvisoryStripeCount uint64 = 32

	// 0x55504d43 is ASCII "UPMC"; the low bits identify a UUID stripe.
	preparedTreeAdvisoryNamespace int64 = 0x55504d4300000000
)

func lockChecklist(
	ctx context.Context,
	tx *sql.Tx,
	checklistID uuid.UUID,
) (lockedChecklist, bool, error) {
	var checklist lockedChecklist
	err := tx.QueryRowContext(
		ctx,
		`SELECT id, owner_uid, sync_version, deleted_at
		 FROM user_pmcs_checklists
		 WHERE id = $1
		 FOR UPDATE`,
		checklistID,
	).Scan(
		&checklist.id,
		&checklist.ownerUID,
		&checklist.syncVersion,
		&checklist.deletedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return lockedChecklist{}, false, nil
	}
	if err != nil {
		return lockedChecklist{}, false, fmt.Errorf("lock owned checklist: %w", err)
	}
	return checklist, true, nil
}

func (repository *RepositoryImpl) resolveCreateRetry(
	ctx context.Context,
	tx *sql.Tx,
	ownerUID string,
	checklist lockedChecklist,
	draft shared.PreparedRevision,
) (*MutationResult, error) {
	if checklist.ownerUID == nil || *checklist.ownerUID != ownerUID {
		return nil, hiddenChecklistError()
	}
	if checklist.deletedAt != nil {
		return nil, staleChecklistError()
	}

	var (
		revisionID  uuid.UUID
		name        string
		description string
		contentHash []byte
		total       int
	)
	err := tx.QueryRowContext(
		ctx,
		`SELECT id, name, description, content_hash,
		        count(*) OVER ()
		 FROM user_pmcs_revisions
		 WHERE checklist_id = $1
		 ORDER BY id
		 LIMIT 1`,
		checklist.id,
	).Scan(&revisionID, &name, &description, &contentHash, &total)
	if err == nil &&
		total == 1 &&
		revisionID == draft.Input.ID &&
		name == draft.Input.Name &&
		description == draft.Input.Description &&
		bytes.Equal(contentHash, draft.Hash[:]) {
		aggregate, loadErr := loadOwnedAggregate(
			ctx,
			tx,
			ownerUID,
			checklist.id,
		)
		if loadErr != nil {
			return nil, loadErr
		}
		if aggregate.Draft != nil &&
			aggregate.Publication == nil &&
			aggregate.Community == nil {
			return &MutationResult{
				Aggregate:  *aggregate,
				Idempotent: true,
			}, nil
		}
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("inspect checklist create retry: %w", err)
	}
	return nil, staleChecklistError()
}

func insertDraftRoot(
	ctx context.Context,
	tx *sql.Tx,
	checklistID uuid.UUID,
	draft shared.PreparedRevision,
) error {
	_, err := tx.ExecContext(
		ctx,
		`INSERT INTO user_pmcs_revisions
		     (id, checklist_id, state, revision_number, name, description,
		      content_hash, published_at)
		 VALUES ($1, $2, 'draft', NULL, $3, $4, $5, NULL)`,
		draft.Input.ID,
		checklistID,
		draft.Input.Name,
		draft.Input.Description,
		draft.Hash[:],
	)
	if err != nil {
		return fmt.Errorf("insert draft revision: %w", err)
	}
	return nil
}

func ensurePreparedTreeIDsAvailable(
	ctx context.Context,
	tx *sql.Tx,
	revision shared.RevisionInput,
) error {
	ids := []uuid.UUID{revision.ID}
	for _, section := range revision.Sections {
		ids = append(ids, section.ID)
		for _, item := range section.Items {
			ids = append(ids, item.ID)
			for _, notice := range item.Notices {
				ids = append(ids, notice.ID)
			}
			for _, step := range item.ProcedureSteps {
				ids = append(ids, step.ID)
			}
		}
	}

	var exists bool
	if err := tx.QueryRowContext(
		ctx,
		`SELECT EXISTS (
		     SELECT 1 FROM user_pmcs_revisions WHERE id = ANY($1)
		     UNION ALL
		     SELECT 1 FROM user_pmcs_sections WHERE id = ANY($1)
		     UNION ALL
		     SELECT 1 FROM user_pmcs_items WHERE id = ANY($1)
		     UNION ALL
		     SELECT 1 FROM user_pmcs_notices WHERE id = ANY($1)
		     UNION ALL
		     SELECT 1 FROM user_pmcs_procedure_steps WHERE id = ANY($1)
		 )`,
		pq.Array(ids),
	).Scan(&exists); err != nil {
		return fmt.Errorf("inspect submitted UUID ownership: %w", err)
	}
	if exists {
		return shared.NewValidationFailed(
			"revision tree UUID belongs to another resource",
			nil,
		)
	}
	return nil
}

func preparedTreeAdvisoryKeys(revision shared.RevisionInput) []int64 {
	ids := []uuid.UUID{revision.ID}
	for _, section := range revision.Sections {
		ids = append(ids, section.ID)
		for _, item := range section.Items {
			ids = append(ids, item.ID)
			for _, notice := range item.Notices {
				ids = append(ids, notice.ID)
			}
			for _, step := range item.ProcedureSteps {
				ids = append(ids, step.ID)
			}
		}
	}

	keys := make([]int64, 0, len(ids))
	for _, id := range ids {
		digest := sha256.Sum256(id[:])
		stripe := int64(
			binary.BigEndian.Uint64(digest[:8]) %
				preparedTreeAdvisoryStripeCount,
		)
		keys = append(keys, preparedTreeAdvisoryNamespace+stripe)
	}
	// Identical UUIDs always share a stripe. Stripe collisions only add
	// contention, and sorted acquisition prevents overlapping trees from
	// deadlocking.
	sort.Slice(keys, func(left, right int) bool {
		return keys[left] < keys[right]
	})

	deduplicatedKeys := keys[:0]
	for index, key := range keys {
		if index == 0 || key != keys[index-1] {
			deduplicatedKeys = append(deduplicatedKeys, key)
		}
	}
	return deduplicatedKeys
}

func lockPreparedTreeUUIDs(
	ctx context.Context,
	tx *sql.Tx,
	revision shared.RevisionInput,
) error {
	keys := preparedTreeAdvisoryKeys(revision)
	if len(keys) == 0 {
		return nil
	}

	var acquired int
	if err := tx.QueryRowContext(
		ctx,
		`WITH RECURSIVE acquired(position, lock_result) AS (
		     SELECT 1, pg_advisory_xact_lock(($1::bigint[])[1])
		     UNION ALL
		     SELECT position + 1,
		            pg_advisory_xact_lock(($1::bigint[])[position + 1])
		     FROM acquired
		     WHERE position < cardinality($1::bigint[])
		 )
		 SELECT count(*) FROM acquired`,
		pq.Array(keys),
	).Scan(&acquired); err != nil {
		return fmt.Errorf("lock submitted revision tree UUIDs: %w", err)
	}
	if acquired != len(keys) {
		return fmt.Errorf(
			"lock submitted revision tree UUIDs: acquired %d of %d locks",
			acquired,
			len(keys),
		)
	}
	return nil
}

func requireInitializedAccount(
	ctx context.Context,
	queryer persistence.Queryer,
	ownerUID string,
) error {
	var exists bool
	if err := queryer.QueryRowContext(
		ctx,
		`SELECT EXISTS (SELECT 1 FROM users WHERE uid = $1)`,
		ownerUID,
	).Scan(&exists); err != nil {
		return fmt.Errorf("verify user account: %w", err)
	}
	if !exists {
		return shared.NewAccountNotInitialized(
			"account is not initialized",
			nil,
		)
	}
	return nil
}

func currentDraftID(
	ctx context.Context,
	tx *sql.Tx,
	checklistID uuid.UUID,
) (uuid.UUID, bool, error) {
	var revisionID uuid.UUID
	err := tx.QueryRowContext(
		ctx,
		`SELECT id
		 FROM user_pmcs_revisions
		 WHERE checklist_id = $1 AND state = 'draft'`,
		checklistID,
	).Scan(&revisionID)
	if errors.Is(err, sql.ErrNoRows) {
		return uuid.Nil, false, nil
	}
	if err != nil {
		return uuid.Nil, false, fmt.Errorf("read current draft: %w", err)
	}
	return revisionID, true, nil
}

func (repository *RepositoryImpl) advanceChecklist(
	ctx context.Context,
	tx *sql.Tx,
	ownerUID string,
	checklistID uuid.UUID,
) error {
	accountVersion, err := persistence.AdvanceAccountVersion(
		ctx,
		tx,
		ownerUID,
	)
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(
		ctx,
		`UPDATE user_pmcs_checklists
		 SET sync_version = sync_version + 1,
		     account_change_version = $1,
		     updated_at = now()
		 WHERE id = $2`,
		accountVersion,
		checklistID,
	); err != nil {
		return fmt.Errorf("advance owned checklist version: %w", err)
	}
	return nil
}

func loadOwnedAggregate(
	ctx context.Context,
	queryer persistence.Queryer,
	ownerUID string,
	checklistID uuid.UUID,
) (*shared.ChecklistAggregate, error) {
	var aggregate shared.ChecklistAggregate
	err := queryer.QueryRowContext(
		ctx,
		`SELECT id, sync_version, account_change_version,
		        created_at, updated_at, deleted_at
		 FROM user_pmcs_checklists
		 WHERE id = $1 AND owner_uid = $2`,
		checklistID,
		ownerUID,
	).Scan(
		&aggregate.ID,
		&aggregate.SyncVersion,
		&aggregate.AccountChangeVersion,
		&aggregate.CreatedAt,
		&aggregate.UpdatedAt,
		&aggregate.DeletedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, hiddenChecklistError()
	}
	if err != nil {
		return nil, fmt.Errorf("read owned checklist root: %w", err)
	}
	if aggregate.DeletedAt != nil {
		return &aggregate, nil
	}

	rows, err := queryer.QueryContext(
		ctx,
		`SELECT id, state
		 FROM user_pmcs_revisions
		 WHERE checklist_id = $1 AND state IN ('draft', 'published')
		 ORDER BY state, id`,
		checklistID,
	)
	if err != nil {
		return nil, fmt.Errorf("query current owned revisions: %w", err)
	}
	var revisionIDs []uuid.UUID
	revisionStates := make(map[uuid.UUID]string)
	for rows.Next() {
		var (
			revisionID uuid.UUID
			state      string
		)
		if err := rows.Scan(&revisionID, &state); err != nil {
			_ = rows.Close()
			return nil, fmt.Errorf("scan current owned revision: %w", err)
		}
		revisionIDs = append(revisionIDs, revisionID)
		revisionStates[revisionID] = state
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, fmt.Errorf("iterate current owned revisions: %w", err)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("close current owned revisions: %w", err)
	}

	revisions, err := persistence.LoadRevisionTrees(
		ctx,
		queryer,
		revisionIDs,
	)
	if err != nil {
		return nil, err
	}
	for revisionID, state := range revisionStates {
		revision := revisions[revisionID]
		if state == "draft" {
			aggregate.Draft = &revision
		} else {
			aggregate.Publication = &revision
		}
	}
	if err := loadCommunitySummary(ctx, queryer, checklistID, &aggregate); err != nil {
		return nil, err
	}
	return &aggregate, nil
}

func loadCommunitySummary(
	ctx context.Context,
	queryer persistence.Queryer,
	checklistID uuid.UUID,
	aggregate *shared.ChecklistAggregate,
) error {
	var (
		summary        shared.CommunitySourceSummary
		currentRelease uuid.NullUUID
		retiredAt      sql.NullTime
	)
	err := queryer.QueryRowContext(
		ctx,
		`SELECT status, current_release_revision_id,
		        latest_release_revision_number, first_released_at,
		        updated_at, retired_at
		 FROM user_pmcs_community_sources
		 WHERE checklist_id = $1`,
		checklistID,
	).Scan(
		&summary.Status,
		&currentRelease,
		&summary.LatestReleaseRevisionNumber,
		&summary.FirstReleasedAt,
		&summary.UpdatedAt,
		&retiredAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read owned community summary: %w", err)
	}
	if currentRelease.Valid {
		value := currentRelease.UUID
		summary.CurrentReleaseRevisionID = &value
	}
	if retiredAt.Valid {
		value := retiredAt.Time
		summary.RetiredAt = &value
	}
	aggregate.Community = &summary
	return nil
}

func hiddenChecklistError() *shared.APIError {
	return shared.NewResourceNotFound("checklist not found", nil)
}

func staleChecklistError() *shared.APIError {
	return shared.NewStalePrecondition("resource changed", nil)
}
