package owned

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"miltechserver/api/user_pmcs/persistence"
	"miltechserver/api/user_pmcs/shared"
)

type RepositoryImpl struct {
	store  persistence.Store
	config shared.Config
}

func NewRepository(
	store persistence.Store,
	config shared.Config,
) Repository {
	return &RepositoryImpl{store: store, config: config}
}

func (repository *RepositoryImpl) Get(
	ctx context.Context,
	ownerUID string,
	checklistID uuid.UUID,
) (*shared.ChecklistAggregate, error) {
	if err := requireInitializedAccount(
		ctx,
		repository.store.DB,
		ownerUID,
	); err != nil {
		return nil, err
	}
	return loadOwnedAggregate(ctx, repository.store.DB, ownerUID, checklistID)
}

func (repository *RepositoryImpl) Create(
	ctx context.Context,
	ownerUID string,
	checklistID uuid.UUID,
	draft shared.PreparedRevision,
	precondition shared.Precondition,
) (*MutationResult, error) {
	return persistence.WithWriteTx(
		ctx,
		repository.store.DB,
		repository.store.MaxWriteAttempts,
		func(tx *sql.Tx) (*MutationResult, error) {
			if _, err := persistence.LockAccountVersion(
				ctx,
				tx,
				ownerUID,
			); err != nil {
				return nil, err
			}

			existing, found, err := lockChecklist(ctx, tx, checklistID)
			if err != nil {
				return nil, err
			}
			if precondition.Mode != shared.PreconditionCreate {
				return nil, staleChecklistError()
			}
			if found {
				return repository.resolveCreateRetry(
					ctx,
					tx,
					ownerUID,
					existing,
					draft,
				)
			}
			if err := ensurePreparedTreeIDsAvailable(
				ctx,
				tx,
				draft.Input,
			); err != nil {
				return nil, err
			}

			var activeCount int
			if err := tx.QueryRowContext(
				ctx,
				`SELECT count(*)
				 FROM user_pmcs_checklists
				 WHERE owner_uid = $1 AND deleted_at IS NULL`,
				ownerUID,
			).Scan(&activeCount); err != nil {
				return nil, fmt.Errorf("count active owned checklists: %w", err)
			}
			if activeCount >= repository.config.MaxOwnedChecklists {
				return nil, shared.NewContentTooLarge(
					"active owned checklist limit reached",
					map[string]any{
						"limit": repository.config.MaxOwnedChecklists,
					},
				)
			}

			accountVersion, err := persistence.AdvanceAccountVersion(
				ctx,
				tx,
				ownerUID,
			)
			if err != nil {
				return nil, err
			}
			var insertedChecklistID uuid.UUID
			err = tx.QueryRowContext(
				ctx,
				`INSERT INTO user_pmcs_checklists
				     (id, owner_uid, sync_version, account_change_version)
				 VALUES ($1, $2, 1, $3)
				 ON CONFLICT (id) DO NOTHING
				 RETURNING id`,
				checklistID,
				ownerUID,
				accountVersion,
			).Scan(&insertedChecklistID)
			if errors.Is(err, sql.ErrNoRows) {
				concurrentChecklist, concurrentFound, lockErr := lockChecklist(
					ctx,
					tx,
					checklistID,
				)
				if lockErr != nil {
					return nil, lockErr
				}
				if !concurrentFound {
					return nil, fmt.Errorf(
						"owned checklist conflict disappeared",
					)
				}
				return repository.resolveCreateRetry(
					ctx,
					tx,
					ownerUID,
					concurrentChecklist,
					draft,
				)
			}
			if err != nil {
				return nil, fmt.Errorf("insert owned checklist: %w", err)
			}
			if err := insertDraftRoot(ctx, tx, checklistID, draft); err != nil {
				return nil, err
			}
			if err := persistence.ReplaceDraftTree(
				ctx,
				tx,
				checklistID,
				draft,
			); err != nil {
				return nil, err
			}
			aggregate, err := loadOwnedAggregate(
				ctx,
				tx,
				ownerUID,
				checklistID,
			)
			if err != nil {
				return nil, err
			}
			return &MutationResult{
				Aggregate: *aggregate,
				Created:   true,
			}, nil
		},
	)
}

func (repository *RepositoryImpl) PutDraft(
	ctx context.Context,
	ownerUID string,
	checklistID uuid.UUID,
	draft shared.PreparedRevision,
	precondition shared.Precondition,
) (*MutationResult, error) {
	return persistence.WithWriteTx(
		ctx,
		repository.store.DB,
		repository.store.MaxWriteAttempts,
		func(tx *sql.Tx) (*MutationResult, error) {
			if _, err := persistence.LockAccountVersion(
				ctx,
				tx,
				ownerUID,
			); err != nil {
				return nil, err
			}
			checklist, found, err := lockChecklist(ctx, tx, checklistID)
			if err != nil {
				return nil, err
			}
			if !found || checklist.ownerUID == nil ||
				*checklist.ownerUID != ownerUID {
				return nil, hiddenChecklistError()
			}
			if checklist.deletedAt != nil {
				return nil, staleChecklistError()
			}
			if !precondition.Matches(shared.MakeChecklistETag(
				checklistID,
				checklist.syncVersion,
			)) {
				return nil, staleChecklistError()
			}

			currentDraftID, hasDraft, err := currentDraftID(
				ctx,
				tx,
				checklistID,
			)
			if err != nil {
				return nil, err
			}
			preparedIDsChecked := false
			if hasDraft && currentDraftID != draft.Input.ID {
				if err := ensurePreparedTreeIDsAvailable(
					ctx,
					tx,
					draft.Input,
				); err != nil {
					return nil, err
				}
				preparedIDsChecked = true
				if _, err := tx.ExecContext(
					ctx,
					`DELETE FROM user_pmcs_revisions
					 WHERE id = $1 AND checklist_id = $2 AND state = 'draft'`,
					currentDraftID,
					checklistID,
				); err != nil {
					return nil, fmt.Errorf("delete replaced draft: %w", err)
				}
				hasDraft = false
			}
			if !hasDraft {
				if !preparedIDsChecked {
					if err := ensurePreparedTreeIDsAvailable(
						ctx,
						tx,
						draft.Input,
					); err != nil {
						return nil, err
					}
				}
				if err := insertDraftRoot(
					ctx,
					tx,
					checklistID,
					draft,
				); err != nil {
					return nil, err
				}
			}
			if err := persistence.ReplaceDraftTree(
				ctx,
				tx,
				checklistID,
				draft,
			); err != nil {
				return nil, err
			}
			if err := repository.advanceChecklist(
				ctx,
				tx,
				ownerUID,
				checklistID,
			); err != nil {
				return nil, err
			}
			aggregate, err := loadOwnedAggregate(
				ctx,
				tx,
				ownerUID,
				checklistID,
			)
			if err != nil {
				return nil, err
			}
			return &MutationResult{Aggregate: *aggregate}, nil
		},
	)
}

func (repository *RepositoryImpl) DeleteDraft(
	ctx context.Context,
	ownerUID string,
	checklistID uuid.UUID,
	revisionID uuid.UUID,
	precondition shared.Precondition,
) (*MutationResult, error) {
	return persistence.WithWriteTx(
		ctx,
		repository.store.DB,
		repository.store.MaxWriteAttempts,
		func(tx *sql.Tx) (*MutationResult, error) {
			if _, err := persistence.LockAccountVersion(
				ctx,
				tx,
				ownerUID,
			); err != nil {
				return nil, err
			}
			checklist, found, err := lockChecklist(ctx, tx, checklistID)
			if err != nil {
				return nil, err
			}
			if !found || checklist.ownerUID == nil ||
				*checklist.ownerUID != ownerUID {
				return nil, hiddenChecklistError()
			}
			if checklist.deletedAt != nil {
				return nil, staleChecklistError()
			}
			if !precondition.Matches(shared.MakeChecklistETag(
				checklistID,
				checklist.syncVersion,
			)) {
				return nil, staleChecklistError()
			}

			var (
				currentDraftID uuid.UUID
				hasPublication bool
			)
			err = tx.QueryRowContext(
				ctx,
				`SELECT
				     COALESCE(
				         (SELECT id FROM user_pmcs_revisions
				          WHERE checklist_id = $1 AND state = 'draft'),
				         '00000000-0000-0000-0000-000000000000'
				     ),
				     EXISTS (
				         SELECT 1 FROM user_pmcs_revisions
				         WHERE checklist_id = $1 AND state = 'published'
				     )`,
				checklistID,
			).Scan(&currentDraftID, &hasPublication)
			if err != nil {
				return nil, fmt.Errorf("inspect draft deletion state: %w", err)
			}
			if currentDraftID != revisionID || !hasPublication {
				return nil, shared.NewInvalidTransition(
					"draft can be deleted only when it is current and a publication exists",
					nil,
				)
			}
			if _, err := tx.ExecContext(
				ctx,
				`DELETE FROM user_pmcs_revisions
				 WHERE id = $1 AND checklist_id = $2 AND state = 'draft'`,
				revisionID,
				checklistID,
			); err != nil {
				return nil, fmt.Errorf("delete current draft: %w", err)
			}
			if err := repository.advanceChecklist(
				ctx,
				tx,
				ownerUID,
				checklistID,
			); err != nil {
				return nil, err
			}
			aggregate, err := loadOwnedAggregate(
				ctx,
				tx,
				ownerUID,
				checklistID,
			)
			if err != nil {
				return nil, err
			}
			return &MutationResult{Aggregate: *aggregate}, nil
		},
	)
}
