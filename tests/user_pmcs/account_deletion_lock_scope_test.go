package user_pmcs_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"miltechserver/api/user_general"
	"miltechserver/api/user_pmcs/persistence"
	"miltechserver/api/user_pmcs/shared"
	"miltechserver/api/user_pmcs/subscriptions"
)

func TestOrdinaryUnsubscribeDoesNotLockHistoricalSubscriptions(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	ownerUID := accountDeletionLockScopeUser(t, "owner")
	subscriberUID := accountDeletionLockScopeUser(t, "middle")
	historicalUID := accountDeletionLockScopeUser(t, "z-history")
	checklistID := uuid.New()
	revisionID := uuid.New()
	cleanupAccountDeletionLockScope(
		t,
		[]string{ownerUID, subscriberUID, historicalUID},
		checklistID,
	)
	insertAccountDeletionChecklist(t, ownerUID, checklistID, nil)
	revisionOne := int32(1)
	insertAccountDeletionRevision(
		t,
		checklistID,
		revisionID,
		"published",
		&revisionOne,
	)
	insertAccountDeletionSource(t, checklistID, revisionID, 1)
	insertAccountDeletionLockScopeSubscription(
		t,
		subscriberUID,
		checklistID,
		&revisionID,
		false,
	)
	insertAccountDeletionLockScopeSubscription(
		t,
		historicalUID,
		checklistID,
		nil,
		true,
	)

	historicalGate := lockAccountDeletionSubscription(
		t,
		ctx,
		historicalUID,
		checklistID,
	)
	repository := subscriptions.NewRepository(
		persistence.NewStore(testDB, 1),
		shared.DefaultConfig(),
	)
	result := make(chan accountDeletionOperationResult, 1)
	go func() {
		_, err := repository.Unsubscribe(
			ctx,
			subscriberUID,
			checklistID,
			shared.Precondition{
				Mode: shared.PreconditionMatch,
				ETag: shared.MakeSubscriptionETag(checklistID, 1),
			},
		)
		result <- accountDeletionOperationResult{err: err}
	}()

	operation, completed := receiveAccountDeletionOperationWithin(
		result,
		2*time.Second,
	)
	require.NoError(t, historicalGate.Rollback())
	if !completed {
		require.NoError(t, receiveAccountDeletionOperation(t, result))
	}
	require.True(
		t,
		completed,
		"ordinary unsubscribe waited for an unrelated historical tombstone",
	)
	require.NoError(t, operation.err)
	require.Equal(t, 1, accountDeletionCount(
		t,
		`SELECT count(*) FROM user_pmcs_community_sources
		  WHERE checklist_id = $1
		    AND status = 'active'
		    AND current_release_revision_id = $2`,
		checklistID,
		revisionID,
	))
}

func TestFinalUnsubscribeLocksActivePinsButNotHistoricalSubscriptions(
	t *testing.T,
) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	ownerUID := accountDeletionLockScopeUser(t, "owner")
	activePinUID := accountDeletionLockScopeUser(t, "a-active")
	subscriberUID := accountDeletionLockScopeUser(t, "middle")
	historicalUID := accountDeletionLockScopeUser(t, "z-history")
	checklistID := uuid.New()
	revisionID := uuid.New()
	cleanupAccountDeletionLockScope(
		t,
		[]string{
			ownerUID,
			activePinUID,
			subscriberUID,
			historicalUID,
		},
		checklistID,
	)
	insertAccountDeletionChecklist(t, ownerUID, checklistID, nil)
	revisionOne := int32(1)
	insertAccountDeletionRevision(
		t,
		checklistID,
		revisionID,
		"published",
		&revisionOne,
	)
	insertAccountDeletionSource(t, checklistID, revisionID, 1)
	for _, uid := range []string{activePinUID, subscriberUID} {
		insertAccountDeletionLockScopeSubscription(
			t,
			uid,
			checklistID,
			&revisionID,
			false,
		)
	}
	insertAccountDeletionLockScopeSubscription(
		t,
		historicalUID,
		checklistID,
		nil,
		true,
	)
	accountRepository := user_general.NewRepository(
		testDB,
		persistence.NewAccountCleaner(),
	)
	require.NoError(t, accountRepository.DeleteUser(ctx, ownerUID))

	historicalGate := lockAccountDeletionSubscription(
		t,
		ctx,
		historicalUID,
		checklistID,
	)
	activePinTransition, err := testDB.BeginTx(ctx, nil)
	require.NoError(t, err)
	activePinTransitionReleased := false
	t.Cleanup(func() {
		if !activePinTransitionReleased {
			_ = activePinTransition.Rollback()
		}
	})
	_, err = activePinTransition.ExecContext(
		ctx,
		`UPDATE user_pmcs_subscriptions
		    SET installed_revision_id = NULL,
		        deleted_at = now(),
		        sync_version = sync_version + 1,
		        account_change_version = account_change_version + 1,
		        updated_at = now()
		  WHERE subscriber_uid = $1
		    AND checklist_id = $2`,
		activePinUID,
		checklistID,
	)
	require.NoError(t, err)

	repository := subscriptions.NewRepository(
		persistence.NewStore(testDB, 1),
		shared.DefaultConfig(),
	)
	result := make(chan accountDeletionOperationResult, 1)
	go func() {
		_, unsubscribeErr := repository.Unsubscribe(
			ctx,
			subscriberUID,
			checklistID,
			shared.Precondition{
				Mode: shared.PreconditionMatch,
				ETag: shared.MakeSubscriptionETag(checklistID, 1),
			},
		)
		result <- accountDeletionOperationResult{err: unsubscribeErr}
	}()
	require.True(
		t,
		waitForDatabaseActivity(
			5*time.Second,
			`lower(wait_event_type) = 'lock'
			 AND query LIKE '%user_pmcs_subscriptions%'`,
		),
		"final unsubscribe did not stabilize the concurrent active pin",
	)
	require.NoError(t, activePinTransition.Commit())
	activePinTransitionReleased = true

	operation, completed := receiveAccountDeletionOperationWithin(
		result,
		2*time.Second,
	)
	require.NoError(t, historicalGate.Rollback())
	if !completed {
		require.NoError(t, receiveAccountDeletionOperation(t, result))
	}
	require.True(
		t,
		completed,
		"final unsubscribe waited for an unrelated historical tombstone",
	)
	require.NoError(t, operation.err)
	for _, table := range []string{
		"user_pmcs_community_sources",
		"user_pmcs_community_releases",
		"user_pmcs_revisions",
		"user_pmcs_checklists",
	} {
		require.Equal(t, 0, accountDeletionCount(
			t,
			`SELECT count(*) FROM `+table+` WHERE `+
				accountDeletionChecklistColumn(table)+` = $1`,
			checklistID,
		))
	}
	require.Equal(t, 3, accountDeletionCount(
		t,
		`SELECT count(*) FROM user_pmcs_subscriptions
		  WHERE checklist_id = $1
		    AND installed_revision_id IS NULL
		    AND deleted_at IS NOT NULL`,
		checklistID,
	))
}

func TestAccountDeletionLocksActivePinsButNotHistoricalSubscriptions(
	t *testing.T,
) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	ownerUID := accountDeletionLockScopeUser(t, "owner")
	activePinUID := accountDeletionLockScopeUser(t, "a-active")
	historicalUID := accountDeletionLockScopeUser(t, "z-history")
	checklistID := uuid.New()
	revisionID := uuid.New()
	cleanupAccountDeletionLockScope(
		t,
		[]string{ownerUID, activePinUID, historicalUID},
		checklistID,
	)
	insertAccountDeletionChecklist(t, ownerUID, checklistID, nil)
	revisionOne := int32(1)
	insertAccountDeletionRevision(
		t,
		checklistID,
		revisionID,
		"published",
		&revisionOne,
	)
	insertAccountDeletionSource(t, checklistID, revisionID, 1)
	insertAccountDeletionLockScopeSubscription(
		t,
		activePinUID,
		checklistID,
		&revisionID,
		false,
	)
	insertAccountDeletionLockScopeSubscription(
		t,
		historicalUID,
		checklistID,
		nil,
		true,
	)

	historicalGate := lockAccountDeletionSubscription(
		t,
		ctx,
		historicalUID,
		checklistID,
	)
	activePinGate := lockAccountDeletionSubscription(
		t,
		ctx,
		activePinUID,
		checklistID,
	)
	repository := user_general.NewRepository(
		testDB,
		persistence.NewAccountCleaner(),
	)
	result := make(chan accountDeletionOperationResult, 1)
	go func() {
		result <- accountDeletionOperationResult{
			err: repository.DeleteUser(ctx, ownerUID),
		}
	}()
	require.True(
		t,
		waitForDatabaseActivity(
			5*time.Second,
			`lower(wait_event_type) = 'lock'
			 AND query LIKE '%user_pmcs_subscriptions%'`,
		),
		"account deletion did not stabilize the active incoming pin",
	)
	require.NoError(t, activePinGate.Rollback())

	operation, completed := receiveAccountDeletionOperationWithin(
		result,
		2*time.Second,
	)
	require.NoError(t, historicalGate.Rollback())
	if !completed {
		require.NoError(t, receiveAccountDeletionOperation(t, result))
	}
	require.True(
		t,
		completed,
		"account deletion waited for an unrelated historical tombstone",
	)
	require.NoError(t, operation.err)
	require.Equal(t, 1, accountDeletionCount(
		t,
		`SELECT count(*) FROM user_pmcs_subscriptions
		  WHERE subscriber_uid = $1
		    AND checklist_id = $2
		    AND installed_revision_id = $3
		    AND deleted_at IS NULL`,
		activePinUID,
		checklistID,
		revisionID,
	))
	require.Equal(t, 1, accountDeletionCount(
		t,
		`SELECT count(*) FROM user_pmcs_community_sources
		  WHERE checklist_id = $1
		    AND status = 'retired'
		    AND current_release_revision_id IS NULL`,
		checklistID,
	))
}

func accountDeletionLockScopeUser(t *testing.T, prefix string) string {
	t.Helper()
	uid := prefix + "-" + uuid.NewString()
	_, err := testDB.ExecContext(
		context.Background(),
		`INSERT INTO users (uid, email, username, created_at, is_enabled)
		 VALUES ($1, $2, 'user-pmcs-lock-scope', now(), TRUE)`,
		uid,
		uid+"@example.com",
	)
	require.NoError(t, err)
	_, err = testDB.ExecContext(
		context.Background(),
		`INSERT INTO user_pmcs_sync_state (user_uid, current_version)
		 VALUES ($1, 1)`,
		uid,
	)
	require.NoError(t, err)
	return uid
}

func insertAccountDeletionLockScopeSubscription(
	t *testing.T,
	subscriberUID string,
	checklistID uuid.UUID,
	revisionID *uuid.UUID,
	deleted bool,
) {
	t.Helper()
	var deletedAt any
	if deleted {
		deletedAt = time.Now().UTC()
	}
	_, err := testDB.ExecContext(
		context.Background(),
		`INSERT INTO user_pmcs_subscriptions
		     (subscriber_uid, checklist_id, installed_revision_id,
		      sync_version, account_change_version, deleted_at)
		 VALUES ($1, $2, $3, 1, 1, $4)`,
		subscriberUID,
		checklistID,
		revisionID,
		deletedAt,
	)
	require.NoError(t, err)
}

func lockAccountDeletionSubscription(
	t *testing.T,
	ctx context.Context,
	subscriberUID string,
	checklistID uuid.UUID,
) *sql.Tx {
	t.Helper()
	tx, err := testDB.BeginTx(ctx, nil)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = tx.Rollback()
	})
	var lockedUID string
	require.NoError(t, tx.QueryRowContext(
		ctx,
		`SELECT subscriber_uid
		   FROM user_pmcs_subscriptions
		  WHERE subscriber_uid = $1
		    AND checklist_id = $2
		  FOR UPDATE`,
		subscriberUID,
		checklistID,
	).Scan(&lockedUID))
	require.Equal(t, subscriberUID, lockedUID)
	return tx
}

func receiveAccountDeletionOperationWithin(
	result <-chan accountDeletionOperationResult,
	timeout time.Duration,
) (accountDeletionOperationResult, bool) {
	select {
	case operation := <-result:
		return operation, true
	case <-time.After(timeout):
		return accountDeletionOperationResult{}, false
	}
}

func cleanupAccountDeletionLockScope(
	t *testing.T,
	uids []string,
	checklistID uuid.UUID,
) {
	t.Helper()
	t.Cleanup(func() {
		cleanupAccountDeletionFixtures(
			t,
			"",
			"",
			[]uuid.UUID{checklistID},
		)
		for _, uid := range uids {
			_, _ = testDB.ExecContext(
				context.Background(),
				`DELETE FROM users WHERE uid = $1`,
				uid,
			)
		}
	})
}
