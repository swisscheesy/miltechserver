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

func requireAPIError(t *testing.T, err error, status int, code string) {
	t.Helper()
	var apiError *shared.APIError
	require.ErrorAs(t, err, &apiError)
	require.Equal(t, status, apiError.Status)
	require.Equal(t, code, apiError.Code)
}
