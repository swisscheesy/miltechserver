package user_pmcs_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"miltechserver/api/user_pmcs/persistence"
	"miltechserver/api/user_pmcs/shared"
	userpmcssync "miltechserver/api/user_pmcs/sync"
)

func TestAccountDeltaRequiresInitializedAccountAndSupportsZeroState(t *testing.T) {
	repository := newAccountDeltaRepository()

	_, err := repository.GetDelta(
		context.Background(),
		"missing-"+uuid.NewString(),
		0,
		10,
		shared.DefaultConfig().MaxDeltaResponseBytes,
	)
	var apiError *shared.APIError
	require.ErrorAs(t, err, &apiError)
	require.Equal(t, http.StatusConflict, apiError.Status)
	require.Equal(t, "account_not_initialized", apiError.Code)

	userUID := newUserPmcsTestUser(t)
	delta, err := repository.GetDelta(
		context.Background(),
		userUID,
		0,
		10,
		shared.DefaultConfig().MaxDeltaResponseBytes,
	)
	require.NoError(t, err)
	require.Equal(t, int64(0), delta.FromCursor)
	require.Equal(t, int64(0), delta.ThroughCursor)
	require.Equal(t, int64(0), delta.AccountVersion)
	require.False(t, delta.HasMore)
	require.Empty(t, delta.Changes)
	require.NotNil(t, delta.Changes)
}

func TestAccountDeltaMergesCompleteCurrentAggregatesAndTombstones(t *testing.T) {
	ctx := context.Background()
	subscriberUID := newUserPmcsTestUser(t)
	creatorUID := newUserPmcsTestUser(t)
	checklistID := uuid.New()
	draftID := uuid.New()
	publicationID := uuid.New()
	supersededID := uuid.New()
	deletedChecklistID := uuid.New()
	sourceChecklistID := uuid.New()
	installedRevisionID := uuid.New()
	unsubscribedChecklistID := uuid.New()

	insertDeltaChecklist(
		t,
		subscriberUID,
		checklistID,
		2,
		nil,
		[]deltaRevision{
			{
				prepared:       preparedTree(t, supersededID),
				state:          "superseded",
				revisionNumber: 1,
			},
			{
				prepared:       preparedTree(t, publicationID),
				state:          "published",
				revisionNumber: 2,
			},
			{prepared: preparedTree(t, draftID), state: "draft"},
		},
	)
	insertDeltaCommunitySource(t, checklistID, publicationID, 2)
	deletedAt := time.Now().UTC().Truncate(time.Microsecond)
	insertDeltaChecklist(
		t,
		subscriberUID,
		deletedChecklistID,
		4,
		&deletedAt,
		nil,
	)
	insertDeltaChecklist(
		t,
		creatorUID,
		sourceChecklistID,
		1,
		nil,
		[]deltaRevision{
			{
				prepared:       preparedTree(t, installedRevisionID),
				state:          "published",
				revisionNumber: 1,
			},
		},
	)
	releasedAt := insertDeltaCommunitySource(
		t,
		sourceChecklistID,
		installedRevisionID,
		1,
	)
	insertDeltaSubscription(
		t,
		subscriberUID,
		sourceChecklistID,
		&installedRevisionID,
		3,
		nil,
	)
	unsubscribedAt := deletedAt.Add(time.Second)
	insertDeltaSubscription(
		t,
		subscriberUID,
		unsubscribedChecklistID,
		nil,
		5,
		&unsubscribedAt,
	)
	setDeltaAccountVersion(t, subscriberUID, 5)

	t.Cleanup(func() {
		_, err := testDB.ExecContext(
			context.Background(),
			`DELETE FROM user_pmcs_subscriptions
			 WHERE subscriber_uid = $1`,
			subscriberUID,
		)
		require.NoError(t, err)
		for _, releasedChecklistID := range []uuid.UUID{
			checklistID,
			sourceChecklistID,
		} {
			_, err = testDB.ExecContext(
				context.Background(),
				`UPDATE user_pmcs_community_sources
				 SET status = 'retired',
				     current_release_revision_id = NULL,
				     retired_at = now()
				 WHERE checklist_id = $1`,
				releasedChecklistID,
			)
			require.NoError(t, err)
			_, err = testDB.ExecContext(
				context.Background(),
				`DELETE FROM user_pmcs_community_sources WHERE checklist_id = $1`,
				releasedChecklistID,
			)
			require.NoError(t, err)
			_, err = testDB.ExecContext(
				context.Background(),
				`DELETE FROM user_pmcs_community_releases WHERE checklist_id = $1`,
				releasedChecklistID,
			)
			require.NoError(t, err)
		}
	})

	repository := newAccountDeltaRepository()
	delta, err := repository.GetDelta(
		ctx,
		subscriberUID,
		0,
		25,
		shared.DefaultConfig().MaxDeltaResponseBytes,
	)
	require.NoError(t, err)
	require.Equal(t, int64(0), delta.FromCursor)
	require.Equal(t, int64(5), delta.ThroughCursor)
	require.Equal(t, int64(5), delta.AccountVersion)
	require.False(t, delta.HasMore)
	require.Len(t, delta.Changes, 4)
	require.Equal(
		t,
		[]int64{2, 3, 4, 5},
		accountChangeVersions(delta.Changes),
	)

	owned := delta.Changes[0]
	require.Equal(t, userpmcssync.ChangeKindChecklist, owned.Kind)
	require.NotNil(t, owned.Checklist)
	require.Nil(t, owned.Subscription)
	require.Nil(t, owned.Installed)
	require.Equal(t, checklistID, owned.Checklist.ID)
	require.Equal(t, int64(2), owned.Checklist.AccountChangeVersion)
	require.NotNil(t, owned.Checklist.Draft)
	require.Equal(t, draftID, owned.Checklist.Draft.ID)
	require.NotNil(t, owned.Checklist.Publication)
	require.Equal(t, publicationID, owned.Checklist.Publication.ID)
	require.NotEmpty(t, owned.Checklist.Draft.Sections)
	require.NotEmpty(t, owned.Checklist.Publication.Sections)
	require.NotEqual(t, supersededID, owned.Checklist.Publication.ID)
	require.NotNil(t, owned.Checklist.Community)
	require.Equal(t, "active", owned.Checklist.Community.Status)
	require.Equal(
		t,
		&publicationID,
		owned.Checklist.Community.CurrentReleaseRevisionID,
	)

	activeSubscription := delta.Changes[1]
	require.Equal(
		t,
		userpmcssync.ChangeKindSubscription,
		activeSubscription.Kind,
	)
	require.Nil(t, activeSubscription.Checklist)
	require.NotNil(t, activeSubscription.Subscription)
	require.NotNil(t, activeSubscription.Installed)
	require.Equal(t, sourceChecklistID, activeSubscription.Subscription.ChecklistID)
	require.Equal(
		t,
		installedRevisionID,
		activeSubscription.Installed.Revision.ID,
	)
	require.Equal(t, "active", activeSubscription.Installed.SourceStatus)
	require.Equal(t, "user-pmcs-test", activeSubscription.Installed.CreatorDisplayName)
	require.WithinDuration(
		t,
		releasedAt,
		activeSubscription.Installed.ReleasedAt,
		time.Microsecond,
	)
	require.NotEmpty(t, activeSubscription.Installed.Revision.Sections)

	checklistTombstone := delta.Changes[2]
	require.Equal(t, userpmcssync.ChangeKindChecklist, checklistTombstone.Kind)
	require.NotNil(t, checklistTombstone.Checklist)
	require.Equal(t, deletedChecklistID, checklistTombstone.Checklist.ID)
	require.NotNil(t, checklistTombstone.Checklist.DeletedAt)
	require.Nil(t, checklistTombstone.Checklist.Draft)
	require.Nil(t, checklistTombstone.Checklist.Publication)
	require.Nil(t, checklistTombstone.Checklist.Community)
	require.Nil(t, checklistTombstone.Subscription)
	require.Nil(t, checklistTombstone.Installed)

	subscriptionTombstone := delta.Changes[3]
	require.Equal(
		t,
		userpmcssync.ChangeKindSubscription,
		subscriptionTombstone.Kind,
	)
	require.Nil(t, subscriptionTombstone.Checklist)
	require.NotNil(t, subscriptionTombstone.Subscription)
	require.Equal(
		t,
		unsubscribedChecklistID,
		subscriptionTombstone.Subscription.ChecklistID,
	)
	require.NotNil(t, subscriptionTombstone.Subscription.DeletedAt)
	require.Nil(t, subscriptionTombstone.Subscription.InstalledRevisionID)
	require.Nil(t, subscriptionTombstone.Installed)

	limited, err := repository.GetDelta(
		ctx,
		subscriberUID,
		0,
		2,
		shared.DefaultConfig().MaxDeltaResponseBytes,
	)
	require.NoError(t, err)
	require.Len(t, limited.Changes, 2)
	require.Equal(t, int64(3), limited.ThroughCursor)
	require.True(t, limited.HasMore)

	oversizeAlone, err := repository.GetDelta(ctx, subscriberUID, 0, 25, 1)
	require.NoError(t, err)
	require.Len(t, oversizeAlone.Changes, 1)
	require.Equal(t, int64(2), oversizeAlone.ThroughCursor)
	require.True(t, oversizeAlone.HasMore)
	require.NotNil(t, oversizeAlone.Changes[0].Checklist.Draft)
	require.NotEmpty(t, oversizeAlone.Changes[0].Checklist.Draft.Sections)

	empty, err := repository.GetDelta(
		ctx,
		subscriberUID,
		5,
		25,
		shared.DefaultConfig().MaxDeltaResponseBytes,
	)
	require.NoError(t, err)
	require.Equal(t, int64(5), empty.FromCursor)
	require.Equal(t, int64(5), empty.ThroughCursor)
	require.Equal(t, int64(5), empty.AccountVersion)
	require.Empty(t, empty.Changes)
	require.False(t, empty.HasMore)
}

func TestAccountDeltaRepeatableSnapshotDefersConcurrentMutation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	userUID := newUserPmcsTestUser(t)
	checklistID := uuid.New()
	insertDeltaChecklist(t, userUID, checklistID, 1, nil, nil)
	setDeltaAccountVersion(t, userUID, 1)

	blocker, err := testDB.Conn(ctx)
	require.NoError(t, err)
	defer blocker.Close()
	blockerTx, err := blocker.BeginTx(ctx, nil)
	require.NoError(t, err)
	_, err = blockerTx.ExecContext(
		ctx,
		`LOCK TABLE user_pmcs_checklists IN ACCESS EXCLUSIVE MODE`,
	)
	require.NoError(t, err)

	type deltaResult struct {
		delta *userpmcssync.AccountDelta
		err   error
	}
	resultChannel := make(chan deltaResult, 1)
	repository := newAccountDeltaRepository()
	go func() {
		delta, getErr := repository.GetDelta(
			ctx,
			userUID,
			0,
			10,
			shared.DefaultConfig().MaxDeltaResponseBytes,
		)
		resultChannel <- deltaResult{delta: delta, err: getErr}
	}()

	require.Eventually(t, func() bool {
		var blocked int
		queryErr := testDB.QueryRowContext(
			ctx,
			`SELECT count(*)
			 FROM pg_stat_activity
			 WHERE query LIKE '%user_pmcs_account_delta_roots%'
			   AND wait_event_type = 'Lock'`,
		).Scan(&blocked)
		return queryErr == nil && blocked > 0
	}, 5*time.Second, 20*time.Millisecond)

	mutationDone := make(chan error, 1)
	go func() {
		tx, beginErr := testDB.BeginTx(ctx, nil)
		if beginErr != nil {
			mutationDone <- beginErr
			return
		}
		if _, updateErr := tx.ExecContext(
			ctx,
			`UPDATE user_pmcs_sync_state
			 SET current_version = 2, updated_at = now()
			 WHERE user_uid = $1`,
			userUID,
		); updateErr != nil {
			_ = tx.Rollback()
			mutationDone <- updateErr
			return
		}
		if _, updateErr := tx.ExecContext(
			ctx,
			`UPDATE user_pmcs_checklists
			 SET sync_version = 2,
			     account_change_version = 2,
			     updated_at = now()
			 WHERE id = $1`,
			checklistID,
		); updateErr != nil {
			_ = tx.Rollback()
			mutationDone <- updateErr
			return
		}
		mutationDone <- tx.Commit()
	}()

	require.NoError(t, blockerTx.Commit())
	require.NoError(t, <-mutationDone)
	first := <-resultChannel
	require.NoError(t, first.err)
	require.Equal(t, int64(1), first.delta.AccountVersion)
	require.Equal(t, int64(1), first.delta.ThroughCursor)
	require.Len(t, first.delta.Changes, 1)
	require.Equal(
		t,
		int64(1),
		first.delta.Changes[0].AccountChangeVersion,
	)

	next, err := repository.GetDelta(
		ctx,
		userUID,
		first.delta.ThroughCursor,
		10,
		shared.DefaultConfig().MaxDeltaResponseBytes,
	)
	require.NoError(t, err)
	require.Equal(t, int64(2), next.AccountVersion)
	require.Equal(t, int64(2), next.ThroughCursor)
	require.Len(t, next.Changes, 1)
	require.Equal(t, int64(2), next.Changes[0].AccountChangeVersion)
}

type deltaRevision struct {
	prepared       shared.PreparedRevision
	state          string
	revisionNumber int32
}

func newAccountDeltaRepository() userpmcssync.Repository {
	config := shared.DefaultConfig()
	store := persistence.NewStore(testDB, config.TransactionMaxAttempts)
	return userpmcssync.NewRepository(store)
}

func insertDeltaChecklist(
	t *testing.T,
	ownerUID string,
	checklistID uuid.UUID,
	accountVersion int64,
	deletedAt *time.Time,
	revisions []deltaRevision,
) {
	t.Helper()
	ctx := context.Background()
	tx, err := testDB.BeginTx(ctx, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = tx.Rollback() })

	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO user_pmcs_checklists
		     (id, owner_uid, sync_version, account_change_version, deleted_at)
		 VALUES ($1, $2, $3, $3, $4)`,
		checklistID,
		ownerUID,
		accountVersion,
		deletedAt,
	)
	require.NoError(t, err)

	for _, revision := range revisions {
		publishedAt := any(nil)
		revisionNumber := any(nil)
		if revision.state != "draft" {
			publishedAt = time.Now().UTC()
			revisionNumber = revision.revisionNumber
		}
		_, err = tx.ExecContext(
			ctx,
			`INSERT INTO user_pmcs_revisions
			     (id, checklist_id, state, revision_number, name, description,
			      content_hash, published_at)
			 VALUES ($1, $2, 'draft', NULL, $3, $4, $5, NULL)`,
			revision.prepared.Input.ID,
			checklistID,
			revision.prepared.Input.Name,
			revision.prepared.Input.Description,
			revision.prepared.Hash[:],
		)
		require.NoError(t, err)
		require.NoError(
			t,
			persistence.ReplaceDraftTree(
				ctx,
				tx,
				checklistID,
				revision.prepared,
			),
		)
		if revision.state != "draft" {
			_, err = tx.ExecContext(
				ctx,
				`UPDATE user_pmcs_revisions
				 SET state = $1,
				     revision_number = $2,
				     published_at = $3
				 WHERE id = $4`,
				revision.state,
				revisionNumber,
				publishedAt,
				revision.prepared.Input.ID,
			)
			require.NoError(t, err)
		}
	}
	require.NoError(t, tx.Commit())
}

func insertDeltaCommunitySource(
	t *testing.T,
	checklistID uuid.UUID,
	revisionID uuid.UUID,
	revisionNumber int32,
) time.Time {
	t.Helper()
	releasedAt := time.Now().UTC().Truncate(time.Microsecond)
	tx, err := testDB.BeginTx(context.Background(), nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = tx.Rollback() })
	_, err = tx.ExecContext(
		context.Background(),
		`INSERT INTO user_pmcs_community_releases
		     (revision_id, checklist_id, released_at)
		 VALUES ($1, $2, $3)`,
		revisionID,
		checklistID,
		releasedAt,
	)
	require.NoError(t, err)
	_, err = tx.ExecContext(
		context.Background(),
		`INSERT INTO user_pmcs_community_sources
		     (checklist_id, status, current_release_revision_id,
		      latest_release_revision_number, first_released_at, updated_at)
		 VALUES ($1, 'active', $2, $3, $4, $4)`,
		checklistID,
		revisionID,
		revisionNumber,
		releasedAt,
	)
	require.NoError(t, err)
	require.NoError(t, tx.Commit())
	return releasedAt
}

func insertDeltaSubscription(
	t *testing.T,
	subscriberUID string,
	checklistID uuid.UUID,
	installedRevisionID *uuid.UUID,
	accountVersion int64,
	deletedAt *time.Time,
) {
	t.Helper()
	_, err := testDB.ExecContext(
		context.Background(),
		`INSERT INTO user_pmcs_subscriptions
		     (subscriber_uid, checklist_id, installed_revision_id,
		      sync_version, account_change_version, deleted_at)
		 VALUES ($1, $2, $3, $4, $4, $5)`,
		subscriberUID,
		checklistID,
		installedRevisionID,
		accountVersion,
		deletedAt,
	)
	require.NoError(t, err)
}

func setDeltaAccountVersion(t *testing.T, userUID string, version int64) {
	t.Helper()
	_, err := testDB.ExecContext(
		context.Background(),
		`INSERT INTO user_pmcs_sync_state (user_uid, current_version)
		 VALUES ($1, $2)
		 ON CONFLICT (user_uid) DO UPDATE
		 SET current_version = EXCLUDED.current_version,
		     updated_at = now()`,
		userUID,
		version,
	)
	require.NoError(t, err)
}

func accountChangeVersions(changes []userpmcssync.AccountChange) []int64 {
	versions := make([]int64, 0, len(changes))
	for _, change := range changes {
		versions = append(versions, change.AccountChangeVersion)
	}
	return versions
}

func waitForDeltaQuery(
	ctx context.Context,
	pattern string,
) error {
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		var count int
		err := testDB.QueryRowContext(
			ctx,
			`SELECT count(*) FROM pg_stat_activity WHERE query LIKE $1`,
			"%"+strings.ReplaceAll(pattern, "%", "%%")+"%",
		).Scan(&count)
		if err == nil && count > 0 {
			return nil
		}
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("inspect account delta query: %w", err)
		}
		time.Sleep(20 * time.Millisecond)
	}
	return fmt.Errorf("account delta query did not reach barrier")
}
