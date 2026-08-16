package community

import (
	"context"
	"encoding/json"
	"testing"
	"time"

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
	putVoteResult     *shared.CommunityVoteMutation
	putVoteError      error
	deleteVoteResult  *shared.CommunityVoteMutation
	deleteVoteError   error
	putVoteCalls      int
	deleteVoteCalls   int
	receivedVoterUID  string
	receivedDirection int16
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

func (stub *repositoryStub) PutVote(
	_ context.Context,
	voterUID string,
	checklistID uuid.UUID,
	direction int16,
) (*shared.CommunityVoteMutation, error) {
	stub.putVoteCalls++
	stub.receivedVoterUID = voterUID
	stub.receivedChecklist = checklistID
	stub.receivedDirection = direction
	return stub.putVoteResult, stub.putVoteError
}

func (stub *repositoryStub) DeleteVote(
	_ context.Context,
	voterUID string,
	checklistID uuid.UUID,
) (*shared.CommunityVoteMutation, error) {
	stub.deleteVoteCalls++
	stub.receivedVoterUID = voterUID
	stub.receivedChecklist = checklistID
	return stub.deleteVoteResult, stub.deleteVoteError
}

type browseRepositoryStub struct {
	repositoryStub
	browseResult *shared.CommunityPage
	browseError  error
	browseCalls  int
	browseFilter shared.CommunityBrowseFilter
}

func (stub *browseRepositoryStub) Browse(
	_ context.Context,
	filter shared.CommunityBrowseFilter,
) (*shared.CommunityPage, error) {
	stub.browseCalls++
	stub.browseFilter = filter
	return stub.browseResult, stub.browseError
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

func TestBrowsePublicMapsOnlyPublicFieldsAndDefaultsToTop(t *testing.T) {
	checklistID := uuid.New()
	repository := &browseRepositoryStub{
		browseResult: &shared.CommunityPage{
			HasMore: true,
			Items: []shared.CommunitySummary{{
				ChecklistID: checklistID,
				Name:        "Public checklist",
				Score:       -2,
				MyVote:      pointerToVote(1),
				CanVote:     true,
			}},
		},
	}
	service := NewService(repository)

	page, err := service.BrowsePublic(
		context.Background(),
		"",
		"",
		" M1165 ",
		"",
	)

	require.NoError(t, err)
	require.Equal(t, shared.CommunitySortTop, repository.browseFilter.Sort)
	require.Empty(t, repository.browseFilter.ViewerUID)
	require.Equal(t, "m1165", repository.browseFilter.NormalizedModel)
	require.Equal(t, checklistID, page.Items[0].ChecklistID)
	require.Equal(t, int64(-2), page.Items[0].Score)
	encoded, err := json.Marshal(page)
	require.NoError(t, err)
	require.NotContains(t, string(encoded), "my_vote")
	require.NotContains(t, string(encoded), "can_vote")
}

func TestBrowseAuthenticatedProjectsViewerVoteAndEligibility(t *testing.T) {
	updatedAt := time.Date(2026, time.August, 16, 12, 0, 0, 0, time.UTC)
	checklistID := uuid.New()
	repository := &browseRepositoryStub{
		browseResult: &shared.CommunityPage{Items: []shared.CommunitySummary{
			{
				ChecklistID: checklistID,
				UpdatedAt:   updatedAt,
				Score:       3,
				MyVote:      pointerToVote(-1),
				CanVote:     false,
			},
			{ChecklistID: uuid.New(), CanVote: true},
		}},
	}
	service := NewService(repository)

	page, err := service.BrowseAuthenticated(
		context.Background(),
		&bootstrap.User{UserID: "viewer-1"},
		"",
		"10",
		"",
		"recent",
	)

	require.NoError(t, err)
	require.Equal(t, shared.CommunitySortRecent, repository.browseFilter.Sort)
	require.Equal(t, "viewer-1", repository.browseFilter.ViewerUID)
	require.Equal(t, int16(-1), *page.Items[0].MyVote)
	require.False(t, page.Items[0].CanVote)
	require.Nil(t, page.Items[1].MyVote)
	require.True(t, page.Items[1].CanVote)
}

func TestBrowseRejectsCrossSortCursorBeforeRepository(t *testing.T) {
	score := int64(1)
	cursor, err := shared.EncodeCommunityCursor(shared.CommunityCursor{
		Version:   2,
		Sort:      shared.CommunitySortTop,
		Score:     &score,
		UpdatedAt: time.Date(2026, time.August, 16, 12, 0, 0, 0, time.UTC),
		Checklist: uuid.New(),
	})
	require.NoError(t, err)
	repository := &browseRepositoryStub{}
	service := NewService(repository)

	_, err = service.BrowsePublic(
		context.Background(),
		cursor,
		"",
		"",
		"recent",
	)

	requireCommunityAPIError(t, err, 400, "invalid_request")
	require.Zero(t, repository.browseCalls)
}

func TestPutVoteValidatesAndPreservesRepositoryErrors(t *testing.T) {
	checklistID := uuid.New()
	repository := &repositoryStub{
		putVoteResult: &shared.CommunityVoteMutation{
			ChecklistID: checklistID,
			Score:       4,
			MyVote:      pointerToVote(1),
		},
	}
	service := NewService(repository)

	result, err := service.PutVote(
		context.Background(),
		&bootstrap.User{UserID: "voter-1"},
		checklistID.String(),
		1,
	)

	require.NoError(t, err)
	require.Same(t, repository.putVoteResult, result)
	require.Equal(t, 1, repository.putVoteCalls)
	require.Equal(t, "voter-1", repository.receivedVoterUID)
	require.Equal(t, checklistID, repository.receivedChecklist)
	require.Equal(t, int16(1), repository.receivedDirection)

	for _, test := range []struct {
		name      string
		user      *bootstrap.User
		checklist string
		direction int16
		status    int
		code      string
	}{
		{"missing user", nil, checklistID.String(), 1, 401, "authentication_required"},
		{"blank uid", &bootstrap.User{UserID: " "}, checklistID.String(), 1, 401, "authentication_required"},
		{"invalid UUID", &bootstrap.User{UserID: "voter-1"}, "invalid", 1, 400, "invalid_request"},
		{"zero direction", &bootstrap.User{UserID: "voter-1"}, checklistID.String(), 0, 400, "invalid_request"},
		{"positive out of range", &bootstrap.User{UserID: "voter-1"}, checklistID.String(), 2, 400, "invalid_request"},
		{"negative out of range", &bootstrap.User{UserID: "voter-1"}, checklistID.String(), -2, 400, "invalid_request"},
	} {
		t.Run(test.name, func(t *testing.T) {
			invalidRepository := &repositoryStub{}
			invalidService := NewService(invalidRepository)
			_, voteErr := invalidService.PutVote(
				context.Background(),
				test.user,
				test.checklist,
				test.direction,
			)
			requireCommunityAPIError(t, voteErr, test.status, test.code)
			require.Zero(t, invalidRepository.putVoteCalls)
		})
	}

	repository.putVoteError = shared.NewResourceNotFound("source unavailable", nil)
	_, err = service.PutVote(
		context.Background(),
		&bootstrap.User{UserID: "voter-1"},
		checklistID.String(),
		-1,
	)
	require.Same(t, repository.putVoteError, err)
}

func TestDeleteVoteParsesAuthenticatedVoterAndPreservesRepositoryError(t *testing.T) {
	checklistID := uuid.New()
	repository := &repositoryStub{
		deleteVoteResult: &shared.CommunityVoteMutation{ChecklistID: checklistID},
	}
	service := NewService(repository)

	result, err := service.DeleteVote(
		context.Background(),
		&bootstrap.User{UserID: "voter-1"},
		checklistID.String(),
	)
	require.NoError(t, err)
	require.Same(t, repository.deleteVoteResult, result)
	require.Equal(t, 1, repository.deleteVoteCalls)
	require.Equal(t, "voter-1", repository.receivedVoterUID)
	require.Equal(t, checklistID, repository.receivedChecklist)

	for _, test := range []struct {
		name      string
		user      *bootstrap.User
		checklist string
		status    int
		code      string
	}{
		{"missing user", nil, checklistID.String(), 401, "authentication_required"},
		{"nil UUID", &bootstrap.User{UserID: "voter-1"}, uuid.Nil.String(), 400, "invalid_request"},
	} {
		t.Run(test.name, func(t *testing.T) {
			invalidRepository := &repositoryStub{}
			invalidService := NewService(invalidRepository)
			_, deleteErr := invalidService.DeleteVote(
				context.Background(), test.user, test.checklist,
			)
			requireCommunityAPIError(t, deleteErr, test.status, test.code)
			require.Zero(t, invalidRepository.deleteVoteCalls)
		})
	}

	repository.deleteVoteError = shared.NewAccountNotInitialized(
		"account is not initialized", nil,
	)
	_, err = service.DeleteVote(
		context.Background(),
		&bootstrap.User{UserID: "voter-1"},
		checklistID.String(),
	)
	require.Same(t, repository.deleteVoteError, err)
}

func pointerToVote(value int16) *int16 {
	return &value
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
