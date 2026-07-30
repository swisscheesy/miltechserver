package owned

import (
	"context"
	"slices"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"miltechserver/api/user_pmcs/shared"
	"miltechserver/bootstrap"
)

type repositoryStub struct {
	getResult         *shared.ChecklistAggregate
	getError          error
	getRevisionResult *HistoricalRevisionResult
	getRevisionError  error
	mutationResult    *MutationResult
	mutationError     error
	getCalls          int
	getRevisionCalls  int
	createCalls       int
	putDraftCalls     int
	deleteDraftCalls  int
	publishCalls      int
	receivedOwnerUID  string
	receivedChecklist uuid.UUID
	receivedRevision  uuid.UUID
	receivedDraft     shared.PreparedRevision
	receivedCondition shared.Precondition
}

func (stub *repositoryStub) GetRevision(
	_ context.Context,
	ownerUID string,
	checklistID uuid.UUID,
	revisionID uuid.UUID,
) (*HistoricalRevisionResult, error) {
	stub.getRevisionCalls++
	stub.receivedOwnerUID = ownerUID
	stub.receivedChecklist = checklistID
	stub.receivedRevision = revisionID
	return stub.getRevisionResult, stub.getRevisionError
}

func (stub *repositoryStub) Get(
	_ context.Context,
	ownerUID string,
	checklistID uuid.UUID,
) (*shared.ChecklistAggregate, error) {
	stub.getCalls++
	stub.receivedOwnerUID = ownerUID
	stub.receivedChecklist = checklistID
	return stub.getResult, stub.getError
}

func (stub *repositoryStub) Create(
	_ context.Context,
	ownerUID string,
	checklistID uuid.UUID,
	draft shared.PreparedRevision,
	precondition shared.Precondition,
) (*MutationResult, error) {
	stub.createCalls++
	stub.receivedOwnerUID = ownerUID
	stub.receivedChecklist = checklistID
	stub.receivedDraft = draft
	stub.receivedCondition = precondition
	return stub.mutationResult, stub.mutationError
}

func (stub *repositoryStub) PutDraft(
	_ context.Context,
	ownerUID string,
	checklistID uuid.UUID,
	draft shared.PreparedRevision,
	precondition shared.Precondition,
) (*MutationResult, error) {
	stub.putDraftCalls++
	stub.receivedOwnerUID = ownerUID
	stub.receivedChecklist = checklistID
	stub.receivedDraft = draft
	stub.receivedCondition = precondition
	return stub.mutationResult, stub.mutationError
}

func (stub *repositoryStub) DeleteDraft(
	_ context.Context,
	ownerUID string,
	checklistID uuid.UUID,
	revisionID uuid.UUID,
	precondition shared.Precondition,
) (*MutationResult, error) {
	stub.deleteDraftCalls++
	stub.receivedOwnerUID = ownerUID
	stub.receivedChecklist = checklistID
	stub.receivedRevision = revisionID
	stub.receivedCondition = precondition
	return stub.mutationResult, stub.mutationError
}

func (stub *repositoryStub) DeleteChecklist(
	_ context.Context,
	ownerUID string,
	checklistID uuid.UUID,
	precondition shared.Precondition,
) (*MutationResult, error) {
	stub.receivedOwnerUID = ownerUID
	stub.receivedChecklist = checklistID
	stub.receivedCondition = precondition
	return stub.mutationResult, stub.mutationError
}

func (stub *repositoryStub) Publish(
	_ context.Context,
	ownerUID string,
	checklistID uuid.UUID,
	revision shared.PreparedRevision,
	precondition shared.Precondition,
) (*MutationResult, error) {
	stub.publishCalls++
	stub.receivedOwnerUID = ownerUID
	stub.receivedChecklist = checklistID
	stub.receivedRevision = revision.Input.ID
	stub.receivedDraft = revision
	stub.receivedCondition = precondition
	return stub.mutationResult, stub.mutationError
}

func TestGetOwnedRejectsMissingAuthenticationBeforeRepository(t *testing.T) {
	repository := &repositoryStub{}
	service := NewService(repository, shared.DefaultConfig())

	_, _, err := service.Get(context.Background(), nil, uuid.NewString())

	requireAPIError(t, err, 401, "authentication_required")
	require.Zero(t, repository.getCalls)
}

func TestGetOwnedRejectsInvalidChecklistUUIDBeforeRepository(t *testing.T) {
	repository := &repositoryStub{}
	service := NewService(repository, shared.DefaultConfig())

	_, _, err := service.Get(
		context.Background(),
		&bootstrap.User{UserID: "owner-1"},
		"not-a-uuid",
	)

	requireAPIError(t, err, 400, "invalid_request")
	require.Zero(t, repository.getCalls)
}

func TestCreateRequiresIfNoneMatchStarBeforeRepository(t *testing.T) {
	repository := &repositoryStub{}
	service := NewService(repository, shared.DefaultConfig())
	user := &bootstrap.User{UserID: "owner-1"}

	_, _, err := service.Create(
		context.Background(),
		user,
		uuid.NewString(),
		validDraftInput(uuid.New()),
		"",
	)

	requireAPIError(t, err, 428, "precondition_required")
	require.Zero(t, repository.createCalls)
}

func TestCreateRejectsMalformedIfNoneMatchBeforeRepository(t *testing.T) {
	repository := &repositoryStub{}
	service := NewService(repository, shared.DefaultConfig())

	_, _, err := service.Create(
		context.Background(),
		&bootstrap.User{UserID: "owner-1"},
		uuid.NewString(),
		validDraftInput(uuid.New()),
		`"anything"`,
	)

	requireAPIError(t, err, 400, "invalid_precondition")
	require.Zero(t, repository.createCalls)
}

func TestCreateRejectsDraftRevisionNumberBeforeRepository(t *testing.T) {
	repository := &repositoryStub{}
	service := NewService(repository, shared.DefaultConfig())
	input := validDraftInput(uuid.New())
	number := int32(1)
	input.RevisionNumber = &number

	_, _, err := service.Create(
		context.Background(),
		&bootstrap.User{UserID: "owner-1"},
		uuid.NewString(),
		input,
		"*",
	)

	requireAPIError(t, err, 422, "validation_failed")
	require.Zero(t, repository.createCalls)
}

func TestCreatePreparesDraftBeforeRepository(t *testing.T) {
	checklistID := uuid.New()
	revisionID := uuid.New()
	aggregate := shared.ChecklistAggregate{ID: checklistID, SyncVersion: 1}
	repository := &repositoryStub{
		mutationResult: &MutationResult{Aggregate: aggregate, Created: true},
	}
	service := NewService(repository, shared.DefaultConfig())

	result, etag, err := service.Create(
		context.Background(),
		&bootstrap.User{UserID: "owner-1"},
		checklistID.String(),
		validDraftInput(revisionID),
		"*",
	)

	require.NoError(t, err)
	require.Equal(t, aggregate, result.Aggregate)
	require.Equal(t, shared.MakeChecklistETag(checklistID, 1), etag)
	require.Equal(t, "owner-1", repository.receivedOwnerUID)
	require.Equal(t, checklistID, repository.receivedChecklist)
	require.Equal(t, revisionID, repository.receivedDraft.Input.ID)
	require.Equal(t, "m998 hmmwv", repository.receivedDraft.Input.Models[0].NormalizedText)
	require.Equal(t, shared.PreconditionCreate, repository.receivedCondition.Mode)
}

func TestPutDraftRequiresParentChecklistETagBeforeRepository(t *testing.T) {
	repository := &repositoryStub{}
	service := NewService(repository, shared.DefaultConfig())
	revisionID := uuid.New()

	_, _, err := service.PutDraft(
		context.Background(),
		&bootstrap.User{UserID: "owner-1"},
		uuid.NewString(),
		revisionID.String(),
		validDraftInput(revisionID),
		"",
	)

	requireAPIError(t, err, 428, "precondition_required")
	require.Zero(t, repository.putDraftCalls)
}

func TestPutDraftRejectsMalformedParentChecklistETag(t *testing.T) {
	repository := &repositoryStub{}
	service := NewService(repository, shared.DefaultConfig())
	revisionID := uuid.New()

	_, _, err := service.PutDraft(
		context.Background(),
		&bootstrap.User{UserID: "owner-1"},
		uuid.NewString(),
		revisionID.String(),
		validDraftInput(revisionID),
		"*",
	)

	requireAPIError(t, err, 400, "invalid_precondition")
	require.Zero(t, repository.putDraftCalls)
}

func TestPutDraftRejectsBodyAndRouteRevisionMismatch(t *testing.T) {
	repository := &repositoryStub{}
	service := NewService(repository, shared.DefaultConfig())

	_, _, err := service.PutDraft(
		context.Background(),
		&bootstrap.User{UserID: "owner-1"},
		uuid.NewString(),
		uuid.NewString(),
		validDraftInput(uuid.New()),
		`"current"`,
	)

	requireAPIError(t, err, 400, "invalid_request")
	require.Zero(t, repository.putDraftCalls)
}

func TestPutDraftRejectsInvalidRevisionUUIDBeforeRepository(t *testing.T) {
	repository := &repositoryStub{}
	service := NewService(repository, shared.DefaultConfig())

	_, _, err := service.PutDraft(
		context.Background(),
		&bootstrap.User{UserID: "owner-1"},
		uuid.NewString(),
		"not-a-uuid",
		validDraftInput(uuid.New()),
		`"current"`,
	)

	requireAPIError(t, err, 400, "invalid_request")
	require.Zero(t, repository.putDraftCalls)
}

func TestDeleteDraftRequiresParentChecklistETagBeforeRepository(t *testing.T) {
	repository := &repositoryStub{}
	service := NewService(repository, shared.DefaultConfig())

	_, _, err := service.DeleteDraft(
		context.Background(),
		&bootstrap.User{UserID: "owner-1"},
		uuid.NewString(),
		uuid.NewString(),
		"",
	)

	requireAPIError(t, err, 428, "precondition_required")
	require.Zero(t, repository.deleteDraftCalls)
}

func TestPublishRequiresPositiveRevisionNumberBeforeRepository(t *testing.T) {
	repository := &repositoryStub{}
	service := NewService(repository, shared.DefaultConfig())
	revisionID := uuid.New()
	input := completePublicationInput(revisionID, 0)

	_, _, err := service.Publish(
		context.Background(),
		&bootstrap.User{UserID: "owner-1"},
		uuid.NewString(),
		revisionID.String(),
		input,
		`"current"`,
	)

	requireAPIError(t, err, 422, "validation_failed")
	require.Zero(t, repository.publishCalls)
}

func TestPublishRejectsBodyAndRouteRevisionMismatch(t *testing.T) {
	repository := &repositoryStub{}
	service := NewService(repository, shared.DefaultConfig())
	input := completePublicationInput(uuid.New(), 1)

	_, _, err := service.Publish(
		context.Background(),
		&bootstrap.User{UserID: "owner-1"},
		uuid.NewString(),
		uuid.NewString(),
		input,
		`"current"`,
	)

	requireAPIError(t, err, 400, "invalid_request")
	require.Zero(t, repository.publishCalls)
}

func TestPublishPreparesCompleteRevisionBeforeRepository(t *testing.T) {
	checklistID := uuid.New()
	revisionID := uuid.New()
	aggregate := shared.ChecklistAggregate{ID: checklistID, SyncVersion: 2}
	repository := &repositoryStub{
		mutationResult: &MutationResult{Aggregate: aggregate},
	}
	service := NewService(repository, shared.DefaultConfig())
	input := completePublicationInput(revisionID, 1)

	result, etag, err := service.Publish(
		context.Background(),
		&bootstrap.User{UserID: "owner-1"},
		checklistID.String(),
		revisionID.String(),
		input,
		`"current"`,
	)

	require.NoError(t, err)
	require.Equal(t, aggregate, result.Aggregate)
	require.Equal(t, shared.MakeChecklistETag(checklistID, 2), etag)
	require.Equal(t, int32(1), *repository.receivedDraft.Input.RevisionNumber)
	require.Equal(t, "m998 hmmwv", repository.receivedDraft.Input.Models[0].NormalizedText)
	require.Equal(t, shared.PreconditionMatch, repository.receivedCondition.Mode)
}

func TestHistoricalGetReturnsImmutableETag(t *testing.T) {
	checklistID := uuid.New()
	revisionID := uuid.New()
	number := int32(1)
	var storedHash [32]byte
	storedHash[0] = 0x7f
	repository := &repositoryStub{
		getRevisionResult: &HistoricalRevisionResult{
			Revision: HistoricalRevision{
				ID:             revisionID,
				RevisionNumber: &number,
				Name:           "Vehicle PMCS",
				Description:    "Tree intentionally differs from stored hash",
				Models: []shared.ModelValue{
					{DisplayText: "M998", NormalizedText: "m998"},
				},
				Sections: []shared.Section{},
			},
			ContentHash: storedHash,
		},
	}
	service := NewService(repository, shared.DefaultConfig())

	revision, etag, err := service.GetRevision(
		context.Background(),
		&bootstrap.User{UserID: "owner-1"},
		checklistID.String(),
		revisionID.String(),
	)

	require.NoError(t, err)
	require.Equal(t, revisionID, revision.ID)
	require.Equal(
		t,
		immutableRevisionETag(checklistID, revisionID, storedHash),
		etag,
	)
	require.Equal(t, checklistID, repository.receivedChecklist)
	require.Equal(t, revisionID, repository.receivedRevision)
}

func TestPreparedTreeAdvisoryKeysAreStableSortedAndDeduplicated(t *testing.T) {
	revisionID := uuid.MustParse("00000000-0000-0000-0000-000000000003")
	firstNodeID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	secondNodeID := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	input := shared.RevisionInput{
		ID: revisionID,
		Sections: []shared.SectionInput{
			{
				ID: secondNodeID,
				Items: []shared.ItemInput{
					{ID: firstNodeID},
					{ID: secondNodeID},
				},
			},
		},
	}

	keys := preparedTreeAdvisoryKeys(input)
	reordered := input
	reordered.Sections[0].Items[0], reordered.Sections[0].Items[1] =
		reordered.Sections[0].Items[1], reordered.Sections[0].Items[0]

	require.Len(t, keys, 3)
	require.True(t, slices.IsSorted(keys))
	require.Equal(t, keys, preparedTreeAdvisoryKeys(reordered))
}

func TestPreparedTreeAdvisoryKeysAreBoundedAtMaximumTreeCeiling(t *testing.T) {
	input := shared.RevisionInput{ID: uuid.New()}
	for sectionIndex := 0; sectionIndex < 100; sectionIndex++ {
		section := shared.SectionInput{ID: uuid.New()}
		for itemIndex := 0; itemIndex < 20; itemIndex++ {
			item := shared.ItemInput{ID: uuid.New()}
			for noticeIndex := 0; noticeIndex < 2; noticeIndex++ {
				item.Notices = append(
					item.Notices,
					shared.NoticeInput{ID: uuid.New()},
				)
			}
			for stepIndex := 0; stepIndex < 5; stepIndex++ {
				item.ProcedureSteps = append(
					item.ProcedureSteps,
					shared.ProcedureStepInput{ID: uuid.New()},
				)
			}
			section.Items = append(section.Items, item)
		}
		input.Sections = append(input.Sections, section)
	}

	keys := preparedTreeAdvisoryKeys(input)

	require.LessOrEqual(t, len(keys), 32)
}

func validDraftInput(revisionID uuid.UUID) shared.RevisionInput {
	return shared.RevisionInput{
		ID:          revisionID,
		Name:        "Vehicle PMCS",
		Description: "Draft",
		Models: []shared.ModelInput{
			{DisplayText: " M998  HMMWV "},
		},
		Sections: []shared.SectionInput{},
	}
}

func completePublicationInput(
	revisionID uuid.UUID,
	revisionNumber int32,
) shared.RevisionInput {
	input := validDraftInput(revisionID)
	input.RevisionNumber = &revisionNumber
	input.Sections = []shared.SectionInput{
		{
			ID:       uuid.New(),
			Position: 1,
			Title:    "Before operation",
			Items: []shared.ItemInput{
				{
					ID:                        uuid.New(),
					Position:                  1,
					Interval:                  "Before",
					ItemToBeCheckedOrServiced: "Engine compartment",
					ProcedureSteps: []shared.ProcedureStepInput{
						{
							ID:       uuid.New(),
							Position: 1,
							StepText: "Inspect",
						},
					},
				},
			},
		},
	}
	return input
}

func requireAPIError(
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
