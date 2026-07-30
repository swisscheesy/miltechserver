package owned

import (
	"bytes"
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
	tx, err := repository.store.DB.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelRepeatableRead,
		ReadOnly:  true,
	})
	if err != nil {
		return nil, fmt.Errorf("begin owned checklist snapshot: %w", err)
	}
	if err := requireInitializedAccount(
		ctx,
		tx,
		ownerUID,
	); err != nil {
		return nil, rollbackReadTransaction(tx, err)
	}
	aggregate, err := loadOwnedAggregate(ctx, tx, ownerUID, checklistID)
	if err != nil {
		return nil, rollbackReadTransaction(tx, err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit owned checklist snapshot: %w", err)
	}
	return aggregate, nil
}

func (repository *RepositoryImpl) GetRevision(
	ctx context.Context,
	ownerUID string,
	checklistID uuid.UUID,
	revisionID uuid.UUID,
) (*shared.Revision, error) {
	tx, err := repository.store.DB.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelRepeatableRead,
		ReadOnly:  true,
	})
	if err != nil {
		return nil, fmt.Errorf("begin owned revision snapshot: %w", err)
	}
	if err := requireInitializedAccount(ctx, tx, ownerUID); err != nil {
		return nil, rollbackReadTransaction(tx, err)
	}

	var authorizedRevisionID uuid.UUID
	err = tx.QueryRowContext(
		ctx,
		`SELECT revision.id
		 FROM user_pmcs_revisions AS revision
		 JOIN user_pmcs_checklists AS checklist
		   ON checklist.id = revision.checklist_id
		 WHERE checklist.id = $1
		   AND checklist.owner_uid = $2
		   AND checklist.deleted_at IS NULL
		   AND revision.id = $3
		   AND revision.state IN ('published', 'superseded')`,
		checklistID,
		ownerUID,
		revisionID,
	).Scan(&authorizedRevisionID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, rollbackReadTransaction(tx, hiddenChecklistError())
	}
	if err != nil {
		return nil, rollbackReadTransaction(
			tx,
			fmt.Errorf("authorize owned historical revision: %w", err),
		)
	}

	revisions, err := persistence.LoadRevisionTrees(
		ctx,
		tx,
		[]uuid.UUID{authorizedRevisionID},
	)
	if err != nil {
		return nil, rollbackReadTransaction(tx, err)
	}
	revision, found := revisions[authorizedRevisionID]
	if !found {
		return nil, rollbackReadTransaction(
			tx,
			fmt.Errorf("owned historical revision disappeared"),
		)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit owned revision snapshot: %w", err)
	}
	return &revision, nil
}

func rollbackReadTransaction(tx *sql.Tx, cause error) error {
	if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
		return errors.Join(cause, fmt.Errorf("rollback owned checklist snapshot: %w", err))
	}
	return cause
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
			if err := lockPreparedTreeUUIDs(
				ctx,
				tx,
				draft.Input,
			); err != nil {
				return nil, err
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
			if err := lockPreparedTreeUUIDs(
				ctx,
				tx,
				draft.Input,
			); err != nil {
				return nil, err
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

func (repository *RepositoryImpl) Publish(
	ctx context.Context,
	ownerUID string,
	checklistID uuid.UUID,
	revision shared.PreparedRevision,
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
			if revision.Input.RevisionNumber == nil ||
				*revision.Input.RevisionNumber <= 0 {
				return nil, shared.NewInvalidTransition(
					"revision_number must be positive",
					nil,
				)
			}

			idempotent, err := repository.resolvePublicationRetry(
				ctx,
				tx,
				ownerUID,
				checklist,
				revision,
			)
			if err != nil || idempotent != nil {
				return idempotent, err
			}

			submittedNumber := *revision.Input.RevisionNumber
			var nextNumber int32
			if err := tx.QueryRowContext(
				ctx,
				`SELECT COALESCE(MAX(revision_number), 0) + 1
				 FROM user_pmcs_revisions
				 WHERE checklist_id = $1`,
				checklistID,
			).Scan(&nextNumber); err != nil {
				return nil, fmt.Errorf("read next publication number: %w", err)
			}
			if submittedNumber != nextNumber {
				return nil, shared.NewInvalidTransition(
					"revision_number must be the exact next publication number",
					map[string]any{"expected_revision_number": nextNumber},
				)
			}
			if err := lockPreparedTreeUUIDs(
				ctx,
				tx,
				revision.Input,
			); err != nil {
				return nil, err
			}

			draftID, hasDraft, err := currentDraftID(ctx, tx, checklistID)
			if err != nil {
				return nil, err
			}
			if hasDraft && draftID == revision.Input.ID {
				var draftHash []byte
				if err := tx.QueryRowContext(
					ctx,
					`SELECT content_hash
					 FROM user_pmcs_revisions
					 WHERE id = $1
					   AND checklist_id = $2
					   AND state = 'draft'`,
					revision.Input.ID,
					checklistID,
				).Scan(&draftHash); err != nil {
					return nil, fmt.Errorf(
						"read submitted draft hash: %w",
						err,
					)
				}
				if !bytes.Equal(draftHash, revision.Hash[:]) {
					if err := persistence.ReplaceDraftTree(
						ctx,
						tx,
						checklistID,
						revision,
					); err != nil {
						return nil, err
					}
				}
			} else {
				if err := ensurePreparedTreeIDsAvailable(
					ctx,
					tx,
					revision.Input,
				); err != nil {
					return nil, err
				}
				if hasDraft {
					if _, err := tx.ExecContext(
						ctx,
						`DELETE FROM user_pmcs_revisions
						 WHERE id = $1
						   AND checklist_id = $2
						   AND state = 'draft'`,
						draftID,
						checklistID,
					); err != nil {
						return nil, fmt.Errorf(
							"delete replaced publication draft: %w",
							err,
						)
					}
				}
				if err := insertDraftRoot(
					ctx,
					tx,
					checklistID,
					revision,
				); err != nil {
					return nil, err
				}
				if err := persistence.ReplaceDraftTree(
					ctx,
					tx,
					checklistID,
					revision,
				); err != nil {
					return nil, err
				}
			}

			if _, err := tx.ExecContext(
				ctx,
				`UPDATE user_pmcs_revisions
				 SET state = 'superseded'
				 WHERE checklist_id = $1
				   AND state = 'published'
				   AND id <> $2`,
				checklistID,
				revision.Input.ID,
			); err != nil {
				return nil, fmt.Errorf(
					"supersede current publication: %w",
					err,
				)
			}

			var publishedID uuid.UUID
			err = tx.QueryRowContext(
				ctx,
				`UPDATE user_pmcs_revisions
				 SET state = 'published',
				     revision_number = $1,
				     updated_at = now(),
				     published_at = now()
				 WHERE id = $2
				   AND checklist_id = $3
				   AND state = 'draft'
				 RETURNING id`,
				submittedNumber,
				revision.Input.ID,
				checklistID,
			).Scan(&publishedID)
			if errors.Is(err, sql.ErrNoRows) {
				return nil, shared.NewInvalidTransition(
					"revision is not the checklist's mutable draft",
					nil,
				)
			}
			if err != nil {
				return nil, fmt.Errorf("promote draft publication: %w", err)
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

func (repository *RepositoryImpl) resolvePublicationRetry(
	ctx context.Context,
	tx *sql.Tx,
	ownerUID string,
	checklist lockedChecklist,
	revision shared.PreparedRevision,
) (*MutationResult, error) {
	var (
		storedChecklistID uuid.UUID
		state             string
		revisionNumber    sql.NullInt32
		name              string
		description       string
		contentHash       []byte
	)
	err := tx.QueryRowContext(
		ctx,
		`SELECT checklist_id, state, revision_number, name, description,
		        content_hash
		 FROM user_pmcs_revisions
		 WHERE id = $1`,
		revision.Input.ID,
	).Scan(
		&storedChecklistID,
		&state,
		&revisionNumber,
		&name,
		&description,
		&contentHash,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("inspect publication retry: %w", err)
	}
	if storedChecklistID != checklist.id {
		return nil, shared.NewValidationFailed(
			"revision tree UUID belongs to another resource",
			nil,
		)
	}
	if state == "draft" {
		return nil, nil
	}
	if state != "published" && state != "superseded" {
		return nil, shared.NewInvalidTransition(
			"revision cannot be published from its current state",
			nil,
		)
	}
	if !revisionNumber.Valid ||
		revision.Input.RevisionNumber == nil ||
		revisionNumber.Int32 != *revision.Input.RevisionNumber {
		return nil, shared.NewInvalidTransition(
			"revision_number conflicts with immutable publication history",
			nil,
		)
	}
	if name != revision.Input.Name ||
		description != revision.Input.Description ||
		!bytes.Equal(contentHash, revision.Hash[:]) {
		return nil, staleChecklistError()
	}

	aggregate, err := loadOwnedAggregate(
		ctx,
		tx,
		ownerUID,
		checklist.id,
	)
	if err != nil {
		return nil, err
	}
	return &MutationResult{
		Aggregate:  *aggregate,
		Idempotent: true,
	}, nil
}
