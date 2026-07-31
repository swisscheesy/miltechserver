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
		source, sourceFound, err := lockActiveSource(ctx, tx, checklistID)
		if err != nil {
			return nil, err
		}
		finalizerCandidate := sourceFound &&
			source.ownerUID == nil &&
			source.deletedAt != nil &&
			source.status == "retired" &&
			source.currentRevisionID == nil
		var (
			subscription  lockedSubscription
			subscriptions []checklistSubscriptionLock
			found         bool
		)
		if finalizerCandidate {
			subscriptions, err = lockFinalizerSubscriptions(
				ctx,
				tx,
				checklistID,
				subscriberUID,
			)
			if err != nil {
				return nil, err
			}
			subscription, found = findLockedSubscription(
				subscriptions,
				subscriberUID,
			)
		} else {
			subscription, found, err = lockSubscription(
				ctx,
				tx,
				subscriberUID,
				checklistID,
			)
			if err != nil {
				return nil, err
			}
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
		finalizeRetainedContent := finalizerCandidate &&
			!hasOtherActivePin(subscriptions, subscriberUID)
		if finalizeRetainedContent {
			if err := lockRetainedContentDependencies(
				ctx,
				tx,
				checklistID,
			); err != nil {
				return nil, err
			}
		}
		accountVersion, err := persistence.AdvanceAccountVersion(ctx, tx, subscriberUID)
		if err != nil {
			return nil, err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE user_pmcs_subscriptions SET installed_revision_id = NULL, deleted_at = now(), sync_version = sync_version + 1, account_change_version = $1, updated_at = now() WHERE subscriber_uid = $2 AND checklist_id = $3`, accountVersion, subscriberUID, checklistID); err != nil {
			return nil, fmt.Errorf("unsubscribe: %w", err)
		}
		if finalizeRetainedContent {
			if err := deleteFinalRetainedContent(
				ctx,
				tx,
				checklistID,
			); err != nil {
				return nil, err
			}
		}
		return loadMutation(ctx, tx, subscriberUID, checklistID, false, false)
	})
}

func (repository *RepositoryImpl) GetInstalledRelease(ctx context.Context, subscriberUID string, checklistID, revisionID uuid.UUID) (*shared.InstalledChecklistRelease, error) {
	startedAt := time.Now()
	defer func() {
		shared.RecordDBDuration(ctx, time.Since(startedAt))
	}()
	return loadInstalledRelease(ctx, repository.store.DB, subscriberUID, checklistID, revisionID)
}

func (repository *RepositoryImpl) ListUpdates(ctx context.Context, subscriberUID string, after *uuid.UUID, limit int) (*shared.SubscriptionUpdatePage, error) {
	startedAt := time.Now()
	defer func() {
		shared.RecordDBDuration(ctx, time.Since(startedAt))
	}()

	arguments := []any{subscriberUID}
	query := `SELECT subscription.checklist_id,
	                 COALESCE(source.status, 'retired'),
	                 subscription.installed_revision_id,
	                 installed.revision_number,
	                 source.current_release_revision_id,
	                 current_release.revision_number
	          FROM user_pmcs_subscriptions AS subscription
	          JOIN user_pmcs_revisions AS installed
	            ON installed.checklist_id = subscription.checklist_id
	           AND installed.id = subscription.installed_revision_id
	          LEFT JOIN user_pmcs_community_sources AS source
	            ON source.checklist_id = subscription.checklist_id
	          LEFT JOIN user_pmcs_revisions AS current_release
	            ON current_release.checklist_id = source.checklist_id
	           AND current_release.id = source.current_release_revision_id
	          WHERE subscription.subscriber_uid = $1
	            AND subscription.deleted_at IS NULL`
	if after != nil {
		arguments = append(arguments, *after)
		query += fmt.Sprintf(" AND subscription.checklist_id > $%d", len(arguments))
	}
	arguments = append(arguments, limit+1)
	query += fmt.Sprintf(" ORDER BY subscription.checklist_id LIMIT $%d", len(arguments))

	rows, err := repository.store.DB.QueryContext(ctx, query, arguments...)
	if err != nil {
		return nil, fmt.Errorf("list subscription updates: %w", err)
	}
	defer rows.Close()
	items := make([]shared.SubscriptionUpdate, 0, limit+1)
	for rows.Next() {
		var (
			item          shared.SubscriptionUpdate
			currentID     uuid.NullUUID
			currentNumber sql.NullInt32
		)
		if err := rows.Scan(&item.ChecklistID, &item.SourceStatus, &item.InstalledRevisionID, &item.InstalledRevisionNumber, &currentID, &currentNumber); err != nil {
			return nil, fmt.Errorf("scan subscription update: %w", err)
		}
		if currentID.Valid && currentNumber.Valid {
			currentRevisionID := currentID.UUID
			currentReleaseNumber := currentNumber.Int32
			item.CurrentReleaseRevisionID = &currentRevisionID
			item.CurrentReleaseNumber = &currentReleaseNumber
			item.UpdateAvailable = item.SourceStatus == "active" && currentReleaseNumber > item.InstalledRevisionNumber
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate subscription updates: %w", err)
	}
	hasMore := len(items) > limit
	if hasMore {
		items = items[:limit]
	}
	page := &shared.SubscriptionUpdatePage{HasMore: hasMore, Items: items}
	if hasMore {
		cursor, err := shared.EncodeSubscriptionUpdateCursor(shared.SubscriptionUpdateCursor{Version: 1, Checklist: items[len(items)-1].ChecklistID})
		if err != nil {
			return nil, fmt.Errorf("encode subscription update cursor: %w", err)
		}
		page.NextCursor = &cursor
	}
	return page, nil
}

func (repository *RepositoryImpl) AcceptUpdate(ctx context.Context, subscriberUID string, checklistID, revisionID uuid.UUID, precondition shared.Precondition) (*MutationResult, error) {
	return persistence.WithWriteTx(ctx, repository.store.DB, repository.store.MaxWriteAttempts, func(tx *sql.Tx) (*MutationResult, error) {
		if _, err := persistence.LockAccountVersion(ctx, tx, subscriberUID); err != nil {
			return nil, err
		}
		source, found, err := lockActiveSource(ctx, tx, checklistID)
		if err != nil {
			return nil, err
		}
		if !found || source.deletedAt != nil || source.status != "active" || source.currentRevisionID == nil {
			return nil, shared.NewInvalidTransition("community checklist is not available for update", nil)
		}
		subscription, found, err := lockSubscription(ctx, tx, subscriberUID, checklistID)
		if err != nil {
			return nil, err
		}
		if !found || subscription.subscription.DeletedAt != nil || subscription.subscription.InstalledRevisionID == nil {
			return nil, shared.NewResourceNotFound("subscription not found", nil)
		}
		if *source.currentRevisionID != revisionID {
			return nil, shared.NewInvalidTransition("subscription update must target the current community release", nil)
		}
		if *subscription.subscription.InstalledRevisionID == revisionID {
			return loadMutation(ctx, tx, subscriberUID, checklistID, false, true)
		}
		if !precondition.Matches(shared.MakeSubscriptionETag(checklistID, subscription.subscription.SyncVersion)) {
			return nil, staleSubscriptionError()
		}
		installedNumber, currentNumber, err := revisionNumbers(ctx, tx, checklistID, *subscription.subscription.InstalledRevisionID, revisionID)
		if err != nil {
			return nil, err
		}
		if currentNumber <= installedNumber {
			return nil, shared.NewInvalidTransition("subscription update must advance release", nil)
		}
		accountVersion, err := persistence.AdvanceAccountVersion(ctx, tx, subscriberUID)
		if err != nil {
			return nil, err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE user_pmcs_subscriptions
			SET installed_revision_id = $1, sync_version = sync_version + 1,
			    account_change_version = $2, updated_at = now()
			WHERE subscriber_uid = $3 AND checklist_id = $4`, revisionID, accountVersion, subscriberUID, checklistID); err != nil {
			return nil, fmt.Errorf("accept subscription update: %w", err)
		}
		return loadMutation(ctx, tx, subscriberUID, checklistID, false, false)
	})
}

func revisionNumbers(ctx context.Context, tx *sql.Tx, checklistID, installedRevisionID, currentRevisionID uuid.UUID) (int32, int32, error) {
	var installedNumber, currentNumber int32
	err := tx.QueryRowContext(ctx, `SELECT installed.revision_number, current_release.revision_number
		FROM user_pmcs_revisions AS installed
		JOIN user_pmcs_revisions AS current_release ON current_release.checklist_id = installed.checklist_id
		WHERE installed.checklist_id = $1 AND installed.id = $2 AND current_release.id = $3`, checklistID, installedRevisionID, currentRevisionID).Scan(&installedNumber, &currentNumber)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, 0, shared.NewInvalidTransition("subscription update release is unavailable", nil)
	}
	if err != nil {
		return 0, 0, fmt.Errorf("read subscription update revision numbers: %w", err)
	}
	return installedNumber, currentNumber, nil
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

type checklistSubscriptionLock struct {
	subscriberUID string
	lockedSubscription
}

func lockFinalizerSubscriptions(
	ctx context.Context,
	tx *sql.Tx,
	checklistID uuid.UUID,
	subscriberUID string,
) ([]checklistSubscriptionLock, error) {
	rows, err := tx.QueryContext(
		ctx,
		`SELECT subscriber_uid, checklist_id, installed_revision_id,
		        sync_version, account_change_version, created_at, updated_at,
		        deleted_at
		   FROM user_pmcs_subscriptions
		  WHERE checklist_id = $1
		    AND (
		           subscriber_uid = $2
		        OR deleted_at IS NULL
		    )
		  ORDER BY checklist_id, subscriber_uid
		  FOR UPDATE`,
		checklistID,
		subscriberUID,
	)
	if err != nil {
		return nil, fmt.Errorf("lock finalizer subscriptions: %w", err)
	}
	defer rows.Close()

	subscriptions := make([]checklistSubscriptionLock, 0)
	for rows.Next() {
		var value checklistSubscriptionLock
		if err := rows.Scan(
			&value.subscriberUID,
			&value.subscription.ChecklistID,
			&value.subscription.InstalledRevisionID,
			&value.subscription.SyncVersion,
			&value.subscription.AccountChangeVersion,
			&value.subscription.CreatedAt,
			&value.subscription.UpdatedAt,
			&value.subscription.DeletedAt,
		); err != nil {
			return nil, fmt.Errorf("scan checklist subscription lock: %w", err)
		}
		subscriptions = append(subscriptions, value)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate checklist subscription locks: %w", err)
	}
	return subscriptions, nil
}

func findLockedSubscription(
	subscriptions []checklistSubscriptionLock,
	subscriberUID string,
) (lockedSubscription, bool) {
	for _, subscription := range subscriptions {
		if subscription.subscriberUID == subscriberUID {
			return subscription.lockedSubscription, true
		}
	}
	return lockedSubscription{}, false
}

func hasOtherActivePin(
	subscriptions []checklistSubscriptionLock,
	unsubscribingUID string,
) bool {
	for _, subscription := range subscriptions {
		if subscription.subscriberUID != unsubscribingUID &&
			subscription.subscription.DeletedAt == nil &&
			subscription.subscription.InstalledRevisionID != nil {
			return true
		}
	}
	return false
}

func lockRetainedContentDependencies(
	ctx context.Context,
	tx *sql.Tx,
	checklistID uuid.UUID,
) error {
	for _, query := range []string{
		`SELECT id
		   FROM user_pmcs_revisions
		  WHERE checklist_id = $1
		  ORDER BY checklist_id, id
		  FOR UPDATE`,
		`SELECT revision_id
		   FROM user_pmcs_community_releases
		  WHERE checklist_id = $1
		  ORDER BY checklist_id, revision_id
		  FOR UPDATE`,
	} {
		rows, err := tx.QueryContext(ctx, query, checklistID)
		if err != nil {
			return fmt.Errorf("lock retained content dependencies: %w", err)
		}
		for rows.Next() {
			var ignored uuid.UUID
			if err := rows.Scan(&ignored); err != nil {
				_ = rows.Close()
				return fmt.Errorf(
					"scan retained content dependency lock: %w",
					err,
				)
			}
		}
		if err := rows.Err(); err != nil {
			_ = rows.Close()
			return fmt.Errorf(
				"iterate retained content dependency locks: %w",
				err,
			)
		}
		if err := rows.Close(); err != nil {
			return fmt.Errorf(
				"close retained content dependency locks: %w",
				err,
			)
		}
	}
	return nil
}

func deleteFinalRetainedContent(
	ctx context.Context,
	tx *sql.Tx,
	checklistID uuid.UUID,
) error {
	result, err := tx.ExecContext(
		ctx,
		`DELETE FROM user_pmcs_community_sources AS source
		  USING user_pmcs_checklists AS checklist
		  WHERE source.checklist_id = $1
		    AND checklist.id = source.checklist_id
		    AND checklist.owner_uid IS NULL
		    AND checklist.deleted_at IS NOT NULL
		    AND source.status = 'retired'
		    AND source.current_release_revision_id IS NULL`,
		checklistID,
	)
	if err != nil {
		return fmt.Errorf("delete final retained source: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("count deleted retained sources: %w", err)
	}
	if rowsAffected == 0 {
		return nil
	}
	if _, err := tx.ExecContext(
		ctx,
		`DELETE FROM user_pmcs_community_releases
		  WHERE checklist_id = $1`,
		checklistID,
	); err != nil {
		return fmt.Errorf("delete final retained releases: %w", err)
	}
	if _, err := tx.ExecContext(
		ctx,
		`DELETE FROM user_pmcs_revisions
		  WHERE checklist_id = $1`,
		checklistID,
	); err != nil {
		return fmt.Errorf("delete final retained revisions: %w", err)
	}
	if _, err := tx.ExecContext(
		ctx,
		`DELETE FROM user_pmcs_checklists
		  WHERE id = $1
		    AND owner_uid IS NULL
		    AND deleted_at IS NOT NULL`,
		checklistID,
	); err != nil {
		return fmt.Errorf("delete final retained checklist: %w", err)
	}
	return nil
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
