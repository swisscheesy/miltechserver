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
	insertAccountDeletionRevision(
		t, pinnedID, pinnedRevisionID, "superseded", &revisionOne,
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
	require.Equal(t, pinnedRevisionID, after.Revision.ID)

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

type accountDeletionFailingCleaner struct{ markerUID string }

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
	uid := "acct-del-" + label + "-" + uuid.NewString()
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
