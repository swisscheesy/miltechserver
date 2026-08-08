package pmcs_sbs_progress

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/bootstrap"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type repoStub struct {
	inspection     *model.PmcsSbsInspections
	detailUsername *string
	faults         []model.PmcsSbsFaults
	comments       []CommentWithAuthor
	summaries      []InspectionSummary
	savedFault     *model.PmcsSbsFaults
	deletedCount   int64
	err            error

	lookupUsernameResult *string
	lookupUsernameErr    error
	lookupUsernameCalls  int

	existingComment *CommentWithAuthor
	createdComment  *CommentWithAuthor
	updatedComment  *CommentWithAuthor

	capturedUser             *bootstrap.User
	capturedEquipmentID      string
	capturedPmcsID           uuid.UUID
	capturedGuideManual      string
	capturedLimit            int
	capturedOffset           int
	capturedInspection       model.PmcsSbsInspections
	capturedFault            model.PmcsSbsFaults
	capturedDelete           FaultKey
	capturedBulkKeys         []FaultKey
	capturedLookupUsernameID string
	capturedCommentID        uuid.UUID
	capturedCommentText      string
}

func stringPointer(value string) *string {
	return &value
}

func (repo *repoStub) EnsureInspection(user *bootstrap.User, inspection model.PmcsSbsInspections) (*model.PmcsSbsInspections, error) {
	repo.capturedUser = user
	repo.capturedInspection = inspection
	if repo.inspection != nil {
		return repo.inspection, repo.err
	}
	return &inspection, repo.err
}

func (repo *repoStub) GetInspection(user *bootstrap.User, equipmentID string, pmcsID uuid.UUID) (*InspectionDetail, []model.PmcsSbsFaults, []CommentWithAuthor, error) {
	repo.capturedUser = user
	repo.capturedEquipmentID = equipmentID
	repo.capturedPmcsID = pmcsID
	if repo.inspection == nil {
		return nil, repo.faults, repo.comments, repo.err
	}
	return &InspectionDetail{PmcsSbsInspections: *repo.inspection, PerformedByUsername: repo.detailUsername}, repo.faults, repo.comments, repo.err
}

func (repo *repoStub) LookupUsername(userID string) (*string, error) {
	repo.lookupUsernameCalls++
	repo.capturedLookupUsernameID = userID
	return repo.lookupUsernameResult, repo.lookupUsernameErr
}

func (repo *repoStub) ListInspections(user *bootstrap.User, equipmentID string, guideManual string, limit int, offset int) ([]InspectionSummary, error) {
	repo.capturedUser = user
	repo.capturedEquipmentID = equipmentID
	repo.capturedGuideManual = guideManual
	repo.capturedLimit = limit
	repo.capturedOffset = offset
	return repo.summaries, repo.err
}

func (repo *repoStub) DeleteInspection(user *bootstrap.User, equipmentID string, pmcsID uuid.UUID) error {
	repo.capturedUser = user
	repo.capturedEquipmentID = equipmentID
	repo.capturedPmcsID = pmcsID
	return repo.err
}

func (repo *repoStub) UpsertFault(user *bootstrap.User, inspection model.PmcsSbsInspections, fault model.PmcsSbsFaults) (*model.PmcsSbsFaults, error) {
	repo.capturedUser = user
	repo.capturedInspection = inspection
	repo.capturedFault = fault
	if repo.savedFault != nil {
		return repo.savedFault, repo.err
	}
	return &fault, repo.err
}

func (repo *repoStub) DeleteFault(user *bootstrap.User, equipmentID string, key FaultKey) error {
	repo.capturedUser = user
	repo.capturedEquipmentID = equipmentID
	repo.capturedDelete = key
	return repo.err
}

func (repo *repoStub) DeleteFaults(user *bootstrap.User, equipmentID string, pmcsID uuid.UUID, keys []FaultKey) (int64, error) {
	repo.capturedUser = user
	repo.capturedEquipmentID = equipmentID
	repo.capturedPmcsID = pmcsID
	repo.capturedBulkKeys = keys
	return repo.deletedCount, repo.err
}

func (repo *repoStub) CreateComment(user *bootstrap.User, equipmentID string, pmcsID uuid.UUID, text string) (*CommentWithAuthor, error) {
	repo.capturedUser = user
	repo.capturedEquipmentID = equipmentID
	repo.capturedPmcsID = pmcsID
	repo.capturedCommentText = text
	if repo.createdComment != nil {
		return repo.createdComment, repo.err
	}
	return &CommentWithAuthor{PmcsSbsInspectionComments: model.PmcsSbsInspectionComments{
		ID: uuid.New(), PmcsID: pmcsID, AuthorID: user.UserID, Text: text,
	}}, repo.err
}

func (repo *repoStub) GetComment(user *bootstrap.User, equipmentID string, pmcsID uuid.UUID, commentID uuid.UUID) (*CommentWithAuthor, error) {
	repo.capturedUser = user
	repo.capturedEquipmentID = equipmentID
	repo.capturedPmcsID = pmcsID
	repo.capturedCommentID = commentID
	return repo.existingComment, repo.err
}

func (repo *repoStub) UpdateComment(commentID uuid.UUID, text string) (*CommentWithAuthor, error) {
	repo.capturedCommentID = commentID
	repo.capturedCommentText = text
	if repo.updatedComment != nil {
		return repo.updatedComment, repo.err
	}
	return &CommentWithAuthor{PmcsSbsInspectionComments: model.PmcsSbsInspectionComments{ID: commentID, Text: text}}, repo.err
}

func requireUser() *bootstrap.User {
	return &bootstrap.User{UserID: "user-1", Email: "user-1@example.com", Username: "user-1"}
}

func requireServiceError(t *testing.T, err error, target error) {
	t.Helper()
	require.Error(t, err)
	require.Truef(t, errors.Is(err, target), "expected %v, got %v", target, err)
}

const samplePmcsIDStr = "11111111-1111-1111-1111-111111111111"

func samplePmcsID() uuid.UUID {
	return uuid.MustParse(samplePmcsIDStr)
}

func TestEnsureInspectionRequiresAuth(t *testing.T) {
	svc := NewService(&repoStub{})

	_, err := svc.EnsureInspection(nil, "vehicle-1", samplePmcsIDStr, InspectionRequest{InspectionSourceRequest: InspectionSourceRequest{GuideManual: "pmcs_sbs/hmmwv/file.json"}, PerformedDate: time.Now()})

	requireServiceError(t, err, ErrUnauthorized)
}

func TestValidateInspectionSource(t *testing.T) {
	zero := int32(0)
	published := int32(3)
	validCustom := func() InspectionSourceRequest {
		return InspectionSourceRequest{
			SourceType:           "custom",
			CustomChecklistID:    "22222222-2222-2222-2222-222222222222",
			CustomRevisionID:     "33333333-3333-3333-3333-333333333333",
			CustomRevisionNumber: &zero,
			CustomChecklistName:  "Weekly Generator PMCS",
		}
	}

	cases := []struct {
		name       string
		request    InspectionSourceRequest
		wantErr    error
		wantSource ValidatedInspectionSource
	}{
		{
			name:    "legacy guide omission",
			request: InspectionSourceRequest{GuideManual: "pmcs_sbs/hmmwv/file.json"},
			wantSource: ValidatedInspectionSource{
				SourceType:  "guide",
				GuideManual: stringPointer("pmcs_sbs/hmmwv/file.json"),
			},
		},
		{
			name:    "explicit guide",
			request: InspectionSourceRequest{SourceType: "guide", GuideManual: "pmcs_sbs/hmmwv/file.json"},
			wantSource: ValidatedInspectionSource{
				SourceType:  "guide",
				GuideManual: stringPointer("pmcs_sbs/hmmwv/file.json"),
			},
		},
		{
			name:       "custom revision zero",
			request:    validCustom(),
			wantSource: validatedCustomSource(zero),
		},
		{
			name: "custom published revision",
			request: func() InspectionSourceRequest {
				request := validCustom()
				request.CustomRevisionNumber = &published
				return request
			}(),
			wantSource: validatedCustomSource(published),
		},
		{
			name: "mixed guide and custom fields",
			request: func() InspectionSourceRequest {
				request := validCustom()
				request.GuideManual = "pmcs_sbs/hmmwv/file.json"
				return request
			}(),
			wantErr: ErrInvalidRequest,
		},
		{
			name: "missing custom checklist id",
			request: func() InspectionSourceRequest {
				request := validCustom()
				request.CustomChecklistID = ""
				return request
			}(),
			wantErr: ErrInvalidRequest,
		},
		{
			name: "missing custom revision id",
			request: func() InspectionSourceRequest {
				request := validCustom()
				request.CustomRevisionID = ""
				return request
			}(),
			wantErr: ErrInvalidRequest,
		},
		{
			name: "missing custom revision number",
			request: func() InspectionSourceRequest {
				request := validCustom()
				request.CustomRevisionNumber = nil
				return request
			}(),
			wantErr: ErrInvalidRequest,
		},
		{
			name: "missing custom checklist name",
			request: func() InspectionSourceRequest {
				request := validCustom()
				request.CustomChecklistName = " "
				return request
			}(),
			wantErr: ErrInvalidRequest,
		},
		{
			name: "zero custom checklist id",
			request: func() InspectionSourceRequest {
				request := validCustom()
				request.CustomChecklistID = uuid.Nil.String()
				return request
			}(),
			wantErr: ErrInvalidRequest,
		},
		{
			name: "negative custom revision number",
			request: func() InspectionSourceRequest {
				request := validCustom()
				negative := int32(-1)
				request.CustomRevisionNumber = &negative
				return request
			}(),
			wantErr: ErrInvalidRequest,
		},
		{
			name:    "invalid source type",
			request: InspectionSourceRequest{SourceType: "legacy", GuideManual: "pmcs_sbs/hmmwv/file.json"},
			wantErr: ErrInvalidRequest,
		},
		{
			name:    "source type with guide whitespace",
			request: InspectionSourceRequest{SourceType: " guide ", GuideManual: "pmcs_sbs/hmmwv/file.json"},
			wantErr: ErrInvalidRequest,
		},
		{
			name: "source type with custom whitespace",
			request: func() InspectionSourceRequest {
				request := validCustom()
				request.SourceType = " custom "
				return request
			}(),
			wantErr: ErrInvalidRequest,
		},
		{
			name:    "whitespace source type is not omitted",
			request: InspectionSourceRequest{SourceType: " ", GuideManual: "pmcs_sbs/hmmwv/file.json"},
			wantErr: ErrInvalidRequest,
		},
		{
			name: "custom source with whitespace guide manual",
			request: func() InspectionSourceRequest {
				request := validCustom()
				request.GuideManual = " "
				return request
			}(),
			wantErr: ErrInvalidRequest,
		},
		{
			name:    "legacy guide with whitespace custom field",
			request: InspectionSourceRequest{GuideManual: "pmcs_sbs/hmmwv/file.json", CustomChecklistID: " "},
			wantErr: ErrInvalidRequest,
		},
		{
			name:    "explicit guide with whitespace custom field",
			request: InspectionSourceRequest{SourceType: "guide", GuideManual: "pmcs_sbs/hmmwv/file.json", CustomChecklistName: " "},
			wantErr: ErrInvalidRequest,
		},
		{
			name: "custom checklist name exceeds grapheme limit",
			request: func() InspectionSourceRequest {
				request := validCustom()
				request.CustomChecklistName = strings.Repeat("👍", maxShortFieldGraphemes+1)
				return request
			}(),
			wantErr: ErrInvalidRequest,
		},
		{
			name: "custom checklist name exceeds byte limit",
			request: func() InspectionSourceRequest {
				request := validCustom()
				request.CustomChecklistName = strings.Repeat("a", maxShortFieldBytes+1)
				return request
			}(),
			wantErr: ErrInvalidRequest,
		},
		{
			name: "invalid utf8 custom checklist name",
			request: func() InspectionSourceRequest {
				request := validCustom()
				request.CustomChecklistName = string([]byte{0xff})
				return request
			}(),
			wantErr: ErrInvalidRequest,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := normalizeInspectionSource(tc.request)
			if tc.wantErr != nil {
				requireServiceError(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.wantSource, got)
		})
	}
}

func validatedCustomSource(revisionNumber int32) ValidatedInspectionSource {
	checklistID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	revisionID := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	checklistName := "Weekly Generator PMCS"
	return ValidatedInspectionSource{
		SourceType:           "custom",
		CustomChecklistID:    &checklistID,
		CustomRevisionID:     &revisionID,
		CustomRevisionNumber: &revisionNumber,
		CustomChecklistName:  &checklistName,
	}
}

func TestEnsureInspectionRejectsInvalidValues(t *testing.T) {
	svc := NewService(&repoStub{})
	now := time.Now()

	cases := []struct {
		name        string
		equipmentID string
		pmcsID      string
		req         InspectionRequest
		want        error
	}{
		{name: "blank equipment", equipmentID: " ", pmcsID: samplePmcsIDStr, req: InspectionRequest{InspectionSourceRequest: InspectionSourceRequest{GuideManual: "pmcs_sbs/hmmwv/file.json"}, PerformedDate: now}, want: ErrInvalidID},
		{name: "malformed pmcs id", equipmentID: "vehicle-1", pmcsID: "not-a-uuid", req: InspectionRequest{InspectionSourceRequest: InspectionSourceRequest{GuideManual: "pmcs_sbs/hmmwv/file.json"}, PerformedDate: now}, want: ErrInvalidPmcsID},
		{name: "blank pmcs id", equipmentID: "vehicle-1", pmcsID: " ", req: InspectionRequest{InspectionSourceRequest: InspectionSourceRequest{GuideManual: "pmcs_sbs/hmmwv/file.json"}, PerformedDate: now}, want: ErrInvalidPmcsID},
		{name: "invalid guide manual", equipmentID: "vehicle-1", pmcsID: samplePmcsIDStr, req: InspectionRequest{InspectionSourceRequest: InspectionSourceRequest{GuideManual: "pmcs/hmmwv/file.json"}, PerformedDate: now}, want: ErrInvalidGuideManual},
		{name: "zero performed date", equipmentID: "vehicle-1", pmcsID: samplePmcsIDStr, req: InspectionRequest{InspectionSourceRequest: InspectionSourceRequest{GuideManual: "pmcs_sbs/hmmwv/file.json"}}, want: ErrInvalidRequest},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.EnsureInspection(requireUser(), tc.equipmentID, tc.pmcsID, tc.req)
			requireServiceError(t, err, tc.want)
		})
	}
}

func TestEnsureInspectionMapsResponse(t *testing.T) {
	performedBy := "user-1"
	stub := &repoStub{inspection: &model.PmcsSbsInspections{
		ID:            samplePmcsID(),
		EquipmentID:   "vehicle-1",
		SourceType:    "guide",
		GuideManual:   stringPointer("pmcs_sbs/hmmwv/file.json"),
		PerformedDate: time.Now().UTC(),
		PerformedBy:   &performedBy,
	}}
	svc := NewService(stub)

	resp, err := svc.EnsureInspection(requireUser(), "vehicle-1", samplePmcsIDStr, InspectionRequest{
		InspectionSourceRequest: InspectionSourceRequest{GuideManual: "pmcs_sbs/hmmwv/file.json"},
		PerformedDate:           time.Now(),
	})

	require.NoError(t, err)
	require.Equal(t, samplePmcsID(), resp.ID)
	require.Equal(t, "guide", resp.SourceType)
	require.Equal(t, "vehicle-1", stub.capturedInspection.EquipmentID)
	require.NotNil(t, stub.capturedInspection.PerformedBy)
	require.Equal(t, "user-1", *stub.capturedInspection.PerformedBy)
	require.Empty(t, resp.Faults)
}

func TestMapInspectionCustomSourceOmitsGuideManual(t *testing.T) {
	checklistID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	revisionID := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	revisionNumber := int32(3)
	checklistName := "Weekly Generator PMCS"

	response := mapInspection(model.PmcsSbsInspections{
		ID:                   samplePmcsID(),
		EquipmentID:          "vehicle-1",
		SourceType:           "custom",
		CustomChecklistID:    &checklistID,
		CustomRevisionID:     &revisionID,
		CustomRevisionNumber: &revisionNumber,
		CustomChecklistName:  &checklistName,
	}, nil, nil, nil)

	require.Equal(t, "custom", response.SourceType)
	require.Nil(t, response.GuideManual)
	require.Equal(t, &checklistID, response.CustomChecklistID)
	require.Equal(t, &revisionID, response.CustomRevisionID)
	require.Equal(t, &revisionNumber, response.CustomRevisionNumber)
	require.Equal(t, &checklistName, response.CustomChecklistName)

	body, err := json.Marshal(response)
	require.NoError(t, err)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(body, &payload))
	require.Equal(t, "custom", payload["source_type"])
	require.NotContains(t, payload, "guide_manual")
	require.Equal(t, checklistID.String(), payload["custom_checklist_id"])
}

func TestEnsureInspectionResolvesPerformedByUsernameFromCallerWithoutLookup(t *testing.T) {
	performedBy := "user-1"
	stub := &repoStub{inspection: &model.PmcsSbsInspections{
		ID:            samplePmcsID(),
		EquipmentID:   "vehicle-1",
		GuideManual:   stringPointer("pmcs_sbs/hmmwv/file.json"),
		PerformedDate: time.Now().UTC(),
		PerformedBy:   &performedBy,
	}}
	svc := NewService(stub)
	user := &bootstrap.User{UserID: "user-1", Username: "jsmith"}

	resp, err := svc.EnsureInspection(user, "vehicle-1", samplePmcsIDStr, InspectionRequest{
		InspectionSourceRequest: InspectionSourceRequest{GuideManual: "pmcs_sbs/hmmwv/file.json"},
		PerformedDate:           time.Now(),
	})

	require.NoError(t, err)
	require.NotNil(t, resp.PerformedByUsername)
	require.Equal(t, "jsmith", *resp.PerformedByUsername)
	require.Equal(t, 0, stub.lookupUsernameCalls)
}

func TestEnsureInspectionResolvesPerformedByUsernameViaLookupWhenStickyOwnerDiffers(t *testing.T) {
	performedBy := "original-user"
	lookupResult := "original-username"
	stub := &repoStub{
		inspection: &model.PmcsSbsInspections{
			ID:            samplePmcsID(),
			EquipmentID:   "vehicle-1",
			GuideManual:   stringPointer("pmcs_sbs/hmmwv/file.json"),
			PerformedDate: time.Now().UTC(),
			PerformedBy:   &performedBy,
		},
		lookupUsernameResult: &lookupResult,
	}
	svc := NewService(stub)
	user := &bootstrap.User{UserID: "editor-user", Username: "editor"}

	resp, err := svc.EnsureInspection(user, "vehicle-1", samplePmcsIDStr, InspectionRequest{
		InspectionSourceRequest: InspectionSourceRequest{GuideManual: "pmcs_sbs/hmmwv/file.json"},
		PerformedDate:           time.Now(),
	})

	require.NoError(t, err)
	require.NotNil(t, resp.PerformedByUsername)
	require.Equal(t, "original-username", *resp.PerformedByUsername)
	require.Equal(t, 1, stub.lookupUsernameCalls)
	require.Equal(t, "original-user", stub.capturedLookupUsernameID)
}

func TestGetInspectionRejectsInvalidPmcsID(t *testing.T) {
	svc := NewService(&repoStub{})

	_, err := svc.GetInspection(requireUser(), "vehicle-1", "not-a-uuid")

	requireServiceError(t, err, ErrInvalidPmcsID)
}

func TestGetInspectionMapsFaults(t *testing.T) {
	now := time.Now().UTC()
	stub := &repoStub{
		inspection: &model.PmcsSbsInspections{ID: samplePmcsID(), EquipmentID: "vehicle-1", GuideManual: stringPointer("pmcs_sbs/hmmwv/file.json"), PerformedDate: now},
		faults: []model.PmcsSbsFaults{{
			PmcsID: samplePmcsID(), SectionID: "before", ItemIndex: 0, ItemNo: "1", Status: "x", FaultText: "leak", CreatedAt: now, UpdatedAt: now,
		}},
	}
	svc := NewService(stub)

	resp, err := svc.GetInspection(requireUser(), "vehicle-1", samplePmcsIDStr)

	require.NoError(t, err)
	require.Equal(t, samplePmcsID(), stub.capturedPmcsID)
	require.Len(t, resp.Faults, 1)
	require.Equal(t, "leak", resp.Faults[0].FaultText)
}

func TestListInspectionsAppliesDefaultLimitAndOffset(t *testing.T) {
	stub := &repoStub{summaries: []InspectionSummary{}}
	svc := NewService(stub)

	_, err := svc.ListInspections(requireUser(), "vehicle-1", ListInspectionsRequest{})

	require.NoError(t, err)
	require.Equal(t, 1000, stub.capturedLimit)
	require.Equal(t, 0, stub.capturedOffset)
}

func TestListInspectionsValidatesGuideManualFilterWhenProvided(t *testing.T) {
	svc := NewService(&repoStub{})

	_, err := svc.ListInspections(requireUser(), "vehicle-1", ListInspectionsRequest{GuideManual: "pmcs/hmmwv/file.json"})

	requireServiceError(t, err, ErrInvalidGuideManual)
}

func TestListInspectionsMapsGuideAndCustomSummaries(t *testing.T) {
	now := time.Now().UTC()
	performedBy := "user-1"
	performedByUsername := "jsmith"
	guideManual := "pmcs_sbs/hmmwv/file.json"
	customID := uuid.New()
	customChecklistID := uuid.New()
	customRevisionID := uuid.New()
	customRevisionNumber := int32(3)
	customChecklistName := "Weekly Generator PMCS"
	stub := &repoStub{summaries: []InspectionSummary{
		{ID: samplePmcsID(), SourceType: "guide", GuideManual: &guideManual, PerformedDate: now, FaultCount: 2, CommentCount: 1, CreatedAt: now, PerformedBy: &performedBy, PerformedByUsername: &performedByUsername},
		{ID: customID, SourceType: "custom", CustomChecklistID: &customChecklistID, CustomRevisionID: &customRevisionID, CustomRevisionNumber: &customRevisionNumber, CustomChecklistName: &customChecklistName, PerformedDate: now.Add(-time.Hour), FaultCount: 1, CommentCount: 2, CreatedAt: now},
	}}
	svc := NewService(stub)

	resp, err := svc.ListInspections(requireUser(), "vehicle-1", ListInspectionsRequest{Limit: 10, Offset: 0})

	require.NoError(t, err)
	require.Equal(t, 2, resp.Count)
	require.Equal(t, "guide", resp.Inspections[0].SourceType)
	require.Equal(t, &guideManual, resp.Inspections[0].GuideManual)
	require.Equal(t, 2, resp.Inspections[0].FaultCount)
	require.Equal(t, 1, resp.Inspections[0].CommentCount)
	require.NotNil(t, resp.Inspections[0].PerformedBy)
	require.Equal(t, "user-1", *resp.Inspections[0].PerformedBy)
	require.NotNil(t, resp.Inspections[0].PerformedByUsername)
	require.Equal(t, "jsmith", *resp.Inspections[0].PerformedByUsername)
	require.Equal(t, customID, resp.Inspections[1].ID)
	require.Equal(t, "custom", resp.Inspections[1].SourceType)
	require.Nil(t, resp.Inspections[1].GuideManual)
	require.Equal(t, &customChecklistID, resp.Inspections[1].CustomChecklistID)
	require.Equal(t, &customRevisionID, resp.Inspections[1].CustomRevisionID)
	require.Equal(t, &customRevisionNumber, resp.Inspections[1].CustomRevisionNumber)
	require.Equal(t, &customChecklistName, resp.Inspections[1].CustomChecklistName)
	require.Equal(t, 1, resp.Inspections[1].FaultCount)
	require.Equal(t, 2, resp.Inspections[1].CommentCount)
}

func TestDeleteInspectionValidatesPmcsID(t *testing.T) {
	svc := NewService(&repoStub{})

	err := svc.DeleteInspection(requireUser(), "vehicle-1", "not-a-uuid")

	requireServiceError(t, err, ErrInvalidPmcsID)
}

func TestDeleteInspectionPassesParsedID(t *testing.T) {
	stub := &repoStub{}
	svc := NewService(stub)

	err := svc.DeleteInspection(requireUser(), " vehicle-1 ", samplePmcsIDStr)

	require.NoError(t, err)
	require.Equal(t, "vehicle-1", stub.capturedEquipmentID)
	require.Equal(t, samplePmcsID(), stub.capturedPmcsID)
}

func TestUpsertFaultRejectsInvalidValues(t *testing.T) {
	svc := NewService(&repoStub{})
	baseReq := func() FaultRequest {
		return FaultRequest{InspectionSourceRequest: InspectionSourceRequest{GuideManual: "pmcs_sbs/hmmwv/file.json"}, PerformedDate: time.Now(), SectionID: "before", ItemIndex: 0, ItemNo: "1", Status: "X", FaultText: "leak"}
	}

	cases := []struct {
		name        string
		equipmentID string
		pmcsID      string
		mutate      func(FaultRequest) FaultRequest
		want        error
	}{
		{name: "blank equipment", equipmentID: " ", pmcsID: samplePmcsIDStr, mutate: func(r FaultRequest) FaultRequest { return r }, want: ErrInvalidID},
		{name: "malformed pmcs id", equipmentID: "vehicle-1", pmcsID: "bad", mutate: func(r FaultRequest) FaultRequest { return r }, want: ErrInvalidPmcsID},
		{name: "invalid guide manual", equipmentID: "vehicle-1", pmcsID: samplePmcsIDStr, mutate: func(r FaultRequest) FaultRequest {
			r.InspectionSourceRequest.GuideManual = "pmcs_sbs/../file.json"
			return r
		}, want: ErrInvalidGuideManual},
		{name: "blank section", equipmentID: "vehicle-1", pmcsID: samplePmcsIDStr, mutate: func(r FaultRequest) FaultRequest { r.SectionID = " "; return r }, want: ErrInvalidRequest},
		{name: "negative item index", equipmentID: "vehicle-1", pmcsID: samplePmcsIDStr, mutate: func(r FaultRequest) FaultRequest { r.ItemIndex = -1; return r }, want: ErrInvalidRequest},
		{name: "blank item no", equipmentID: "vehicle-1", pmcsID: samplePmcsIDStr, mutate: func(r FaultRequest) FaultRequest { r.ItemNo = " "; return r }, want: ErrInvalidRequest},
		{name: "invalid status", equipmentID: "vehicle-1", pmcsID: samplePmcsIDStr, mutate: func(r FaultRequest) FaultRequest { r.Status = "BAD"; return r }, want: ErrInvalidStatus},
		{name: "blank fault text", equipmentID: "vehicle-1", pmcsID: samplePmcsIDStr, mutate: func(r FaultRequest) FaultRequest { r.FaultText = " "; return r }, want: ErrInvalidRequest},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.UpsertFault(requireUser(), tc.equipmentID, tc.pmcsID, tc.mutate(baseReq()))
			requireServiceError(t, err, tc.want)
		})
	}
}

func TestUpsertFaultAcceptsAllowedStatuses(t *testing.T) {
	stub := &repoStub{}
	svc := NewService(stub)
	cases := []struct {
		input string
		want  string
	}{
		{input: "X", want: "x"},
		{input: "x", want: "x"},
		{input: "/", want: "slash"},
		{input: "slash", want: "slash"},
		{input: "-", want: "dash"},
		{input: "dash", want: "dash"},
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			_, err := svc.UpsertFault(requireUser(), "vehicle-1", samplePmcsIDStr, FaultRequest{
				InspectionSourceRequest: InspectionSourceRequest{GuideManual: "pmcs_sbs/hmmwv/file.json"}, PerformedDate: time.Now(), SectionID: "before", ItemIndex: 0, ItemNo: "1", Status: tc.input, FaultText: "leak",
			})
			require.NoError(t, err)
			require.Equal(t, tc.want, stub.capturedFault.Status)
		})
	}
}

func TestUpsertFaultAcceptsCustomSource(t *testing.T) {
	stub := &repoStub{}
	svc := NewService(stub)
	revisionNumber := int32(0)

	resp, err := svc.UpsertFault(requireUser(), "vehicle-1", samplePmcsIDStr, FaultRequest{
		InspectionSourceRequest: InspectionSourceRequest{
			SourceType:           "custom",
			CustomChecklistID:    "22222222-2222-2222-2222-222222222222",
			CustomRevisionID:     "33333333-3333-3333-3333-333333333333",
			CustomRevisionNumber: &revisionNumber,
			CustomChecklistName:  "Weekly Generator PMCS",
		},
		PerformedDate: time.Now(),
		SectionID:     "44444444-4444-4444-4444-444444444444",
		SectionTitle:  "Before Operation",
		ItemIndex:     0,
		ItemNo:        "1",
		Status:        "X",
		FaultText:     "leak",
	})

	require.NoError(t, err)
	require.Equal(t, "custom", stub.capturedInspection.SourceType)
	require.Nil(t, stub.capturedInspection.GuideManual)
	require.Equal(t, uuid.MustParse("22222222-2222-2222-2222-222222222222"), *stub.capturedInspection.CustomChecklistID)
	require.Equal(t, uuid.MustParse("33333333-3333-3333-3333-333333333333"), *stub.capturedInspection.CustomRevisionID)
	require.Equal(t, int32(0), *stub.capturedInspection.CustomRevisionNumber)
	require.Equal(t, "Weekly Generator PMCS", *stub.capturedInspection.CustomChecklistName)
	require.NotNil(t, stub.capturedFault.SectionTitle)
	require.Equal(t, "Before Operation", *stub.capturedFault.SectionTitle)
	require.NotNil(t, resp.SectionTitle)
	require.Equal(t, "Before Operation", *resp.SectionTitle)
}

func TestUpsertFaultValidatesSectionTitle(t *testing.T) {
	baseRequest := func() FaultRequest {
		return FaultRequest{
			InspectionSourceRequest: InspectionSourceRequest{GuideManual: "pmcs_sbs/hmmwv/file.json"},
			PerformedDate:           time.Now(),
			SectionID:               "before",
			ItemNo:                  "1",
			Status:                  "x",
			FaultText:               "leak",
		}
	}

	cases := []struct {
		name  string
		title string
		want  error
	}{
		{name: "omitted", title: ""},
		{name: "at grapheme limit", title: strings.Repeat("👍", maxShortFieldGraphemes)},
		{name: "at byte limit", title: shortFieldAtByteLimit()},
		{name: "over grapheme limit", title: strings.Repeat("👍", maxShortFieldGraphemes+1), want: ErrInvalidRequest},
		{name: "over byte limit", title: shortFieldAtByteLimit() + "a", want: ErrInvalidRequest},
		{name: "invalid utf8", title: string([]byte{0xff}), want: ErrInvalidRequest},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stub := &repoStub{}
			svc := NewService(stub)
			request := baseRequest()
			request.SectionTitle = tc.title

			_, err := svc.UpsertFault(requireUser(), "vehicle-1", samplePmcsIDStr, request)
			if tc.want != nil {
				requireServiceError(t, err, tc.want)
				return
			}
			require.NoError(t, err)
			if tc.title == "" {
				require.Nil(t, stub.capturedFault.SectionTitle)
				return
			}
			require.NotNil(t, stub.capturedFault.SectionTitle)
			require.Equal(t, tc.title, *stub.capturedFault.SectionTitle)
		})
	}
}

func shortFieldAtByteLimit() string {
	return strings.Repeat("a", maxShortFieldGraphemes) + strings.Repeat("\u0301", (maxShortFieldBytes-maxShortFieldGraphemes)/2)
}

func TestUpsertFaultReturnsMappedResponse(t *testing.T) {
	now := time.Now().UTC()
	stub := &repoStub{savedFault: &model.PmcsSbsFaults{
		PmcsID: samplePmcsID(), SectionID: "before", ItemIndex: 0, ItemNo: "1", Status: "x", FaultText: "leak", CreatedAt: now, UpdatedAt: now,
	}}
	svc := NewService(stub)

	resp, err := svc.UpsertFault(requireUser(), "vehicle-1", samplePmcsIDStr, FaultRequest{
		InspectionSourceRequest: InspectionSourceRequest{GuideManual: "pmcs_sbs/hmmwv/file.json"}, PerformedDate: now, SectionID: "before", ItemIndex: 0, ItemNo: "1", Status: "X", FaultText: "leak",
	})

	require.NoError(t, err)
	require.Equal(t, "vehicle-1", stub.capturedInspection.EquipmentID)
	require.Equal(t, samplePmcsID(), stub.capturedInspection.ID)
	require.Equal(t, "x", stub.capturedFault.Status)
	require.Equal(t, samplePmcsID(), resp.PmcsID)
}

func TestDeleteFaultPassesValidatedKey(t *testing.T) {
	stub := &repoStub{}
	svc := NewService(stub)

	err := svc.DeleteFault(requireUser(), " vehicle-1 ", samplePmcsIDStr, DeleteFaultRequest{SectionID: " before ", ItemIndex: 0})

	require.NoError(t, err)
	require.Equal(t, "vehicle-1", stub.capturedEquipmentID)
	require.Equal(t, samplePmcsID(), stub.capturedDelete.PmcsID)
	require.Equal(t, "before", stub.capturedDelete.SectionID)
}

func TestDeleteFaultsPassesValidatedKeysAndCounts(t *testing.T) {
	stub := &repoStub{deletedCount: 1}
	svc := NewService(stub)

	resp, err := svc.DeleteFaults(requireUser(), " vehicle-1 ", samplePmcsIDStr, BulkDeleteFaultRequest{
		Faults: []BulkDeleteFaultItemRequest{
			{SectionID: " before ", ItemIndex: 0},
			{SectionID: " after ", ItemIndex: 2},
		},
	})

	require.NoError(t, err)
	require.Equal(t, "vehicle-1", stub.capturedEquipmentID)
	require.Equal(t, []FaultKey{
		{PmcsID: samplePmcsID(), SectionID: "before", ItemIndex: 0},
		{PmcsID: samplePmcsID(), SectionID: "after", ItemIndex: 2},
	}, stub.capturedBulkKeys)
	require.Equal(t, 2, resp.RequestedCount)
	require.Equal(t, 1, resp.DeletedCount)
}

func TestDeleteFaultsRequiresAuth(t *testing.T) {
	svc := NewService(&repoStub{})

	_, err := svc.DeleteFaults(nil, "vehicle-1", samplePmcsIDStr, BulkDeleteFaultRequest{
		Faults: []BulkDeleteFaultItemRequest{{SectionID: "before", ItemIndex: 0}},
	})

	requireServiceError(t, err, ErrUnauthorized)
}

func TestValidateBulkDeleteFaultRequestRejectsInvalidValues(t *testing.T) {
	svc := NewService(&repoStub{})
	validFaults := []BulkDeleteFaultItemRequest{{SectionID: "before", ItemIndex: 0}}
	tooManyFaults := make([]BulkDeleteFaultItemRequest, maxBulkDeleteFaults+1)
	for i := range tooManyFaults {
		tooManyFaults[i] = BulkDeleteFaultItemRequest{SectionID: "before", ItemIndex: int32(i)}
	}

	cases := []struct {
		name        string
		equipmentID string
		pmcsID      string
		req         BulkDeleteFaultRequest
		want        error
	}{
		{name: "blank equipment", equipmentID: " ", pmcsID: samplePmcsIDStr, req: BulkDeleteFaultRequest{Faults: validFaults}, want: ErrInvalidID},
		{name: "malformed pmcs id", equipmentID: "vehicle-1", pmcsID: "bad", req: BulkDeleteFaultRequest{Faults: validFaults}, want: ErrInvalidPmcsID},
		{name: "empty faults", equipmentID: "vehicle-1", pmcsID: samplePmcsIDStr, req: BulkDeleteFaultRequest{Faults: []BulkDeleteFaultItemRequest{}}, want: ErrInvalidRequest},
		{name: "too many faults", equipmentID: "vehicle-1", pmcsID: samplePmcsIDStr, req: BulkDeleteFaultRequest{Faults: tooManyFaults}, want: ErrInvalidRequest},
		{name: "blank section", equipmentID: "vehicle-1", pmcsID: samplePmcsIDStr, req: BulkDeleteFaultRequest{Faults: []BulkDeleteFaultItemRequest{{SectionID: " ", ItemIndex: 0}}}, want: ErrInvalidRequest},
		{name: "negative item index", equipmentID: "vehicle-1", pmcsID: samplePmcsIDStr, req: BulkDeleteFaultRequest{Faults: []BulkDeleteFaultItemRequest{{SectionID: "before", ItemIndex: -1}}}, want: ErrInvalidRequest},
		{name: "duplicate key", equipmentID: "vehicle-1", pmcsID: samplePmcsIDStr, req: BulkDeleteFaultRequest{Faults: []BulkDeleteFaultItemRequest{{SectionID: " before ", ItemIndex: 0}, {SectionID: "before", ItemIndex: 0}}}, want: ErrInvalidRequest},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, _, _, err := svc.validateBulkDeleteFaultRequest(tc.equipmentID, tc.pmcsID, tc.req)
			requireServiceError(t, err, tc.want)
		})
	}
}

func TestEnsureInspectionTrimsAndClearsNotes(t *testing.T) {
	stub := &repoStub{}
	svc := NewService(stub)

	blank := "   "
	_, err := svc.EnsureInspection(requireUser(), "vehicle-1", samplePmcsIDStr, InspectionRequest{
		InspectionSourceRequest: InspectionSourceRequest{GuideManual: "pmcs_sbs/hmmwv/file.json"}, PerformedDate: time.Now(), Notes: &blank,
	})

	require.NoError(t, err)
	require.Nil(t, stub.capturedInspection.Notes)

	padded := "  looks fine  "
	_, err = svc.EnsureInspection(requireUser(), "vehicle-1", samplePmcsIDStr, InspectionRequest{
		InspectionSourceRequest: InspectionSourceRequest{GuideManual: "pmcs_sbs/hmmwv/file.json"}, PerformedDate: time.Now(), Notes: &padded,
	})

	require.NoError(t, err)
	require.NotNil(t, stub.capturedInspection.Notes)
	require.Equal(t, "looks fine", *stub.capturedInspection.Notes)
}

func TestEnsureInspectionRejectsOverlongNotes(t *testing.T) {
	svc := NewService(&repoStub{})
	tooLong := strings.Repeat("a", maxNotesLength+1)

	_, err := svc.EnsureInspection(requireUser(), "vehicle-1", samplePmcsIDStr, InspectionRequest{
		InspectionSourceRequest: InspectionSourceRequest{GuideManual: "pmcs_sbs/hmmwv/file.json"}, PerformedDate: time.Now(), Notes: &tooLong,
	})

	requireServiceError(t, err, ErrInvalidRequest)
}

func TestGetInspectionMapsNotesAndComments(t *testing.T) {
	now := time.Now().UTC()
	notes := "clean inspection"
	authorUsername := "jsmith"
	stub := &repoStub{
		inspection: &model.PmcsSbsInspections{ID: samplePmcsID(), EquipmentID: "vehicle-1", GuideManual: stringPointer("pmcs_sbs/hmmwv/file.json"), PerformedDate: now, Notes: &notes},
		comments: []CommentWithAuthor{{
			PmcsSbsInspectionComments: model.PmcsSbsInspectionComments{ID: uuid.New(), PmcsID: samplePmcsID(), AuthorID: "user-1", Text: "looks good", CreatedAt: now},
			AuthorUsername:            &authorUsername,
		}},
	}
	svc := NewService(stub)

	resp, err := svc.GetInspection(requireUser(), "vehicle-1", samplePmcsIDStr)

	require.NoError(t, err)
	require.NotNil(t, resp.Notes)
	require.Equal(t, "clean inspection", *resp.Notes)
	require.Len(t, resp.Comments, 1)
	require.Equal(t, "looks good", resp.Comments[0].Text)
	require.NotNil(t, resp.Comments[0].AuthorUsername)
	require.Equal(t, "jsmith", *resp.Comments[0].AuthorUsername)
}

func TestCreateCommentRequiresAuth(t *testing.T) {
	svc := NewService(&repoStub{})

	_, err := svc.CreateComment(nil, "vehicle-1", samplePmcsIDStr, CreateCommentRequest{Text: "hello"})

	requireServiceError(t, err, ErrUnauthorized)
}

func TestCreateCommentRejectsInvalidText(t *testing.T) {
	svc := NewService(&repoStub{})

	cases := []struct {
		name string
		text string
	}{
		{name: "blank", text: "   "},
		{name: "too long", text: strings.Repeat("a", maxCommentTextLength+1)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.CreateComment(requireUser(), "vehicle-1", samplePmcsIDStr, CreateCommentRequest{Text: tc.text})
			requireServiceError(t, err, ErrInvalidCommentText)
		})
	}
}

func TestCreateCommentTrimsTextAndPassesThrough(t *testing.T) {
	stub := &repoStub{}
	svc := NewService(stub)

	resp, err := svc.CreateComment(requireUser(), "vehicle-1", samplePmcsIDStr, CreateCommentRequest{Text: "  looks good  "})

	require.NoError(t, err)
	require.Equal(t, "looks good", stub.capturedCommentText)
	require.Equal(t, "looks good", resp.Text)
	require.Equal(t, samplePmcsID(), stub.capturedPmcsID)
}

func TestUpdateCommentRequiresAuthorship(t *testing.T) {
	stub := &repoStub{existingComment: &CommentWithAuthor{
		PmcsSbsInspectionComments: model.PmcsSbsInspectionComments{ID: samplePmcsID(), AuthorID: "someone-else", Text: "original"},
	}}
	svc := NewService(stub)

	_, err := svc.UpdateComment(requireUser(), "vehicle-1", samplePmcsIDStr, samplePmcsIDStr, UpdateCommentRequest{Text: "edited"})

	requireServiceError(t, err, ErrForbidden)
}

func TestUpdateCommentSucceedsForAuthor(t *testing.T) {
	stub := &repoStub{existingComment: &CommentWithAuthor{
		PmcsSbsInspectionComments: model.PmcsSbsInspectionComments{ID: samplePmcsID(), AuthorID: "user-1", Text: "original"},
	}}
	svc := NewService(stub)

	_, err := svc.UpdateComment(requireUser(), "vehicle-1", samplePmcsIDStr, samplePmcsIDStr, UpdateCommentRequest{Text: "edited"})

	require.NoError(t, err)
	require.Equal(t, "vehicle-1", stub.capturedEquipmentID)
	require.Equal(t, samplePmcsID(), stub.capturedPmcsID)
	require.Equal(t, "edited", stub.capturedCommentText)
}

func TestDeleteCommentRequiresAuthorshipAndUsesSentinelText(t *testing.T) {
	stub := &repoStub{existingComment: &CommentWithAuthor{
		PmcsSbsInspectionComments: model.PmcsSbsInspectionComments{ID: samplePmcsID(), AuthorID: "user-1", Text: "original"},
	}}
	svc := NewService(stub)

	_, err := svc.DeleteComment(requireUser(), "vehicle-1", samplePmcsIDStr, samplePmcsIDStr)

	require.NoError(t, err)
	require.Equal(t, "vehicle-1", stub.capturedEquipmentID)
	require.Equal(t, samplePmcsID(), stub.capturedPmcsID)
	require.Equal(t, deletedCommentText, stub.capturedCommentText)
}

func TestDeleteCommentRejectsNonAuthor(t *testing.T) {
	stub := &repoStub{existingComment: &CommentWithAuthor{
		PmcsSbsInspectionComments: model.PmcsSbsInspectionComments{ID: samplePmcsID(), AuthorID: "someone-else", Text: "original"},
	}}
	svc := NewService(stub)

	_, err := svc.DeleteComment(requireUser(), "vehicle-1", samplePmcsIDStr, samplePmcsIDStr)

	requireServiceError(t, err, ErrForbidden)
}
