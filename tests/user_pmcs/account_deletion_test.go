package user_pmcs_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"miltechserver/api/user_general"
	"miltechserver/api/user_pmcs/community"
	"miltechserver/api/user_pmcs/persistence"
	"miltechserver/api/user_pmcs/shared"
	"miltechserver/api/user_pmcs/subscriptions"
	"miltechserver/bootstrap"
)

func TestAccountDeletionRetainsOnlyPinnedPublicContent(t *testing.T) {
	ctx := context.Background()
	ownerUID := accountDeletionUser(t, "owner")
	subscriberUID := accountDeletionUser(t, "subscriber")
	secondSubscriberUID := accountDeletionUser(t, "subscriber-two")
	tombstonedSubscriberUID := accountDeletionUser(t, "subscriber-deleted")
	privateID := uuid.New()
	privateRevisionID := uuid.New()
	tombstoneID := uuid.New()
	unpinnedID := uuid.New()
	unpinnedRevisionID := uuid.New()
	pinnedID := uuid.New()
	pinnedRevisionID := uuid.New()
	unpinnedCurrentID := uuid.New()
	externalID := uuid.New()
	allChecklistIDs := []uuid.UUID{
		privateID, tombstoneID, unpinnedID, pinnedID, externalID,
	}
	t.Cleanup(func() {
		cleanupAccountDeletionFixtures(
			t, ownerUID, subscriberUID, allChecklistIDs,
		)
	})
	t.Cleanup(func() {
		for _, uid := range []string{
			secondSubscriberUID,
			tombstonedSubscriberUID,
		} {
			_, _ = testDB.ExecContext(
				context.Background(),
				`DELETE FROM users WHERE uid = $1`,
				uid,
			)
		}
	})

	insertAccountDeletionChecklist(
		t, ownerUID, privateID, nil,
	)
	insertAccountDeletionRevision(
		t, privateID, privateRevisionID, "draft", nil,
	)
	deletedAt := time.Now().UTC()
	insertAccountDeletionChecklist(
		t, ownerUID, tombstoneID, &deletedAt,
	)
	insertAccountDeletionChecklist(t, ownerUID, unpinnedID, nil)
	revisionOne := int32(1)
	insertAccountDeletionRevision(
		t, unpinnedID, unpinnedRevisionID, "published", &revisionOne,
	)
	insertAccountDeletionSource(
		t, unpinnedID, unpinnedRevisionID, 1,
	)
	insertAccountDeletionChecklist(t, ownerUID, pinnedID, nil)
	pinnedTree := preparedTree(t, pinnedRevisionID)
	insertAccountDeletionPreparedRevision(
		t,
		pinnedID,
		pinnedTree,
		"superseded",
		revisionOne,
	)
	revisionTwo := int32(2)
	insertAccountDeletionRevision(
		t, pinnedID, unpinnedCurrentID, "published", &revisionTwo,
	)
	_, err := testDB.ExecContext(
		ctx,
		`INSERT INTO user_pmcs_community_releases
		     (revision_id, checklist_id, released_at)
		 VALUES ($1, $2, now())`,
		pinnedRevisionID,
		pinnedID,
	)
	require.NoError(t, err)
	_, err = testDB.ExecContext(
		ctx,
		`INSERT INTO user_pmcs_subscriptions
		     (subscriber_uid, checklist_id, installed_revision_id,
		      sync_version, account_change_version)
		 VALUES ($1, $2, $3, 1, 1)`,
		secondSubscriberUID,
		pinnedID,
		pinnedRevisionID,
	)
	require.NoError(t, err)
	_, err = testDB.ExecContext(
		ctx,
		`INSERT INTO user_pmcs_subscriptions
		     (subscriber_uid, checklist_id, installed_revision_id,
		      sync_version, account_change_version, deleted_at)
		 VALUES ($1, $2, NULL, 1, 1, now())`,
		tombstonedSubscriberUID,
		pinnedID,
	)
	require.NoError(t, err)
	insertAccountDeletionSource(t, pinnedID, unpinnedCurrentID, 2)
	_, err = testDB.ExecContext(
		ctx,
		`INSERT INTO user_pmcs_subscriptions
		     (subscriber_uid, checklist_id, installed_revision_id,
		      sync_version, account_change_version)
		 VALUES ($1, $2, $3, 1, 1)`,
		subscriberUID,
		pinnedID,
		pinnedRevisionID,
	)
	require.NoError(t, err)

	insertAccountDeletionChecklist(t, subscriberUID, externalID, nil)
	_, err = testDB.ExecContext(
		ctx,
		`INSERT INTO user_pmcs_subscriptions
		     (subscriber_uid, checklist_id, installed_revision_id,
		      sync_version, account_change_version, deleted_at)
		 VALUES ($1, $2, NULL, 1, 1, now())`,
		ownerUID,
		externalID,
	)
	require.NoError(t, err)

	subscriptionService := subscriptions.NewService(
		subscriptions.NewRepository(
			persistence.NewStore(testDB, 3),
			shared.DefaultConfig(),
		),
		shared.DefaultConfig(),
	)
	before, beforeETag, err := subscriptionService.GetInstalledRelease(
		ctx,
		&bootstrap.User{UserID: subscriberUID},
		pinnedID.String(),
		pinnedRevisionID.String(),
	)
	require.NoError(t, err)
	require.Equal(t, "active", before.SourceStatus)
	require.Equal(t, "user-pmcs-test", before.CreatorDisplayName)
	require.NotEmpty(t, before.Revision.Models)
	require.Len(t, before.Revision.Sections, 1)
	require.Len(t, before.Revision.Sections[0].Items, 1)
	require.Len(t, before.Revision.Sections[0].Items[0].Notices, 1)
	require.Len(t, before.Revision.Sections[0].Items[0].ProcedureSteps, 1)
	beforeRevision := before.Revision

	repository := user_general.NewRepository(
		testDB,
		persistence.NewAccountCleaner(),
	)
	require.NoError(t, repository.DeleteUser(ctx, ownerUID))

	require.Equal(t, 0, accountDeletionCount(
		t, `SELECT count(*) FROM users WHERE uid = $1`, ownerUID,
	))
	require.Equal(t, 0, accountDeletionCount(
		t,
		`SELECT count(*) FROM user_pmcs_sync_state WHERE user_uid = $1`,
		ownerUID,
	))
	require.Equal(t, 0, accountDeletionCount(
		t,
		`SELECT count(*) FROM user_pmcs_subscriptions
		  WHERE subscriber_uid = $1`,
		ownerUID,
	))
	for _, removedID := range []uuid.UUID{
		privateID, tombstoneID, unpinnedID,
	} {
		require.Equal(t, 0, accountDeletionCount(
			t,
			`SELECT count(*) FROM user_pmcs_checklists WHERE id = $1`,
			removedID,
		))
	}
	require.Equal(t, 0, accountDeletionCount(
		t,
		`SELECT count(*) FROM user_pmcs_revisions WHERE id IN ($1, $2)`,
		privateRevisionID,
		unpinnedRevisionID,
	))

	var (
		retainedOwner sql.NullString
		retainedAt    sql.NullTime
		sourceStatus  string
		currentID     uuid.NullUUID
	)
	err = testDB.QueryRowContext(
		ctx,
		`SELECT owner_uid, deleted_at
		   FROM user_pmcs_checklists WHERE id = $1`,
		pinnedID,
	).Scan(&retainedOwner, &retainedAt)
	require.NoError(t, err)
	require.False(t, retainedOwner.Valid)
	require.True(t, retainedAt.Valid)
	err = testDB.QueryRowContext(
		ctx,
		`SELECT status, current_release_revision_id
		   FROM user_pmcs_community_sources WHERE checklist_id = $1`,
		pinnedID,
	).Scan(&sourceStatus, &currentID)
	require.NoError(t, err)
	require.Equal(t, "retired", sourceStatus)
	require.False(t, currentID.Valid)
	require.Equal(t, 1, accountDeletionCount(
		t,
		`SELECT count(*) FROM user_pmcs_community_releases
		  WHERE checklist_id = $1 AND revision_id = $2`,
		pinnedID,
		pinnedRevisionID,
	))
	require.Equal(t, 0, accountDeletionCount(
		t,
		`SELECT count(*) FROM user_pmcs_community_releases
		  WHERE checklist_id = $1 AND revision_id = $2`,
		pinnedID,
		unpinnedCurrentID,
	))
	require.Equal(t, 1, accountDeletionCount(
		t,
		`SELECT count(*) FROM user_pmcs_revisions WHERE id = $1`,
		pinnedRevisionID,
	))
	require.Equal(t, 0, accountDeletionCount(
		t,
		`SELECT count(*) FROM user_pmcs_revisions WHERE id = $1`,
		unpinnedCurrentID,
	))

	after, afterETag, err := subscriptionService.GetInstalledRelease(
		ctx,
		&bootstrap.User{UserID: subscriberUID},
		pinnedID.String(),
		pinnedRevisionID.String(),
	)
	require.NoError(t, err)
	require.Equal(t, "retired", after.SourceStatus)
	require.Equal(t, "Deleted user", after.CreatorDisplayName)
	require.NotEmpty(t, afterETag)
	require.NotEqual(t, beforeETag, afterETag)
	require.Equal(t, beforeRevision, after.Revision)

	publicRepository := community.NewRepository(
		persistence.NewStore(testDB, 3),
		shared.DefaultConfig(),
	)
	page, err := publicRepository.Browse(
		ctx,
		shared.CommunityBrowseFilter{Limit: 20},
	)
	require.NoError(t, err)
	for _, item := range page.Items {
		require.NotEqual(t, pinnedID, item.ChecklistID)
	}
	_, err = publicRepository.GetCurrentRelease(ctx, pinnedID)
	require.Error(t, err)
}

func TestAccountDeletionWaitsForConcurrentUnsubscribeBeforePinPruning(
	t *testing.T,
) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	ownerUID := accountDeletionUser(t, "race-owner")
	subscriberUID := accountDeletionUser(t, "race-subscriber")
	checklistID := uuid.New()
	revisionID := uuid.New()
	t.Cleanup(func() {
		cleanupAccountDeletionFixtures(
			t,
			ownerUID,
			subscriberUID,
			[]uuid.UUID{checklistID},
		)
	})
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
	_, err := testDB.ExecContext(
		ctx,
		`INSERT INTO user_pmcs_subscriptions
		     (subscriber_uid, checklist_id, installed_revision_id,
		      sync_version, account_change_version)
		 VALUES ($1, $2, $3, 1, 1)`,
		subscriberUID,
		checklistID,
		revisionID,
	)
	require.NoError(t, err)

	unsubscribeTransaction, err := testDB.BeginTx(ctx, nil)
	require.NoError(t, err)
	unsubscribeCommitted := false
	t.Cleanup(func() {
		if !unsubscribeCommitted {
			_ = unsubscribeTransaction.Rollback()
		}
	})
	_, err = unsubscribeTransaction.ExecContext(
		ctx,
		`UPDATE user_pmcs_subscriptions
		    SET installed_revision_id = NULL,
		        sync_version = sync_version + 1,
		        account_change_version = account_change_version + 1,
		        updated_at = now(),
		        deleted_at = now()
		  WHERE subscriber_uid = $1
		    AND checklist_id = $2`,
		subscriberUID,
		checklistID,
	)
	require.NoError(t, err)

	accountRepository := user_general.NewRepository(
		testDB,
		persistence.NewAccountCleaner(),
	)
	deletionResult := make(chan accountDeletionOperationResult, 1)
	go func() {
		deletionResult <- accountDeletionOperationResult{
			err: accountRepository.DeleteUser(ctx, ownerUID),
		}
	}()

	waitedForUnsubscribe := waitForDatabaseActivity(
		5*time.Second,
		`lower(wait_event_type) = 'lock'
		 AND query LIKE '%user_pmcs_subscriptions%'`,
	)
	require.NoError(t, unsubscribeTransaction.Commit())
	unsubscribeCommitted = true
	require.NoError(t, receiveAccountDeletionOperation(t, deletionResult))
	require.True(
		t,
		waitedForUnsubscribe,
		"account deletion did not wait for the unsubscribe row lock",
	)

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
	require.Equal(t, 1, accountDeletionCount(
		t,
		`SELECT count(*) FROM user_pmcs_subscriptions
		  WHERE subscriber_uid = $1
		    AND checklist_id = $2
		    AND installed_revision_id IS NULL
		    AND deleted_at IS NOT NULL`,
		subscriberUID,
		checklistID,
	))
}

func TestAccountDeletionFirstThenFinalUnsubscribeRemovesRetainedContent(
	t *testing.T,
) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	ownerUID := accountDeletionUser(t, "deletion-first-owner")
	subscriberUID := accountDeletionUser(t, "deletion-first-subscriber")
	checklistID := uuid.New()
	revisionID := uuid.New()
	t.Cleanup(func() {
		cleanupAccountDeletionFixtures(
			t,
			ownerUID,
			subscriberUID,
			[]uuid.UUID{checklistID},
		)
	})
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
	_, err := testDB.ExecContext(
		ctx,
		`INSERT INTO user_pmcs_subscriptions
		     (subscriber_uid, checklist_id, installed_revision_id,
		      sync_version, account_change_version)
		 VALUES ($1, $2, $3, 1, 1)`,
		subscriberUID,
		checklistID,
		revisionID,
	)
	require.NoError(t, err)

	gate, err := testDB.BeginTx(ctx, nil)
	require.NoError(t, err)
	gateCommitted := false
	t.Cleanup(func() {
		if !gateCommitted {
			_ = gate.Rollback()
		}
	})
	var lockedVersion int64
	require.NoError(t, gate.QueryRowContext(
		ctx,
		`SELECT current_version
		   FROM user_pmcs_sync_state
		  WHERE user_uid = $1
		  FOR UPDATE`,
		subscriberUID,
	).Scan(&lockedVersion))

	subscriptionRepository := subscriptions.NewRepository(
		persistence.NewStore(testDB, 1),
		shared.DefaultConfig(),
	)
	unsubscribeResult := make(chan accountDeletionOperationResult, 1)
	go func() {
		_, unsubscribeErr := subscriptionRepository.Unsubscribe(
			ctx,
			subscriberUID,
			checklistID,
			shared.Precondition{
				Mode: shared.PreconditionMatch,
				ETag: shared.MakeSubscriptionETag(checklistID, 1),
			},
		)
		unsubscribeResult <- accountDeletionOperationResult{
			err: unsubscribeErr,
		}
	}()
	require.True(t, waitForDatabaseActivity(
		5*time.Second,
		`lower(wait_event_type) = 'lock'
		 AND query LIKE '%user_pmcs_sync_state%'`,
	))

	accountRepository := user_general.NewRepository(
		testDB,
		persistence.NewAccountCleaner(),
	)
	require.NoError(t, accountRepository.DeleteUser(ctx, ownerUID))
	require.NoError(t, gate.Commit())
	gateCommitted = true
	require.NoError(t, receiveAccountDeletionOperation(
		t,
		unsubscribeResult,
	))

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
	require.Equal(t, 1, accountDeletionCount(
		t,
		`SELECT count(*) FROM user_pmcs_subscriptions
		  WHERE subscriber_uid = $1
		    AND checklist_id = $2
		    AND installed_revision_id IS NULL
		    AND deleted_at IS NOT NULL`,
		subscriberUID,
		checklistID,
	))
}

func TestUnsubscribeFinalPinCleanupPreservesIneligibleContent(t *testing.T) {
	t.Run("active owned source", func(t *testing.T) {
		ownerUID := accountDeletionUser(t, "active-owner")
		subscriberUID := accountDeletionUser(t, "active-subscriber")
		checklistID := uuid.New()
		revisionID := uuid.New()
		t.Cleanup(func() {
			cleanupAccountDeletionFixtures(
				t,
				ownerUID,
				subscriberUID,
				[]uuid.UUID{checklistID},
			)
		})
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
		_, err := testDB.ExecContext(
			context.Background(),
			`INSERT INTO user_pmcs_subscriptions
			     (subscriber_uid, checklist_id, installed_revision_id,
			      sync_version, account_change_version)
			 VALUES ($1, $2, $3, 1, 1)`,
			subscriberUID,
			checklistID,
			revisionID,
		)
		require.NoError(t, err)

		repository := subscriptions.NewRepository(
			persistence.NewStore(testDB, 1),
			shared.DefaultConfig(),
		)
		_, err = repository.Unsubscribe(
			context.Background(),
			subscriberUID,
			checklistID,
			shared.Precondition{
				Mode: shared.PreconditionMatch,
				ETag: shared.MakeSubscriptionETag(checklistID, 1),
			},
		)
		require.NoError(t, err)

		require.Equal(t, 1, accountDeletionCount(
			t,
			`SELECT count(*) FROM user_pmcs_checklists
			  WHERE id = $1 AND owner_uid = $2 AND deleted_at IS NULL`,
			checklistID,
			ownerUID,
		))
		require.Equal(t, 1, accountDeletionCount(
			t,
			`SELECT count(*) FROM user_pmcs_community_sources
			  WHERE checklist_id = $1
			    AND status = 'active'
			    AND current_release_revision_id = $2`,
			checklistID,
			revisionID,
		))
		require.Equal(t, 1, accountDeletionCount(
			t,
			`SELECT count(*) FROM user_pmcs_community_releases
			  WHERE checklist_id = $1 AND revision_id = $2`,
			checklistID,
			revisionID,
		))
	})

	t.Run("another active pin", func(t *testing.T) {
		ownerUID := accountDeletionUser(t, "pinned-owner")
		firstSubscriberUID := accountDeletionUser(t, "pinned-first")
		secondSubscriberUID := accountDeletionUser(t, "pinned-second")
		checklistID := uuid.New()
		revisionID := uuid.New()
		t.Cleanup(func() {
			cleanupAccountDeletionFixtures(
				t,
				ownerUID,
				firstSubscriberUID,
				[]uuid.UUID{checklistID},
			)
			_, _ = testDB.ExecContext(
				context.Background(),
				`DELETE FROM users WHERE uid = $1`,
				secondSubscriberUID,
			)
		})
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
		for _, subscriberUID := range []string{
			firstSubscriberUID,
			secondSubscriberUID,
		} {
			_, err := testDB.ExecContext(
				context.Background(),
				`INSERT INTO user_pmcs_subscriptions
				     (subscriber_uid, checklist_id, installed_revision_id,
				      sync_version, account_change_version)
				 VALUES ($1, $2, $3, 1, 1)`,
				subscriberUID,
				checklistID,
				revisionID,
			)
			require.NoError(t, err)
		}
		accountRepository := user_general.NewRepository(
			testDB,
			persistence.NewAccountCleaner(),
		)
		require.NoError(t, accountRepository.DeleteUser(
			context.Background(),
			ownerUID,
		))

		repository := subscriptions.NewRepository(
			persistence.NewStore(testDB, 1),
			shared.DefaultConfig(),
		)
		_, err := repository.Unsubscribe(
			context.Background(),
			firstSubscriberUID,
			checklistID,
			shared.Precondition{
				Mode: shared.PreconditionMatch,
				ETag: shared.MakeSubscriptionETag(checklistID, 1),
			},
		)
		require.NoError(t, err)

		require.Equal(t, 1, accountDeletionCount(
			t,
			`SELECT count(*) FROM user_pmcs_checklists
			  WHERE id = $1 AND owner_uid IS NULL
			    AND deleted_at IS NOT NULL`,
			checklistID,
		))
		require.Equal(t, 1, accountDeletionCount(
			t,
			`SELECT count(*) FROM user_pmcs_community_sources
			  WHERE checklist_id = $1 AND status = 'retired'
			    AND current_release_revision_id IS NULL`,
			checklistID,
		))
		require.Equal(t, 1, accountDeletionCount(
			t,
			`SELECT count(*) FROM user_pmcs_community_releases
			  WHERE checklist_id = $1 AND revision_id = $2`,
			checklistID,
			revisionID,
		))
		require.Equal(t, 1, accountDeletionCount(
			t,
			`SELECT count(*) FROM user_pmcs_subscriptions
			  WHERE subscriber_uid = $1
			    AND checklist_id = $2
			    AND installed_revision_id = $3
			    AND deleted_at IS NULL`,
			secondSubscriberUID,
			checklistID,
			revisionID,
		))
	})
}

func accountDeletionChecklistColumn(table string) string {
	if table == "user_pmcs_checklists" {
		return "id"
	}
	return "checklist_id"
}

func receiveAccountDeletionOperation(
	t *testing.T,
	result <-chan accountDeletionOperationResult,
) error {
	t.Helper()
	select {
	case operation := <-result:
		return operation.err
	case <-time.After(10 * time.Second):
		t.Fatal("concurrent account deletion operation did not complete")
		return nil
	}
}

type accountDeletionFailingCleaner struct{ markerUID string }
type accountDeletionOperationResult struct{ err error }

func (cleaner accountDeletionFailingCleaner) CleanupAccount(
	ctx context.Context,
	tx *sql.Tx,
	_ string,
) error {
	_, err := tx.ExecContext(
		ctx,
		`DELETE FROM user_pmcs_sync_state WHERE user_uid = $1`,
		cleaner.markerUID,
	)
	if err != nil {
		return err
	}
	return errors.New("injected cleanup failure")
}

func TestAccountDeletionRollsBackCleanerAndUserDeletionTogether(t *testing.T) {
	ctx := context.Background()
	uid := accountDeletionUser(t, "rollback")
	t.Cleanup(func() {
		_, _ = testDB.ExecContext(
			context.Background(),
			`DELETE FROM users WHERE uid = $1`,
			uid,
		)
	})
	repository := user_general.NewRepository(
		testDB,
		accountDeletionFailingCleaner{markerUID: uid},
	)

	err := repository.DeleteUser(ctx, uid)
	require.ErrorContains(t, err, "injected cleanup failure")
	require.Equal(t, 1, accountDeletionCount(
		t, `SELECT count(*) FROM users WHERE uid = $1`, uid,
	))
	require.Equal(t, 1, accountDeletionCount(
		t,
		`SELECT count(*) FROM user_pmcs_sync_state WHERE user_uid = $1`,
		uid,
	))
}

func TestAccountDeletionRestrictiveFKRejectsRevisionBeforeRelease(t *testing.T) {
	ownerUID := accountDeletionUser(t, "fk-owner")
	checklistID := uuid.New()
	revisionID := uuid.New()
	t.Cleanup(func() {
		cleanupAccountDeletionFixtures(
			t,
			ownerUID,
			"",
			[]uuid.UUID{checklistID},
		)
	})
	insertAccountDeletionChecklist(t, ownerUID, checklistID, nil)
	revisionOne := int32(1)
	insertAccountDeletionRevision(
		t, checklistID, revisionID, "published", &revisionOne,
	)
	insertAccountDeletionSource(t, checklistID, revisionID, 1)

	tx, err := testDB.BeginTx(context.Background(), nil)
	require.NoError(t, err)
	defer tx.Rollback()
	_, err = tx.ExecContext(
		context.Background(),
		`DELETE FROM user_pmcs_revisions WHERE id = $1`,
		revisionID,
	)
	require.Error(t, err)
	require.Error(t, tx.Commit())
}

func accountDeletionUser(t *testing.T, label string) string {
	t.Helper()
	uid := "ad-" + uuid.NewString()
	_, err := testDB.ExecContext(
		context.Background(),
		`INSERT INTO users (uid, email, username, created_at, is_enabled)
		 VALUES ($1, $2, 'user-pmcs-test', now(), TRUE)`,
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

func insertAccountDeletionChecklist(
	t *testing.T,
	ownerUID string,
	checklistID uuid.UUID,
	deletedAt *time.Time,
) {
	t.Helper()
	_, err := testDB.ExecContext(
		context.Background(),
		`INSERT INTO user_pmcs_checklists
		     (id, owner_uid, sync_version, account_change_version, deleted_at)
		 VALUES ($1, $2, 1, 1, $3)`,
		checklistID,
		ownerUID,
		deletedAt,
	)
	require.NoError(t, err)
}

func insertAccountDeletionRevision(
	t *testing.T,
	checklistID uuid.UUID,
	revisionID uuid.UUID,
	state string,
	revisionNumber *int32,
) {
	t.Helper()
	var publishedAt any
	if revisionNumber != nil {
		publishedAt = time.Now().UTC()
	}
	_, err := testDB.ExecContext(
		context.Background(),
		`INSERT INTO user_pmcs_revisions
		     (id, checklist_id, state, revision_number, name, description,
		      content_hash, published_at)
		 VALUES ($1, $2, $3, $4, '', '', $5, $6)`,
		revisionID,
		checklistID,
		state,
		revisionNumber,
		make([]byte, 32),
		publishedAt,
	)
	require.NoError(t, err)
}

func insertAccountDeletionPreparedRevision(
	t *testing.T,
	checklistID uuid.UUID,
	prepared shared.PreparedRevision,
	state string,
	revisionNumber int32,
) {
	t.Helper()
	insertAccountDeletionRevision(
		t,
		checklistID,
		prepared.Input.ID,
		"draft",
		nil,
	)
	tx, err := testDB.BeginTx(context.Background(), nil)
	require.NoError(t, err)
	defer tx.Rollback()
	require.NoError(t, persistence.ReplaceDraftTree(
		context.Background(),
		tx,
		checklistID,
		prepared,
	))
	_, err = tx.ExecContext(
		context.Background(),
		`UPDATE user_pmcs_revisions
		    SET state = $1,
		        revision_number = $2,
		        published_at = now()
		  WHERE id = $3`,
		state,
		revisionNumber,
		prepared.Input.ID,
	)
	require.NoError(t, err)
	require.NoError(t, tx.Commit())
}

func insertAccountDeletionSource(
	t *testing.T,
	checklistID uuid.UUID,
	currentRevisionID uuid.UUID,
	latestNumber int32,
) {
	t.Helper()
	now := time.Now().UTC()
	_, err := testDB.ExecContext(
		context.Background(),
		`INSERT INTO user_pmcs_community_releases
		     (revision_id, checklist_id, released_at)
		 VALUES ($1, $2, $3)`,
		currentRevisionID,
		checklistID,
		now,
	)
	require.NoError(t, err)
	_, err = testDB.ExecContext(
		context.Background(),
		`INSERT INTO user_pmcs_community_sources
		     (checklist_id, status, current_release_revision_id,
		      latest_release_revision_number, first_released_at, updated_at)
		 VALUES ($1, 'active', $2, $3, $4, $4)`,
		checklistID,
		currentRevisionID,
		latestNumber,
		now,
	)
	require.NoError(t, err)
}

func accountDeletionCount(t *testing.T, query string, args ...any) int {
	t.Helper()
	var count int
	require.NoError(
		t,
		testDB.QueryRowContext(
			context.Background(), query, args...,
		).Scan(&count),
	)
	return count
}

func cleanupAccountDeletionFixtures(
	t *testing.T,
	firstUID string,
	secondUID string,
	checklistIDs []uuid.UUID,
) {
	t.Helper()
	ctx := context.Background()
	for _, checklistID := range checklistIDs {
		_, _ = testDB.ExecContext(
			ctx,
			`DELETE FROM user_pmcs_subscriptions WHERE checklist_id = $1`,
			checklistID,
		)
		_, _ = testDB.ExecContext(
			ctx,
			`UPDATE user_pmcs_community_sources
			    SET status = 'retired',
			        current_release_revision_id = NULL,
			        retired_at = COALESCE(retired_at, now())
			  WHERE checklist_id = $1`,
			checklistID,
		)
		_, _ = testDB.ExecContext(
			ctx,
			`DELETE FROM user_pmcs_community_sources
			  WHERE checklist_id = $1`,
			checklistID,
		)
		_, _ = testDB.ExecContext(
			ctx,
			`DELETE FROM user_pmcs_community_releases
			  WHERE checklist_id = $1`,
			checklistID,
		)
		_, _ = testDB.ExecContext(
			ctx,
			`DELETE FROM user_pmcs_revisions WHERE checklist_id = $1`,
			checklistID,
		)
		_, _ = testDB.ExecContext(
			ctx,
			`DELETE FROM user_pmcs_checklists WHERE id = $1`,
			checklistID,
		)
	}
	for _, uid := range []string{firstUID, secondUID} {
		if uid == "" {
			continue
		}
		_, _ = testDB.ExecContext(
			ctx,
			`DELETE FROM users WHERE uid = $1`,
			uid,
		)
	}
}
