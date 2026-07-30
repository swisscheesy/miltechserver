package user_pmcs_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"miltechserver/api/user_pmcs/community"
	"miltechserver/api/user_pmcs/persistence"
	"miltechserver/api/user_pmcs/shared"
	"miltechserver/api/user_pmcs/subscriptions"
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

func TestSubscriptionUnsubscribeIsIdempotentAndPinnedReadSurvivesRetention(t *testing.T) {
	fixture := newReleasedChecklistFixture(t, 1)
	released, err := fixture.repository.Release(context.Background(), fixture.ownerUID, fixture.checklist, fixture.revisions[0].Input.ID, checklistPrecondition(fixture.checklist, fixture.aggregate.SyncVersion))
	require.NoError(t, err)
	subscriberUID := newUserPmcsTestUser(t)
	repository := subscriptions.NewRepository(persistence.NewStore(testDB, 3), shared.DefaultConfig())
	installed, err := repository.Install(context.Background(), subscriberUID, fixture.checklist, shared.Precondition{Mode: shared.PreconditionCreate})
	require.NoError(t, err)
	_, err = fixture.repository.Retire(context.Background(), fixture.ownerUID, fixture.checklist, checklistPrecondition(fixture.checklist, released.Aggregate.SyncVersion))
	require.NoError(t, err)
	pinned, err := repository.GetInstalledRelease(context.Background(), subscriberUID, fixture.checklist, fixture.revisions[0].Input.ID)
	require.NoError(t, err)
	require.Equal(t, "retired", pinned.SourceStatus)
	_, err = testDB.ExecContext(context.Background(), `UPDATE users SET username = NULL WHERE uid = $1`, fixture.ownerUID)
	require.NoError(t, err)
	pinned, err = repository.GetInstalledRelease(context.Background(), subscriberUID, fixture.checklist, fixture.revisions[0].Input.ID)
	require.NoError(t, err)
	require.Equal(t, "Deleted user", pinned.CreatorDisplayName)
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
