package community

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"miltechserver/api/user_pmcs/shared"
	"miltechserver/bootstrap"
)

type repositoryStub struct {
	releaseResult     *ReleaseMutationResult
	releaseError      error
	retireResult      *ReleaseMutationResult
	retireError       error
	releaseCalls      int
	retireCalls       int
	receivedOwnerUID  string
	receivedChecklist uuid.UUID
	receivedRevision  uuid.UUID
	receivedCondition shared.Precondition
}

func (stub *repositoryStub) Release(
	_ context.Context,
	ownerUID string,
	checklistID uuid.UUID,
	revisionID uuid.UUID,
	precondition shared.Precondition,
) (*ReleaseMutationResult, error) {
	stub.releaseCalls++
	stub.receivedOwnerUID = ownerUID
	stub.receivedChecklist = checklistID
	stub.receivedRevision = revisionID
	stub.receivedCondition = precondition
	return stub.releaseResult, stub.releaseError
}

func (stub *repositoryStub) Retire(
	_ context.Context,
	ownerUID string,
	checklistID uuid.UUID,
	precondition shared.Precondition,
) (*ReleaseMutationResult, error) {
	stub.retireCalls++
	stub.receivedOwnerUID = ownerUID
	stub.receivedChecklist = checklistID
	stub.receivedCondition = precondition
	return stub.retireResult, stub.retireError
}

func TestReleaseParsesAuthenticationIDsAndParentPrecondition(t *testing.T) {
	checklistID := uuid.New()
	revisionID := uuid.New()
	aggregate := shared.ChecklistAggregate{
		ID:          checklistID,
		SyncVersion: 9,
	}
	repository := &repositoryStub{
		releaseResult: &ReleaseMutationResult{Aggregate: aggregate},
	}
	service := NewService(repository)
	parentETag := shared.MakeChecklistETag(checklistID, 8)

	result, etag, err := service.Release(
		context.Background(),
		&bootstrap.User{UserID: "owner-1"},
		checklistID.String(),
		revisionID.String(),
		parentETag,
	)

	require.NoError(t, err)
	require.Same(t, repository.releaseResult, result)
	require.Equal(t, shared.MakeChecklistETag(checklistID, 9), etag)
	require.Equal(t, 1, repository.releaseCalls)
	require.Equal(t, "owner-1", repository.receivedOwnerUID)
	require.Equal(t, checklistID, repository.receivedChecklist)
	require.Equal(t, revisionID, repository.receivedRevision)
	require.Equal(t, shared.Precondition{
		Mode: shared.PreconditionMatch,
		ETag: parentETag,
	}, repository.receivedCondition)
}

func TestReleaseRejectsInvalidInputBeforeRepository(t *testing.T) {
	validID := uuid.NewString()
	tests := []struct {
		name       string
		user       *bootstrap.User
		checklist  string
		revision   string
		ifMatch    string
		wantStatus int
		wantCode   string
	}{
		{
			name:       "authentication",
			checklist:  validID,
			revision:   validID,
			ifMatch:    `"etag"`,
			wantStatus: 401,
			wantCode:   "authentication_required",
		},
		{
			name:       "checklist UUID",
			user:       &bootstrap.User{UserID: "owner-1"},
			checklist:  "invalid",
			revision:   validID,
			ifMatch:    `"etag"`,
			wantStatus: 400,
			wantCode:   "invalid_request",
		},
		{
			name:       "revision UUID",
			user:       &bootstrap.User{UserID: "owner-1"},
			checklist:  validID,
			revision:   uuid.Nil.String(),
			ifMatch:    `"etag"`,
			wantStatus: 400,
			wantCode:   "invalid_request",
		},
		{
			name:       "missing parent ETag",
			user:       &bootstrap.User{UserID: "owner-1"},
			checklist:  validID,
			revision:   validID,
			wantStatus: 428,
			wantCode:   "precondition_required",
		},
		{
			name:       "malformed parent ETag",
			user:       &bootstrap.User{UserID: "owner-1"},
			checklist:  validID,
			revision:   validID,
			ifMatch:    "*",
			wantStatus: 400,
			wantCode:   "invalid_precondition",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository := &repositoryStub{}
			service := NewService(repository)
			_, _, err := service.Release(
				context.Background(),
				test.user,
				test.checklist,
				test.revision,
				test.ifMatch,
			)
			requireCommunityAPIError(
				t,
				err,
				test.wantStatus,
				test.wantCode,
			)
			require.Zero(t, repository.releaseCalls)
		})
	}
}

func TestRetireParsesParentPreconditionAndReturnsCurrentETag(t *testing.T) {
	checklistID := uuid.New()
	aggregate := shared.ChecklistAggregate{
		ID:          checklistID,
		SyncVersion: 4,
	}
	repository := &repositoryStub{
		retireResult: &ReleaseMutationResult{Aggregate: aggregate},
	}
	service := NewService(repository)

	result, etag, err := service.Retire(
		context.Background(),
		&bootstrap.User{UserID: "owner-1"},
		checklistID.String(),
		shared.MakeChecklistETag(checklistID, 3),
	)

	require.NoError(t, err)
	require.Same(t, repository.retireResult, result)
	require.Equal(t, shared.MakeChecklistETag(checklistID, 4), etag)
	require.Equal(t, 1, repository.retireCalls)
	require.Equal(t, checklistID, repository.receivedChecklist)
}

func TestReleaseRejectsEmptyRepositoryResult(t *testing.T) {
	service := NewService(&repositoryStub{})

	_, _, err := service.Release(
		context.Background(),
		&bootstrap.User{UserID: "owner-1"},
		uuid.NewString(),
		uuid.NewString(),
		`"etag"`,
	)

	requireCommunityAPIError(t, err, 500, "internal_error")
}

func requireCommunityAPIError(
	t *testing.T,
	err error,
	status int,
	code string,
) {
	t.Helper()
	var apiError *shared.APIError
	require.ErrorAs(t, err, &apiError)
	require.Equal(t, status, apiError.Status)
	require.Equal(t, code, apiError.Code)
}
