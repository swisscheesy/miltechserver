package user_pmcs_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"miltechserver/api/user_pmcs/community"
	"miltechserver/api/user_pmcs/persistence"
	"miltechserver/api/user_pmcs/shared"
	"miltechserver/api/user_pmcs/subscriptions"
	"miltechserver/bootstrap"
)

func TestSubscriptionInstallUnsubscribeResubscribeAndPinnedRead(t *testing.T) {
	fixture := newReleasedChecklistFixture(t, 1)
	_, err := fixture.repository.Release(context.Background(), fixture.ownerUID, fixture.checklist, fixture.revisions[0].Input.ID, checklistPrecondition(fixture.checklist, fixture.aggregate.SyncVersion))
	require.NoError(t, err)
	subscriberUID := newUserPmcsTestUser(t)
	repository := subscriptions.NewRepository(persistence.NewStore(testDB, 3), shared.DefaultConfig())

	installed, err := repository.Install(context.Background(), subscriberUID, fixture.checklist, shared.Precondition{Mode: shared.PreconditionCreate})
	require.NoError(t, err)
	require.True(t, installed.Created)
	require.Equal(t, fixture.revisions[0].Input.ID, *installed.Subscription.InstalledRevisionID)

	_, err = repository.GetInstalledRelease(context.Background(), subscriberUID, fixture.checklist, uuid.New())
	requireAPIIntegrationError(t, err, 404, "resource_not_found")
	pinned, err := repository.GetInstalledRelease(context.Background(), subscriberUID, fixture.checklist, fixture.revisions[0].Input.ID)
	require.NoError(t, err)
	require.Equal(t, fixture.revisions[0].Input.ID, pinned.Revision.ID)

	unsubscribed, err := repository.Unsubscribe(context.Background(), subscriberUID, fixture.checklist, shared.Precondition{Mode: shared.PreconditionMatch, ETag: shared.MakeSubscriptionETag(fixture.checklist, installed.Subscription.SyncVersion)})
	require.NoError(t, err)
	require.NotNil(t, unsubscribed.Subscription.DeletedAt)
	require.Nil(t, unsubscribed.Subscription.InstalledRevisionID)

	_, err = repository.Install(context.Background(), subscriberUID, fixture.checklist, shared.Precondition{Mode: shared.PreconditionCreate})
	requireAPIIntegrationError(t, err, 412, "stale_precondition")
	resubscribed, err := repository.Install(context.Background(), subscriberUID, fixture.checklist, shared.Precondition{Mode: shared.PreconditionMatch, ETag: shared.MakeSubscriptionETag(fixture.checklist, unsubscribed.Subscription.SyncVersion)})
	require.NoError(t, err)
	require.Equal(t, fixture.revisions[0].Input.ID, *resubscribed.Subscription.InstalledRevisionID)
}

func TestSubscriptionInstallRejectsOwnerAndRetiredSource(t *testing.T) {
	fixture := newReleasedChecklistFixture(t, 1)
	released, err := fixture.repository.Release(context.Background(), fixture.ownerUID, fixture.checklist, fixture.revisions[0].Input.ID, checklistPrecondition(fixture.checklist, fixture.aggregate.SyncVersion))
	require.NoError(t, err)
	repository := subscriptions.NewRepository(persistence.NewStore(testDB, 3), shared.DefaultConfig())
	_, err = repository.Install(context.Background(), fixture.ownerUID, fixture.checklist, shared.Precondition{Mode: shared.PreconditionCreate})
	requireAPIIntegrationError(t, err, 409, "invalid_transition")
	_, err = fixture.repository.Retire(context.Background(), fixture.ownerUID, fixture.checklist, checklistPrecondition(fixture.checklist, released.Aggregate.SyncVersion))
	require.NoError(t, err)
	_, err = repository.Install(context.Background(), newUserPmcsTestUser(t), fixture.checklist, shared.Precondition{Mode: shared.PreconditionCreate})
	requireAPIIntegrationError(t, err, 409, "invalid_transition")
}

func TestSubscriptionInstallPinsCurrentReleaseAndRetriesIdempotently(t *testing.T) {
	fixture := newReleasedChecklistFixture(t, 2)
	first, err := fixture.repository.Release(context.Background(), fixture.ownerUID, fixture.checklist, fixture.revisions[0].Input.ID, checklistPrecondition(fixture.checklist, fixture.aggregate.SyncVersion))
	require.NoError(t, err)
	second, err := fixture.repository.Release(context.Background(), fixture.ownerUID, fixture.checklist, fixture.revisions[1].Input.ID, checklistPrecondition(fixture.checklist, first.Aggregate.SyncVersion))
	require.NoError(t, err)
	_ = second
	repository := subscriptions.NewRepository(persistence.NewStore(testDB, 3), shared.DefaultConfig())
	subscriberUID := newUserPmcsTestUser(t)
	installed, err := repository.Install(context.Background(), subscriberUID, fixture.checklist, shared.Precondition{Mode: shared.PreconditionCreate})
	require.NoError(t, err)
	require.Equal(t, fixture.revisions[1].Input.ID, *installed.Subscription.InstalledRevisionID)
	retried, err := repository.Install(context.Background(), subscriberUID, fixture.checklist, shared.Precondition{Mode: shared.PreconditionCreate})
	require.NoError(t, err)
	require.True(t, retried.Idempotent)
	require.Equal(t, installed.Subscription.SyncVersion, retried.Subscription.SyncVersion)
}

func TestSubscriptionInstallRequiresMatchingActiveIfMatch(t *testing.T) {
	fixture := newReleasedChecklistFixture(t, 1)
	_, err := fixture.repository.Release(context.Background(), fixture.ownerUID, fixture.checklist, fixture.revisions[0].Input.ID, checklistPrecondition(fixture.checklist, fixture.aggregate.SyncVersion))
	require.NoError(t, err)
	repository := subscriptions.NewRepository(persistence.NewStore(testDB, 1), shared.DefaultConfig())
	subscriberUID := newUserPmcsTestUser(t)
	installed, err := repository.Install(context.Background(), subscriberUID, fixture.checklist, shared.Precondition{Mode: shared.PreconditionCreate})
	require.NoError(t, err)
	matched, err := repository.Install(context.Background(), subscriberUID, fixture.checklist, shared.Precondition{Mode: shared.PreconditionMatch, ETag: shared.MakeSubscriptionETag(fixture.checklist, installed.Subscription.SyncVersion)})
	require.NoError(t, err)
	require.True(t, matched.Idempotent)
	_, err = repository.Install(context.Background(), subscriberUID, fixture.checklist, shared.Precondition{Mode: shared.PreconditionMatch, ETag: `"stale"`})
	requireAPIIntegrationError(t, err, 412, "stale_precondition")
}

func TestSubscriptionInstallAndReleaseUseConsistentPostgresLockOrder(t *testing.T) {
	fixture := newReleasedChecklistFixture(t, 1)
	released, err := fixture.repository.Release(context.Background(), fixture.ownerUID, fixture.checklist, fixture.revisions[0].Input.ID, checklistPrecondition(fixture.checklist, fixture.aggregate.SyncVersion))
	require.NoError(t, err)
	repository := subscriptions.NewRepository(persistence.NewStore(testDB, 1), shared.DefaultConfig())
	subscriberUIDs := make([]string, 20)
	for index := range subscriberUIDs {
		subscriberUIDs[index] = newUserPmcsTestUser(t)
	}

	for attempt := 0; attempt < 20; attempt++ {
		attempt := attempt
		start := make(chan struct{})
		results := make(chan error, 2)
		var waitGroup sync.WaitGroup
		waitGroup.Add(2)
		go func() {
			defer waitGroup.Done()
			<-start
			_, installErr := repository.Install(context.Background(), subscriberUIDs[attempt], fixture.checklist, shared.Precondition{Mode: shared.PreconditionCreate})
			results <- installErr
		}()
		go func() {
			defer waitGroup.Done()
			<-start
			_, releaseErr := fixture.repository.Release(context.Background(), fixture.ownerUID, fixture.checklist, fixture.revisions[0].Input.ID, checklistPrecondition(fixture.checklist, released.Aggregate.SyncVersion))
			results <- releaseErr
		}()
		close(start)
		completed := make(chan struct{})
		go func() { waitGroup.Wait(); close(completed) }()
		select {
		case <-completed:
		case <-time.After(5 * time.Second):
			t.Fatalf("attempt %d did not complete", attempt)
		}
		close(results)
		for result := range results {
			require.NoError(t, result)
		}
	}
}

func TestSubscriptionUnsubscribeIsIdempotentAndPinnedReadSurvivesRetention(t *testing.T) {
	fixture := newReleasedChecklistFixture(t, 1)
	released, err := fixture.repository.Release(context.Background(), fixture.ownerUID, fixture.checklist, fixture.revisions[0].Input.ID, checklistPrecondition(fixture.checklist, fixture.aggregate.SyncVersion))
	require.NoError(t, err)
	subscriberUID := newUserPmcsTestUser(t)
	repository := subscriptions.NewRepository(persistence.NewStore(testDB, 3), shared.DefaultConfig())
	service := subscriptions.NewService(repository, shared.DefaultConfig())
	installed, err := repository.Install(context.Background(), subscriberUID, fixture.checklist, shared.Precondition{Mode: shared.PreconditionCreate})
	require.NoError(t, err)
	user := &bootstrap.User{UserID: subscriberUID}
	beforeRetirement, beforeRetirementETag, err := service.GetInstalledRelease(context.Background(), user, fixture.checklist.String(), fixture.revisions[0].Input.ID.String())
	require.NoError(t, err)
	require.Equal(t, "active", beforeRetirement.SourceStatus)
	_, err = fixture.repository.Retire(context.Background(), fixture.ownerUID, fixture.checklist, checklistPrecondition(fixture.checklist, released.Aggregate.SyncVersion))
	require.NoError(t, err)
	t.Cleanup(func() {
		_, cleanupErr := testDB.ExecContext(context.Background(), `DELETE FROM user_pmcs_subscriptions WHERE checklist_id = $1`, fixture.checklist)
		require.NoError(t, cleanupErr)
		_, cleanupErr = testDB.ExecContext(context.Background(), `DELETE FROM user_pmcs_community_sources WHERE checklist_id = $1`, fixture.checklist)
		require.NoError(t, cleanupErr)
		_, cleanupErr = testDB.ExecContext(context.Background(), `DELETE FROM user_pmcs_community_releases WHERE checklist_id = $1`, fixture.checklist)
		require.NoError(t, cleanupErr)
		_, cleanupErr = testDB.ExecContext(context.Background(), `DELETE FROM user_pmcs_revisions WHERE checklist_id = $1`, fixture.checklist)
		require.NoError(t, cleanupErr)
		_, cleanupErr = testDB.ExecContext(context.Background(), `DELETE FROM user_pmcs_checklists WHERE id = $1`, fixture.checklist)
		require.NoError(t, cleanupErr)
	})
	pinned, err := repository.GetInstalledRelease(context.Background(), subscriberUID, fixture.checklist, fixture.revisions[0].Input.ID)
	require.NoError(t, err)
	require.Equal(t, "retired", pinned.SourceStatus)
	afterRetirement, afterRetirementETag, err := service.GetInstalledRelease(context.Background(), user, fixture.checklist.String(), fixture.revisions[0].Input.ID.String())
	require.NoError(t, err)
	require.Equal(t, "retired", afterRetirement.SourceStatus)
	require.NotEqual(t, beforeRetirementETag, afterRetirementETag)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	group := router.Group("/api/v1/auth")
	group.Use(func(context *gin.Context) { context.Set("user", user); context.Next() })
	subscriptions.RegisterRoutes(group, service)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/auth/user-pmcs/subscriptions/"+fixture.checklist.String()+"/installed-releases/"+fixture.revisions[0].Input.ID.String(), nil)
	request.Header.Set("If-None-Match", beforeRetirementETag)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	require.Equal(t, http.StatusOK, response.Code)
	require.Contains(t, response.Body.String(), `"source_status":"retired"`)
	_, err = testDB.ExecContext(context.Background(), `UPDATE users SET username = $1 WHERE uid = $2`, "Renamed creator", fixture.ownerUID)
	require.NoError(t, err)
	updatedCreator, updatedCreatorETag, err := service.GetInstalledRelease(context.Background(), user, fixture.checklist.String(), fixture.revisions[0].Input.ID.String())
	require.NoError(t, err)
	require.Equal(t, "Renamed creator", updatedCreator.CreatorDisplayName)
	require.NotEqual(t, afterRetirementETag, updatedCreatorETag)
	_, err = testDB.ExecContext(context.Background(), `UPDATE user_pmcs_checklists SET owner_uid = NULL, deleted_at = now() WHERE id = $1`, fixture.checklist)
	require.NoError(t, err)
	_, err = testDB.ExecContext(context.Background(), `DELETE FROM users WHERE uid = $1`, fixture.ownerUID)
	require.NoError(t, err)
	pinned, err = repository.GetInstalledRelease(context.Background(), subscriberUID, fixture.checklist, fixture.revisions[0].Input.ID)
	require.NoError(t, err)
	require.Equal(t, "Deleted user", pinned.CreatorDisplayName)
	anonymized, anonymizedETag, err := service.GetInstalledRelease(context.Background(), user, fixture.checklist.String(), fixture.revisions[0].Input.ID.String())
	require.NoError(t, err)
	require.Equal(t, "Deleted user", anonymized.CreatorDisplayName)
	require.NotEqual(t, updatedCreatorETag, anonymizedETag)
	unsubscribed, err := repository.Unsubscribe(context.Background(), subscriberUID, fixture.checklist, shared.Precondition{Mode: shared.PreconditionMatch, ETag: shared.MakeSubscriptionETag(fixture.checklist, installed.Subscription.SyncVersion)})
	require.NoError(t, err)
	retried, err := repository.Unsubscribe(context.Background(), subscriberUID, fixture.checklist, shared.Precondition{Mode: shared.PreconditionMatch, ETag: `"stale"`})
	require.NoError(t, err)
	require.True(t, retried.Idempotent)
	require.Equal(t, unsubscribed.Subscription.SyncVersion, retried.Subscription.SyncVersion)
}

func TestSubscriptionInstallEnforcesConfiguredActiveCeiling(t *testing.T) {
	firstFixture := newReleasedChecklistFixture(t, 1)
	_, err := firstFixture.repository.Release(context.Background(), firstFixture.ownerUID, firstFixture.checklist, firstFixture.revisions[0].Input.ID, checklistPrecondition(firstFixture.checklist, firstFixture.aggregate.SyncVersion))
	require.NoError(t, err)
	secondFixture := newReleasedChecklistFixture(t, 1)
	_, err = secondFixture.repository.Release(context.Background(), secondFixture.ownerUID, secondFixture.checklist, secondFixture.revisions[0].Input.ID, checklistPrecondition(secondFixture.checklist, secondFixture.aggregate.SyncVersion))
	require.NoError(t, err)
	config := shared.DefaultConfig()
	config.MaxActiveSubscriptions = 1
	repository := subscriptions.NewRepository(persistence.NewStore(testDB, 3), config)
	subscriberUID := newUserPmcsTestUser(t)
	_, err = repository.Install(context.Background(), subscriberUID, firstFixture.checklist, shared.Precondition{Mode: shared.PreconditionCreate})
	require.NoError(t, err)
	_, err = repository.Install(context.Background(), subscriberUID, secondFixture.checklist, shared.Precondition{Mode: shared.PreconditionCreate})
	requireAPIIntegrationError(t, err, 409, "invalid_transition")
}

var _ = community.Repository(nil)
