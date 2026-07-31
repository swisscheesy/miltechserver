package user_pmcs_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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
	pmcssync "miltechserver/api/user_pmcs/sync"
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

func TestSubscriptionAcceptUpdateHidesMissingOrDeletedSubscriptionBeforeSourceState(
	t *testing.T,
) {
	ctx := context.Background()
	repository := subscriptions.NewRepository(
		persistence.NewStore(testDB, 3),
		shared.DefaultConfig(),
	)

	t.Run("unknown source and missing subscription", func(t *testing.T) {
		subscriberUID := newUserPmcsTestUser(t)
		checklistID := uuid.New()
		_, err := repository.AcceptUpdate(
			ctx,
			subscriberUID,
			checklistID,
			uuid.New(),
			shared.Precondition{
				Mode: shared.PreconditionMatch,
				ETag: `"unknown"`,
			},
		)
		requireAPIIntegrationError(t, err, 404, "resource_not_found")
	})

	t.Run("retired source and deleted subscription", func(t *testing.T) {
		fixture := newReleasedChecklistFixture(t, 1)
		released, err := fixture.repository.Release(
			ctx,
			fixture.ownerUID,
			fixture.checklist,
			fixture.revisions[0].Input.ID,
			checklistPrecondition(
				fixture.checklist,
				fixture.aggregate.SyncVersion,
			),
		)
		require.NoError(t, err)
		subscriberUID := newUserPmcsTestUser(t)
		installed, err := repository.Install(
			ctx,
			subscriberUID,
			fixture.checklist,
			shared.Precondition{Mode: shared.PreconditionCreate},
		)
		require.NoError(t, err)
		unsubscribed, err := repository.Unsubscribe(
			ctx,
			subscriberUID,
			fixture.checklist,
			shared.Precondition{
				Mode: shared.PreconditionMatch,
				ETag: shared.MakeSubscriptionETag(
					fixture.checklist,
					installed.Subscription.SyncVersion,
				),
			},
		)
		require.NoError(t, err)
		_, err = fixture.repository.Retire(
			ctx,
			fixture.ownerUID,
			fixture.checklist,
			checklistPrecondition(
				fixture.checklist,
				released.Aggregate.SyncVersion,
			),
		)
		require.NoError(t, err)

		_, err = repository.AcceptUpdate(
			ctx,
			subscriberUID,
			fixture.checklist,
			fixture.revisions[0].Input.ID,
			shared.Precondition{
				Mode: shared.PreconditionMatch,
				ETag: shared.MakeSubscriptionETag(
					fixture.checklist,
					unsubscribed.Subscription.SyncVersion,
				),
			},
		)
		requireAPIIntegrationError(t, err, 404, "resource_not_found")
	})

	t.Run("retired source with active subscription stays a transition error", func(t *testing.T) {
		fixture := newReleasedChecklistFixture(t, 1)
		released, err := fixture.repository.Release(
			ctx,
			fixture.ownerUID,
			fixture.checklist,
			fixture.revisions[0].Input.ID,
			checklistPrecondition(
				fixture.checklist,
				fixture.aggregate.SyncVersion,
			),
		)
		require.NoError(t, err)
		subscriberUID := newUserPmcsTestUser(t)
		installed, err := repository.Install(
			ctx,
			subscriberUID,
			fixture.checklist,
			shared.Precondition{Mode: shared.PreconditionCreate},
		)
		require.NoError(t, err)
		_, err = fixture.repository.Retire(
			ctx,
			fixture.ownerUID,
			fixture.checklist,
			checklistPrecondition(
				fixture.checklist,
				released.Aggregate.SyncVersion,
			),
		)
		require.NoError(t, err)

		_, err = repository.AcceptUpdate(
			ctx,
			subscriberUID,
			fixture.checklist,
			fixture.revisions[0].Input.ID,
			shared.Precondition{
				Mode: shared.PreconditionMatch,
				ETag: shared.MakeSubscriptionETag(
					fixture.checklist,
					installed.Subscription.SyncVersion,
				),
			},
		)
		requireAPIIntegrationError(t, err, 409, "invalid_transition")
	})
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

func TestSubscriptionUpdateDiscoveryAndAcceptance(t *testing.T) {
	fixture := newReleasedChecklistFixture(t, 2)
	firstRelease, err := fixture.repository.Release(context.Background(), fixture.ownerUID, fixture.checklist, fixture.revisions[0].Input.ID, checklistPrecondition(fixture.checklist, fixture.aggregate.SyncVersion))
	require.NoError(t, err)
	subscriberUID := newUserPmcsTestUser(t)
	repository := subscriptions.NewRepository(persistence.NewStore(testDB, 3), shared.DefaultConfig())
	installed, err := repository.Install(context.Background(), subscriberUID, fixture.checklist, shared.Precondition{Mode: shared.PreconditionCreate})
	require.NoError(t, err)
	_, err = fixture.repository.Release(context.Background(), fixture.ownerUID, fixture.checklist, fixture.revisions[1].Input.ID, checklistPrecondition(fixture.checklist, firstRelease.Aggregate.SyncVersion))
	require.NoError(t, err)
	beforeDiscoveryVersion := installed.Subscription.SyncVersion
	beforeDiscoveryAccountVersion := installed.Subscription.AccountChangeVersion

	updates, err := repository.ListUpdates(context.Background(), subscriberUID, nil, 50)
	require.NoError(t, err)
	require.Len(t, updates.Items, 1)
	require.Equal(t, fixture.checklist, updates.Items[0].ChecklistID)
	require.Equal(t, fixture.revisions[0].Input.ID, updates.Items[0].InstalledRevisionID)
	require.Equal(t, int32(1), updates.Items[0].InstalledRevisionNumber)
	require.Equal(t, fixture.revisions[1].Input.ID, *updates.Items[0].CurrentReleaseRevisionID)
	require.Equal(t, int32(2), *updates.Items[0].CurrentReleaseNumber)
	require.True(t, updates.Items[0].UpdateAvailable)
	var discoverySyncVersion, discoveryAccountVersion int64
	err = testDB.QueryRowContext(context.Background(), `SELECT sync_version, account_change_version FROM user_pmcs_subscriptions WHERE subscriber_uid = $1 AND checklist_id = $2`, subscriberUID, fixture.checklist).Scan(&discoverySyncVersion, &discoveryAccountVersion)
	require.NoError(t, err)
	require.Equal(t, beforeDiscoveryVersion, discoverySyncVersion)
	require.Equal(t, beforeDiscoveryAccountVersion, discoveryAccountVersion)

	gateway := gin.New()
	group := gateway.Group("/api/v1/auth")
	group.Use(func(context *gin.Context) {
		context.Set("user", &bootstrap.User{UserID: subscriberUID})
		context.Next()
	})
	service := subscriptions.NewService(repository, shared.DefaultConfig())
	subscriptions.RegisterRoutes(group, service)

	updatesRequest := httptest.NewRequest(http.MethodGet, "/api/v1/auth/user-pmcs/subscriptions/updates?limit=1", nil)
	updatesResponse := httptest.NewRecorder()
	gateway.ServeHTTP(updatesResponse, updatesRequest)
	require.Equal(t, http.StatusOK, updatesResponse.Code)
	require.Contains(t, updatesResponse.Body.String(), fixture.revisions[1].Input.ID.String())
	require.Contains(t, updatesResponse.Body.String(), `"update_available":true`)
	require.NotContains(t, updatesResponse.Body.String(), `"sections"`)

	acceptRequest := httptest.NewRequest(http.MethodPut, "/api/v1/auth/user-pmcs/subscriptions/"+fixture.checklist.String()+"/installed-releases/"+fixture.revisions[1].Input.ID.String(), nil)
	acceptRequest.Header.Set("If-Match", shared.MakeSubscriptionETag(fixture.checklist, installed.Subscription.SyncVersion))
	acceptResponse := httptest.NewRecorder()
	gateway.ServeHTTP(acceptResponse, acceptRequest)
	require.Equal(t, http.StatusOK, acceptResponse.Code)
	require.Contains(t, acceptResponse.Body.String(), fixture.revisions[1].Input.ID.String())

	accepted, err := repository.AcceptUpdate(context.Background(), subscriberUID, fixture.checklist, fixture.revisions[1].Input.ID, shared.Precondition{Mode: shared.PreconditionMatch, ETag: `"stale retry"`})
	require.NoError(t, err)
	require.True(t, accepted.Idempotent)
	require.Equal(t, fixture.revisions[1].Input.ID, *accepted.Subscription.InstalledRevisionID)

	delta, err := pmcssync.NewRepository(persistence.NewStore(testDB, 3)).GetDelta(context.Background(), subscriberUID, beforeDiscoveryAccountVersion, 10, shared.DefaultConfig().MaxDeltaResponseBytes)
	require.NoError(t, err)
	require.Len(t, delta.Changes, 1)
	require.Equal(t, fixture.revisions[1].Input.ID, *delta.Changes[0].Subscription.InstalledRevisionID)
	require.NotNil(t, delta.Changes[0].Installed)
	require.Equal(t, fixture.revisions[1].Input.ID, delta.Changes[0].Installed.Revision.ID)

	_, err = repository.AcceptUpdate(context.Background(), subscriberUID, fixture.checklist, fixture.revisions[0].Input.ID, shared.Precondition{Mode: shared.PreconditionMatch, ETag: shared.MakeSubscriptionETag(fixture.checklist, accepted.Subscription.SyncVersion)})
	requireAPIIntegrationError(t, err, 409, "invalid_transition")
	pinned, err := repository.GetInstalledRelease(context.Background(), subscriberUID, fixture.checklist, fixture.revisions[1].Input.ID)
	require.NoError(t, err)
	require.Equal(t, fixture.revisions[1].Input.ID, pinned.Revision.ID)
}

func TestSubscriptionUpdateDiscoveryUsesChecklistUUIDKeysetAndReportsRetiredSources(t *testing.T) {
	firstFixture := newReleasedChecklistFixture(t, 1)
	firstRelease, err := firstFixture.repository.Release(context.Background(), firstFixture.ownerUID, firstFixture.checklist, firstFixture.revisions[0].Input.ID, checklistPrecondition(firstFixture.checklist, firstFixture.aggregate.SyncVersion))
	require.NoError(t, err)
	secondFixture := newReleasedChecklistFixture(t, 1)
	_, err = secondFixture.repository.Release(context.Background(), secondFixture.ownerUID, secondFixture.checklist, secondFixture.revisions[0].Input.ID, checklistPrecondition(secondFixture.checklist, secondFixture.aggregate.SyncVersion))
	require.NoError(t, err)
	subscriberUID := newUserPmcsTestUser(t)
	repository := subscriptions.NewRepository(persistence.NewStore(testDB, 3), shared.DefaultConfig())
	_, err = repository.Install(context.Background(), subscriberUID, firstFixture.checklist, shared.Precondition{Mode: shared.PreconditionCreate})
	require.NoError(t, err)
	_, err = repository.Install(context.Background(), subscriberUID, secondFixture.checklist, shared.Precondition{Mode: shared.PreconditionCreate})
	require.NoError(t, err)
	_, err = firstFixture.repository.Retire(context.Background(), firstFixture.ownerUID, firstFixture.checklist, checklistPrecondition(firstFixture.checklist, firstRelease.Aggregate.SyncVersion))
	require.NoError(t, err)

	firstPage, err := repository.ListUpdates(context.Background(), subscriberUID, nil, 1)
	require.NoError(t, err)
	require.Len(t, firstPage.Items, 1)
	require.True(t, firstPage.HasMore)
	require.NotNil(t, firstPage.NextCursor)
	firstCursor, err := shared.DecodeSubscriptionUpdateCursor(*firstPage.NextCursor)
	require.NoError(t, err)
	require.Equal(t, firstPage.Items[0].ChecklistID, firstCursor.Checklist)
	if firstPage.Items[0].SourceStatus == "retired" {
		require.Nil(t, firstPage.Items[0].CurrentReleaseRevisionID)
		require.False(t, firstPage.Items[0].UpdateAvailable)
	}

	secondPage, err := repository.ListUpdates(context.Background(), subscriberUID, &firstCursor.Checklist, 1)
	require.NoError(t, err)
	require.Len(t, secondPage.Items, 1)
	require.Greater(t, secondPage.Items[0].ChecklistID.String(), firstPage.Items[0].ChecklistID.String())
	for _, item := range append(firstPage.Items, secondPage.Items...) {
		if item.ChecklistID == firstFixture.checklist {
			require.Equal(t, "retired", item.SourceStatus)
			require.Nil(t, item.CurrentReleaseRevisionID)
			require.False(t, item.UpdateAvailable)
		}
	}
}

func TestSubscriptionUpdateDiscoveryReportsActiveSubscriptionWithoutUpdateWithoutMutation(t *testing.T) {
	fixture := newReleasedChecklistFixture(t, 1)
	_, err := fixture.repository.Release(context.Background(), fixture.ownerUID, fixture.checklist, fixture.revisions[0].Input.ID, checklistPrecondition(fixture.checklist, fixture.aggregate.SyncVersion))
	require.NoError(t, err)
	subscriberUID := newUserPmcsTestUser(t)
	repository := subscriptions.NewRepository(persistence.NewStore(testDB, 3), shared.DefaultConfig())
	installed, err := repository.Install(context.Background(), subscriberUID, fixture.checklist, shared.Precondition{Mode: shared.PreconditionCreate})
	require.NoError(t, err)
	var accountVersion int64
	err = testDB.QueryRowContext(context.Background(), `SELECT current_version FROM user_pmcs_sync_state WHERE user_uid = $1`, subscriberUID).Scan(&accountVersion)
	require.NoError(t, err)

	page, err := repository.ListUpdates(context.Background(), subscriberUID, nil, 50)
	require.NoError(t, err)
	require.Len(t, page.Items, 1)
	item := page.Items[0]
	require.Equal(t, fixture.checklist, item.ChecklistID)
	require.Equal(t, "active", item.SourceStatus)
	require.Equal(t, fixture.revisions[0].Input.ID, item.InstalledRevisionID)
	require.Equal(t, int32(1), item.InstalledRevisionNumber)
	require.NotNil(t, item.CurrentReleaseRevisionID)
	require.Equal(t, fixture.revisions[0].Input.ID, *item.CurrentReleaseRevisionID)
	require.NotNil(t, item.CurrentReleaseNumber)
	require.Equal(t, int32(1), *item.CurrentReleaseNumber)
	require.False(t, item.UpdateAvailable)
	payload, err := json.Marshal(item)
	require.NoError(t, err)
	require.NotContains(t, string(payload), `"sections"`)

	var syncVersion, changeVersion, currentAccountVersion int64
	err = testDB.QueryRowContext(context.Background(), `SELECT sync_version, account_change_version FROM user_pmcs_subscriptions WHERE subscriber_uid = $1 AND checklist_id = $2`, subscriberUID, fixture.checklist).Scan(&syncVersion, &changeVersion)
	require.NoError(t, err)
	require.Equal(t, installed.Subscription.SyncVersion, syncVersion)
	require.Equal(t, installed.Subscription.AccountChangeVersion, changeVersion)
	err = testDB.QueryRowContext(context.Background(), `SELECT current_version FROM user_pmcs_sync_state WHERE user_uid = $1`, subscriberUID).Scan(&currentAccountVersion)
	require.NoError(t, err)
	require.Equal(t, accountVersion, currentAccountVersion)
}

func TestSubscriptionAcceptUpdateRejectsStaleETagWithoutMutation(t *testing.T) {
	fixture := newReleasedChecklistFixture(t, 2)
	firstRelease, err := fixture.repository.Release(context.Background(), fixture.ownerUID, fixture.checklist, fixture.revisions[0].Input.ID, checklistPrecondition(fixture.checklist, fixture.aggregate.SyncVersion))
	require.NoError(t, err)
	subscriberUID := newUserPmcsTestUser(t)
	repository := subscriptions.NewRepository(persistence.NewStore(testDB, 3), shared.DefaultConfig())
	installed, err := repository.Install(context.Background(), subscriberUID, fixture.checklist, shared.Precondition{Mode: shared.PreconditionCreate})
	require.NoError(t, err)
	_, err = fixture.repository.Release(context.Background(), fixture.ownerUID, fixture.checklist, fixture.revisions[1].Input.ID, checklistPrecondition(fixture.checklist, firstRelease.Aggregate.SyncVersion))
	require.NoError(t, err)
	var accountVersion int64
	err = testDB.QueryRowContext(context.Background(), `SELECT current_version FROM user_pmcs_sync_state WHERE user_uid = $1`, subscriberUID).Scan(&accountVersion)
	require.NoError(t, err)

	_, err = repository.AcceptUpdate(context.Background(), subscriberUID, fixture.checklist, fixture.revisions[1].Input.ID, shared.Precondition{Mode: shared.PreconditionMatch, ETag: `"stale"`})
	requireAPIIntegrationError(t, err, 412, "stale_precondition")

	var installedRevisionID uuid.UUID
	var syncVersion, changeVersion, currentAccountVersion int64
	err = testDB.QueryRowContext(context.Background(), `SELECT installed_revision_id, sync_version, account_change_version FROM user_pmcs_subscriptions WHERE subscriber_uid = $1 AND checklist_id = $2`, subscriberUID, fixture.checklist).Scan(&installedRevisionID, &syncVersion, &changeVersion)
	require.NoError(t, err)
	require.Equal(t, fixture.revisions[0].Input.ID, installedRevisionID)
	require.Equal(t, installed.Subscription.SyncVersion, syncVersion)
	require.Equal(t, installed.Subscription.AccountChangeVersion, changeVersion)
	err = testDB.QueryRowContext(context.Background(), `SELECT current_version FROM user_pmcs_sync_state WHERE user_uid = $1`, subscriberUID).Scan(&currentAccountVersion)
	require.NoError(t, err)
	require.Equal(t, accountVersion, currentAccountVersion)
}

func TestSubscriptionUpdateDiscoveryExplainUsesApprovedIndex(t *testing.T) {
	fixture := seedPerformanceSubscriptionsWithNoise(t, 500, 50)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	rows, err := testDB.QueryContext(ctx, `EXPLAIN (ANALYZE, BUFFERS)
		SELECT subscription.checklist_id, COALESCE(source.status, 'retired'), subscription.installed_revision_id, installed.revision_number, source.current_release_revision_id, current_release.revision_number
		FROM user_pmcs_subscriptions AS subscription
		JOIN user_pmcs_revisions AS installed ON installed.checklist_id = subscription.checklist_id AND installed.id = subscription.installed_revision_id
		LEFT JOIN user_pmcs_community_sources AS source ON source.checklist_id = subscription.checklist_id
		LEFT JOIN user_pmcs_revisions AS current_release ON current_release.checklist_id = source.checklist_id AND current_release.id = source.current_release_revision_id
		WHERE subscription.subscriber_uid = $1 AND subscription.deleted_at IS NULL
		ORDER BY subscription.checklist_id LIMIT 51`, fixture.noiseUserUIDs[0])
	require.NoError(t, err)
	defer rows.Close()
	planLines := make([]string, 0)
	for rows.Next() {
		var line string
		require.NoError(t, rows.Scan(&line))
		planLines = append(planLines, line)
	}
	require.NoError(t, rows.Err())
	plan := strings.Join(planLines, "\n")
	t.Logf(
		"Task 12 EXPLAIN (500 active subscriptions; committed and analyzed fixture):\n%s",
		plan,
	)
	requireSubscriptionUpdatePlanUsesApprovedIndex(t, plan)
	require.Contains(t, plan, "rows=51")
	require.Contains(t, plan, "Buffers:")
	require.Contains(t, plan, "Execution Time:")
}

func requireSubscriptionUpdatePlanUsesApprovedIndex(t *testing.T, plan string) {
	t.Helper()
	require.NotContains(
		t,
		plan,
		"Seq Scan on user_pmcs_subscriptions",
		"subscription update discovery must not scan the subscription relation sequentially",
	)

	approvedIndexes := []string{
		"user_pmcs_subscriptions_active_update_idx",
		"user_pmcs_subscriptions_delta_idx",
		"user_pmcs_subscriptions_pkey",
	}
	for _, indexName := range approvedIndexes {
		if strings.Contains(plan, indexName) {
			return
		}
	}
	require.FailNowf(
		t,
		"subscription update discovery did not use an approved subscription index",
		"approved indexes: %s\nplan:\n%s",
		strings.Join(approvedIndexes, ", "),
		plan,
	)
}

var _ = community.Repository(nil)
