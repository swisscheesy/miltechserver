package community

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

func NewRepository(store persistence.Store, config shared.Config) Repository {
	return &RepositoryImpl{store: store, config: config}
}

func (repository *RepositoryImpl) Release(
	ctx context.Context,
	ownerUID string,
	checklistID uuid.UUID,
	revisionID uuid.UUID,
	precondition shared.Precondition,
) (*ReleaseMutationResult, error) {
	return persistence.WithWriteTx(
		ctx,
		repository.store.DB,
		repository.store.MaxWriteAttempts,
		func(tx *sql.Tx) (*ReleaseMutationResult, error) {
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

			source, sourceExists, err := lockSource(ctx, tx, checklistID)
			if err != nil {
				return nil, err
			}
			revision, revisionExists, err := lockRevision(
				ctx,
				tx,
				checklistID,
				revisionID,
			)
			if err != nil {
				return nil, err
			}
			if !revisionExists ||
				(revision.state != "published" &&
					revision.state != "superseded") ||
				revision.revisionNumber == nil {
				return nil, invalidReleaseTransition(
					"community release requires a published revision",
				)
			}
			if err := validateImmutableRevision(
				ctx,
				tx,
				revisionID,
				revision,
				repository.config,
			); err != nil {
				return nil, err
			}

			if sourceExists &&
				source.status == "active" &&
				source.currentRevisionID != nil &&
				*source.currentRevisionID == revisionID {
				aggregate, err := loadAggregate(
					ctx,
					tx,
					ownerUID,
					checklistID,
				)
				if err != nil {
					return nil, err
				}
				return &ReleaseMutationResult{
					Aggregate:  *aggregate,
					Idempotent: true,
				}, nil
			}
			if !precondition.Matches(shared.MakeChecklistETag(
				checklistID,
				checklist.syncVersion,
			)) {
				return nil, staleChecklistError()
			}
			if sourceExists &&
				*revision.revisionNumber <= source.latestRevisionNumber {
				return nil, invalidReleaseTransition(
					"community release revision number must increase",
				)
			}

			if _, err := tx.ExecContext(
				ctx,
				`INSERT INTO user_pmcs_community_releases
				     (revision_id, checklist_id)
				 VALUES ($1, $2)`,
				revisionID,
				checklistID,
			); err != nil {
				return nil, fmt.Errorf("insert community release: %w", err)
			}
			if sourceExists {
				if _, err := tx.ExecContext(
					ctx,
					`UPDATE user_pmcs_community_sources
					 SET current_release_revision_id = $1,
					     status = 'active',
					     latest_release_revision_number = $2,
					     updated_at = now(),
					     retired_at = NULL
					 WHERE checklist_id = $3`,
					revisionID,
					*revision.revisionNumber,
					checklistID,
				); err != nil {
					return nil, fmt.Errorf("advance community source: %w", err)
				}
			} else if _, err := tx.ExecContext(
				ctx,
				`INSERT INTO user_pmcs_community_sources
				     (checklist_id, status, current_release_revision_id,
				      latest_release_revision_number, first_released_at,
				      updated_at, retired_at)
				 VALUES ($1, 'active', $2, $3, now(), now(), NULL)`,
				checklistID,
				revisionID,
				*revision.revisionNumber,
			); err != nil {
				return nil, fmt.Errorf("insert community source: %w", err)
			}

			if err := advanceChecklist(
				ctx,
				tx,
				ownerUID,
				checklistID,
			); err != nil {
				return nil, err
			}
			aggregate, err := loadAggregate(
				ctx,
				tx,
				ownerUID,
				checklistID,
			)
			if err != nil {
				return nil, err
			}
			return &ReleaseMutationResult{Aggregate: *aggregate}, nil
		},
	)
}

func (repository *RepositoryImpl) Retire(
	ctx context.Context,
	ownerUID string,
	checklistID uuid.UUID,
	precondition shared.Precondition,
) (*ReleaseMutationResult, error) {
	return persistence.WithWriteTx(
		ctx,
		repository.store.DB,
		repository.store.MaxWriteAttempts,
		func(tx *sql.Tx) (*ReleaseMutationResult, error) {
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
			source, sourceExists, err := lockSource(ctx, tx, checklistID)
			if err != nil {
				return nil, err
			}
			if !sourceExists {
				return nil, invalidReleaseTransition(
					"community source has not been released",
				)
			}
			if source.status == "retired" {
				aggregate, err := loadAggregate(
					ctx,
					tx,
					ownerUID,
					checklistID,
				)
				if err != nil {
					return nil, err
				}
				return &ReleaseMutationResult{
					Aggregate:  *aggregate,
					Idempotent: true,
				}, nil
			}
			if !precondition.Matches(shared.MakeChecklistETag(
				checklistID,
				checklist.syncVersion,
			)) {
				return nil, staleChecklistError()
			}
			if _, err := tx.ExecContext(
				ctx,
				`UPDATE user_pmcs_community_sources
				 SET current_release_revision_id = NULL,
				     status = 'retired',
				     updated_at = now(),
				     retired_at = now()
				 WHERE checklist_id = $1`,
				checklistID,
			); err != nil {
				return nil, fmt.Errorf("retire community source: %w", err)
			}
			if err := advanceChecklist(
				ctx,
				tx,
				ownerUID,
				checklistID,
			); err != nil {
				return nil, err
			}
			aggregate, err := loadAggregate(
				ctx,
				tx,
				ownerUID,
				checklistID,
			)
			if err != nil {
				return nil, err
			}
			return &ReleaseMutationResult{Aggregate: *aggregate}, nil
		},
	)
}

func advanceChecklist(
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
		return fmt.Errorf("advance released checklist: %w", err)
	}
	return nil
}

func loadAggregate(
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
		return nil, fmt.Errorf("load released checklist root: %w", err)
	}
	if aggregate.DeletedAt != nil {
		return &aggregate, nil
	}

	if err := loadCurrentRevisions(
		ctx,
		queryer,
		checklistID,
		&aggregate,
	); err != nil {
		return nil, err
	}
	if err := loadSourceSummary(
		ctx,
		queryer,
		checklistID,
		&aggregate,
	); err != nil {
		return nil, err
	}
	return &aggregate, nil
}
