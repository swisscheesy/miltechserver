package subscriptions

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

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

type lockedSource struct {
	status            string
	currentRevisionID *uuid.UUID
	ownerUID          *string
	deletedAt         *time.Time
}
type lockedSubscription struct{ subscription shared.Subscription }

func (repository *RepositoryImpl) Install(ctx context.Context, subscriberUID string, checklistID uuid.UUID, precondition shared.Precondition) (*MutationResult, error) {
	return persistence.WithWriteTx(ctx, repository.store.DB, repository.store.MaxWriteAttempts, func(tx *sql.Tx) (*MutationResult, error) {
		if _, err := persistence.LockAccountVersion(ctx, tx, subscriberUID); err != nil {
			return nil, err
		}
		source, found, err := lockActiveSource(ctx, tx, checklistID)
		if err != nil {
			return nil, err
		}
		if !found {
			return nil, shared.NewResourceNotFound("community checklist not found", nil)
		}
		if source.deletedAt != nil || source.status != "active" || source.currentRevisionID == nil {
			return nil, shared.NewInvalidTransition("community checklist is not available for installation", nil)
		}
		if source.ownerUID != nil && *source.ownerUID == subscriberUID {
			return nil, shared.NewInvalidTransition("owners cannot subscribe to their own checklist", nil)
		}
		subscription, exists, err := lockSubscription(ctx, tx, subscriberUID, checklistID)
		if err != nil {
			return nil, err
		}
		if !exists {
			if precondition.Mode != shared.PreconditionCreate {
				return nil, staleSubscriptionError()
			}
			var activeCount int
			if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM user_pmcs_subscriptions WHERE subscriber_uid = $1 AND deleted_at IS NULL`, subscriberUID).Scan(&activeCount); err != nil {
				return nil, fmt.Errorf("count active subscriptions: %w", err)
			}
			if activeCount >= repository.config.MaxActiveSubscriptions {
				return nil, shared.NewInvalidTransition("active subscription limit reached", map[string]any{"limit": repository.config.MaxActiveSubscriptions})
			}
			accountVersion, err := persistence.AdvanceAccountVersion(ctx, tx, subscriberUID)
			if err != nil {
				return nil, err
			}
			if _, err := tx.ExecContext(ctx, `INSERT INTO user_pmcs_subscriptions (subscriber_uid, checklist_id, installed_revision_id, sync_version, account_change_version) VALUES ($1, $2, $3, 1, $4)`, subscriberUID, checklistID, *source.currentRevisionID, accountVersion); err != nil {
				return nil, fmt.Errorf("insert subscription: %w", err)
			}
			return loadMutation(ctx, tx, subscriberUID, checklistID, true, false)
		}
		current := subscription.subscription
		if current.DeletedAt != nil {
			if !precondition.Matches(shared.MakeSubscriptionETag(checklistID, current.SyncVersion)) {
				return nil, staleSubscriptionError()
			}
			accountVersion, err := persistence.AdvanceAccountVersion(ctx, tx, subscriberUID)
			if err != nil {
				return nil, err
			}
			if _, err := tx.ExecContext(ctx, `UPDATE user_pmcs_subscriptions SET installed_revision_id = $1, deleted_at = NULL, sync_version = sync_version + 1, account_change_version = $2, updated_at = now() WHERE subscriber_uid = $3 AND checklist_id = $4`, *source.currentRevisionID, accountVersion, subscriberUID, checklistID); err != nil {
				return nil, fmt.Errorf("resubscribe: %w", err)
			}
			return loadMutation(ctx, tx, subscriberUID, checklistID, false, false)
		}
		if current.InstalledRevisionID != nil && *current.InstalledRevisionID == *source.currentRevisionID {
			if precondition.Mode != shared.PreconditionCreate && !precondition.Matches(shared.MakeSubscriptionETag(checklistID, current.SyncVersion)) {
				return nil, staleSubscriptionError()
			}
			return loadMutation(ctx, tx, subscriberUID, checklistID, false, true)
		}
		return nil, staleSubscriptionError()
	})
}

func (repository *RepositoryImpl) Unsubscribe(ctx context.Context, subscriberUID string, checklistID uuid.UUID, precondition shared.Precondition) (*MutationResult, error) {
	return persistence.WithWriteTx(ctx, repository.store.DB, repository.store.MaxWriteAttempts, func(tx *sql.Tx) (*MutationResult, error) {
		if _, err := persistence.LockAccountVersion(ctx, tx, subscriberUID); err != nil {
			return nil, err
		}
		subscription, found, err := lockSubscription(ctx, tx, subscriberUID, checklistID)
		if err != nil {
			return nil, err
		}
		if !found {
			return nil, shared.NewResourceNotFound("subscription not found", nil)
		}
		if subscription.subscription.DeletedAt != nil {
			return loadMutation(ctx, tx, subscriberUID, checklistID, false, true)
		}
		if !precondition.Matches(shared.MakeSubscriptionETag(checklistID, subscription.subscription.SyncVersion)) {
			return nil, staleSubscriptionError()
		}
		accountVersion, err := persistence.AdvanceAccountVersion(ctx, tx, subscriberUID)
		if err != nil {
			return nil, err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE user_pmcs_subscriptions SET installed_revision_id = NULL, deleted_at = now(), sync_version = sync_version + 1, account_change_version = $1, updated_at = now() WHERE subscriber_uid = $2 AND checklist_id = $3`, accountVersion, subscriberUID, checklistID); err != nil {
			return nil, fmt.Errorf("unsubscribe: %w", err)
		}
		return loadMutation(ctx, tx, subscriberUID, checklistID, false, false)
	})
}

func (repository *RepositoryImpl) GetInstalledRelease(ctx context.Context, subscriberUID string, checklistID, revisionID uuid.UUID) (*shared.InstalledChecklistRelease, error) {
	return loadInstalledRelease(ctx, repository.store.DB, subscriberUID, checklistID, revisionID)
}

func lockActiveSource(ctx context.Context, tx *sql.Tx, checklistID uuid.UUID) (lockedSource, bool, error) {
	var source lockedSource
	err := tx.QueryRowContext(ctx, `SELECT owner_uid, deleted_at FROM user_pmcs_checklists WHERE id = $1 FOR UPDATE`, checklistID).Scan(&source.ownerUID, &source.deletedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return lockedSource{}, false, nil
	}
	if err != nil {
		return lockedSource{}, false, fmt.Errorf("lock source checklist: %w", err)
	}
	var current uuid.NullUUID
	err = tx.QueryRowContext(ctx, `SELECT status, current_release_revision_id FROM user_pmcs_community_sources WHERE checklist_id = $1 FOR UPDATE`, checklistID).Scan(&source.status, &current)
	if errors.Is(err, sql.ErrNoRows) {
		return lockedSource{}, false, nil
	}
	if err != nil {
		return lockedSource{}, false, fmt.Errorf("lock community source: %w", err)
	}
	if current.Valid {
		value := current.UUID
		source.currentRevisionID = &value
	}
	return source, true, nil
}

func lockSubscription(ctx context.Context, tx *sql.Tx, subscriberUID string, checklistID uuid.UUID) (lockedSubscription, bool, error) {
	var value lockedSubscription
	err := tx.QueryRowContext(ctx, `SELECT checklist_id, installed_revision_id, sync_version, account_change_version, created_at, updated_at, deleted_at FROM user_pmcs_subscriptions WHERE subscriber_uid = $1 AND checklist_id = $2 FOR UPDATE`, subscriberUID, checklistID).Scan(&value.subscription.ChecklistID, &value.subscription.InstalledRevisionID, &value.subscription.SyncVersion, &value.subscription.AccountChangeVersion, &value.subscription.CreatedAt, &value.subscription.UpdatedAt, &value.subscription.DeletedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return lockedSubscription{}, false, nil
	}
	if err != nil {
		return lockedSubscription{}, false, fmt.Errorf("lock subscription: %w", err)
	}
	return value, true, nil
}

func loadMutation(ctx context.Context, queryer persistence.Queryer, subscriberUID string, checklistID uuid.UUID, created, idempotent bool) (*MutationResult, error) {
	var subscription shared.Subscription
	err := queryer.QueryRowContext(ctx, `SELECT checklist_id, installed_revision_id, sync_version, account_change_version, created_at, updated_at, deleted_at FROM user_pmcs_subscriptions WHERE subscriber_uid = $1 AND checklist_id = $2`, subscriberUID, checklistID).Scan(&subscription.ChecklistID, &subscription.InstalledRevisionID, &subscription.SyncVersion, &subscription.AccountChangeVersion, &subscription.CreatedAt, &subscription.UpdatedAt, &subscription.DeletedAt)
	if err != nil {
		return nil, fmt.Errorf("load subscription: %w", err)
	}
	result := &MutationResult{Subscription: subscription, Created: created, Idempotent: idempotent}
	if subscription.InstalledRevisionID != nil {
		installed, err := loadInstalledRelease(ctx, queryer, subscriberUID, checklistID, *subscription.InstalledRevisionID)
		if err != nil {
			return nil, err
		}
		result.Installed = installed
	}
	return result, nil
}

func loadInstalledRelease(ctx context.Context, queryer persistence.Queryer, subscriberUID string, checklistID, revisionID uuid.UUID) (*shared.InstalledChecklistRelease, error) {
	var release shared.InstalledChecklistRelease
	var username sql.NullString
	err := queryer.QueryRowContext(ctx, `SELECT subscription.checklist_id, COALESCE(source.status, 'retired'), owner.username, community_release.released_at FROM user_pmcs_subscriptions AS subscription JOIN user_pmcs_community_releases AS community_release ON community_release.checklist_id = subscription.checklist_id AND community_release.revision_id = subscription.installed_revision_id LEFT JOIN user_pmcs_community_sources AS source ON source.checklist_id = subscription.checklist_id LEFT JOIN user_pmcs_checklists AS checklist ON checklist.id = subscription.checklist_id LEFT JOIN users AS owner ON owner.uid = checklist.owner_uid WHERE subscription.subscriber_uid = $1 AND subscription.checklist_id = $2 AND subscription.installed_revision_id = $3 AND subscription.deleted_at IS NULL`, subscriberUID, checklistID, revisionID).Scan(&release.ChecklistID, &release.SourceStatus, &username, &release.ReleasedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, shared.NewResourceNotFound("installed checklist release not found", nil)
	}
	if err != nil {
		return nil, fmt.Errorf("load installed release: %w", err)
	}
	if username.Valid {
		release.CreatorDisplayName = username.String
	} else {
		release.CreatorDisplayName = "Deleted user"
	}
	trees, err := persistence.LoadRevisionTrees(ctx, queryer, []uuid.UUID{revisionID})
	if err != nil {
		return nil, err
	}
	revision, found := trees[revisionID]
	if !found {
		return nil, fmt.Errorf("installed release revision disappeared")
	}
	release.Revision = revision
	return &release, nil
}

func staleSubscriptionError() *shared.APIError {
	return shared.NewStalePrecondition("subscription has changed", nil)
}
