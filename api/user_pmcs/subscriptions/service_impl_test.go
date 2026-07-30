package subscriptions

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"miltechserver/api/user_pmcs/shared"
	"miltechserver/bootstrap"
)

type subscriptionRepositoryStub struct {
	installResult    *MutationResult
	installError     error
	installCondition shared.Precondition
	installedResult  *shared.InstalledChecklistRelease
	updatesResult    *shared.SubscriptionUpdatePage
	updatesAfter     *uuid.UUID
	updatesLimit     int
	acceptResult     *MutationResult
	acceptCondition  shared.Precondition
}

func (stub *subscriptionRepositoryStub) Install(_ context.Context, _ string, _ uuid.UUID, precondition shared.Precondition) (*MutationResult, error) {
	stub.installCondition = precondition
	return stub.installResult, stub.installError
}
func (stub *subscriptionRepositoryStub) Unsubscribe(context.Context, string, uuid.UUID, shared.Precondition) (*MutationResult, error) {
	return nil, nil
}
func (stub *subscriptionRepositoryStub) GetInstalledRelease(context.Context, string, uuid.UUID, uuid.UUID) (*shared.InstalledChecklistRelease, error) {
	return stub.installedResult, nil
}
func (stub *subscriptionRepositoryStub) ListUpdates(_ context.Context, _ string, after *uuid.UUID, limit int) (*shared.SubscriptionUpdatePage, error) {
	stub.updatesAfter = after
	stub.updatesLimit = limit
	return stub.updatesResult, nil
}
func (stub *subscriptionRepositoryStub) AcceptUpdate(_ context.Context, _ string, _ uuid.UUID, _ uuid.UUID, precondition shared.Precondition) (*MutationResult, error) {
	stub.acceptCondition = precondition
	return stub.acceptResult, nil
}

func TestSubscriptionInstallParsesCreatePrecondition(t *testing.T) {
	checklistID := uuid.New()
	repository := &subscriptionRepositoryStub{installResult: &MutationResult{Subscription: shared.Subscription{ChecklistID: checklistID, SyncVersion: 1}}}
	service := NewService(repository, shared.DefaultConfig())

	result, etag, err := service.Install(context.Background(), &bootstrap.User{UserID: "subscriber-1"}, checklistID.String(), "*", "")

	require.NoError(t, err)
	require.Same(t, repository.installResult, result)
	require.Equal(t, shared.Precondition{Mode: shared.PreconditionCreate}, repository.installCondition)
	require.Equal(t, shared.MakeSubscriptionETag(checklistID, 1), etag)
}

func TestSubscriptionResubscribeRequiresExistingPrecondition(t *testing.T) {
	service := NewService(&subscriptionRepositoryStub{}, shared.DefaultConfig())
	_, _, err := service.Install(context.Background(), &bootstrap.User{UserID: "subscriber-1"}, uuid.NewString(), "", "")
	requireAPIError(t, err, 428, "precondition_required")
}

func TestSubscriptionPinnedReadUsesImmutableValidator(t *testing.T) {
	checklistID := uuid.New()
	revisionID := uuid.New()
	repository := &subscriptionRepositoryStub{installedResult: &shared.InstalledChecklistRelease{ChecklistID: checklistID, Revision: shared.Revision{ID: revisionID, Models: []shared.ModelValue{}, Sections: []shared.Section{}}}}
	service := NewService(repository, shared.DefaultConfig())

	_, etag, err := service.GetInstalledRelease(context.Background(), &bootstrap.User{UserID: "subscriber-1"}, checklistID.String(), revisionID.String())
	require.NoError(t, err)
	require.NotEmpty(t, etag)
}

func TestSubscriptionPinnedETagRepresentsLiveMetadataAndCompleteRevision(t *testing.T) {
	checklistID := uuid.New()
	revisionID := uuid.New()
	release := &shared.InstalledChecklistRelease{
		ChecklistID:        checklistID,
		SourceStatus:       "active",
		CreatorDisplayName: "Creator",
		Revision:           shared.Revision{ID: revisionID, Name: "PMCS", Models: []shared.ModelValue{}, Sections: []shared.Section{}},
	}
	first, err := installedReleaseETag(release)
	require.NoError(t, err)
	second, err := installedReleaseETag(release)
	require.NoError(t, err)
	require.Equal(t, first, second)
	require.True(t, len(first) > 2 && first[0] == '"' && first[len(first)-1] == '"')

	retired := *release
	retired.SourceStatus = "retired"
	retiredETag, err := installedReleaseETag(&retired)
	require.NoError(t, err)
	require.NotEqual(t, first, retiredETag)

	renamed := *release
	renamed.CreatorDisplayName = "Renamed creator"
	renamedETag, err := installedReleaseETag(&renamed)
	require.NoError(t, err)
	require.NotEqual(t, first, renamedETag)

	changedTree := *release
	changedTree.Revision.Name = "Changed PMCS"
	changedTreeETag, err := installedReleaseETag(&changedTree)
	require.NoError(t, err)
	require.NotEqual(t, first, changedTreeETag)
}

func TestSubscriptionAcceptUpdateRequiresExistingPrecondition(t *testing.T) {
	service := NewService(&subscriptionRepositoryStub{}, shared.DefaultConfig())
	_, _, err := service.AcceptUpdate(context.Background(), &bootstrap.User{UserID: "subscriber-1"}, uuid.NewString(), uuid.NewString(), "")
	requireAPIError(t, err, 428, "precondition_required")
}

func TestSubscriptionUpdateDiscoveryUsesConfiguredBoundsAndRejectsMalformedCursor(t *testing.T) {
	checklistID := uuid.New()
	repository := &subscriptionRepositoryStub{updatesResult: &shared.SubscriptionUpdatePage{Items: []shared.SubscriptionUpdate{}}}
	service := NewService(repository, shared.DefaultConfig())

	_, err := service.ListUpdates(context.Background(), &bootstrap.User{UserID: "subscriber-1"}, "", "")
	require.NoError(t, err)
	require.Equal(t, 50, repository.updatesLimit)

	cursor, err := shared.EncodeSubscriptionUpdateCursor(shared.SubscriptionUpdateCursor{Version: 1, Checklist: checklistID})
	require.NoError(t, err)
	_, err = service.ListUpdates(context.Background(), &bootstrap.User{UserID: "subscriber-1"}, cursor, "100")
	require.NoError(t, err)
	require.Equal(t, 100, repository.updatesLimit)
	require.NotNil(t, repository.updatesAfter)
	require.Equal(t, checklistID, *repository.updatesAfter)

	_, err = service.ListUpdates(context.Background(), &bootstrap.User{UserID: "subscriber-1"}, "not-a-cursor", "")
	requireAPIError(t, err, 400, "invalid_request")
	_, err = service.ListUpdates(context.Background(), &bootstrap.User{UserID: "subscriber-1"}, "", "101")
	requireAPIError(t, err, 400, "invalid_request")
}

func requireAPIError(t *testing.T, err error, status int, code string) {
	t.Helper()
	var apiError *shared.APIError
	require.ErrorAs(t, err, &apiError)
	require.Equal(t, status, apiError.Status)
	require.Equal(t, code, apiError.Code)
}
