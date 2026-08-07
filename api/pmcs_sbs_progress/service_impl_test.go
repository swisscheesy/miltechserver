package pmcs_sbs_progress

import (
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

func (repo *repoStub) GetComment(commentID uuid.UUID) (*CommentWithAuthor, error) {
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

	_, err := svc.EnsureInspection(nil, "vehicle-1", samplePmcsIDStr, InspectionRequest{GuideManual: "pmcs_sbs/hmmwv/file.json", PerformedDate: time.Now()})

	requireServiceError(t, err, ErrUnauthorized)
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
		{name: "blank equipment", equipmentID: " ", pmcsID: samplePmcsIDStr, req: InspectionRequest{GuideManual: "pmcs_sbs/hmmwv/file.json", PerformedDate: now}, want: ErrInvalidID},
		{name: "malformed pmcs id", equipmentID: "vehicle-1", pmcsID: "not-a-uuid", req: InspectionRequest{GuideManual: "pmcs_sbs/hmmwv/file.json", PerformedDate: now}, want: ErrInvalidPmcsID},
		{name: "blank pmcs id", equipmentID: "vehicle-1", pmcsID: " ", req: InspectionRequest{GuideManual: "pmcs_sbs/hmmwv/file.json", PerformedDate: now}, want: ErrInvalidPmcsID},
		{name: "invalid guide manual", equipmentID: "vehicle-1", pmcsID: samplePmcsIDStr, req: InspectionRequest{GuideManual: "pmcs/hmmwv/file.json", PerformedDate: now}, want: ErrInvalidGuideManual},
		{name: "zero performed date", equipmentID: "vehicle-1", pmcsID: samplePmcsIDStr, req: InspectionRequest{GuideManual: "pmcs_sbs/hmmwv/file.json"}, want: ErrInvalidRequest},
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
		GuideManual:   stringPointer("pmcs_sbs/hmmwv/file.json"),
		PerformedDate: time.Now().UTC(),
		PerformedBy:   &performedBy,
	}}
	svc := NewService(stub)

	resp, err := svc.EnsureInspection(requireUser(), "vehicle-1", samplePmcsIDStr, InspectionRequest{
		GuideManual:   "pmcs_sbs/hmmwv/file.json",
		PerformedDate: time.Now(),
	})

	require.NoError(t, err)
	require.Equal(t, samplePmcsID(), resp.ID)
	require.Equal(t, "guide", resp.SourceType)
	require.Equal(t, "vehicle-1", stub.capturedInspection.EquipmentID)
	require.NotNil(t, stub.capturedInspection.PerformedBy)
	require.Equal(t, "user-1", *stub.capturedInspection.PerformedBy)
	require.Empty(t, resp.Faults)
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
		GuideManual:   "pmcs_sbs/hmmwv/file.json",
		PerformedDate: time.Now(),
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
		GuideManual:   "pmcs_sbs/hmmwv/file.json",
		PerformedDate: time.Now(),
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

func TestListInspectionsMapsSummaries(t *testing.T) {
	now := time.Now().UTC()
	performedBy := "user-1"
	performedByUsername := "jsmith"
	stub := &repoStub{summaries: []InspectionSummary{
		{ID: samplePmcsID(), GuideManual: "pmcs_sbs/hmmwv/file.json", PerformedDate: now, FaultCount: 2, CreatedAt: now, PerformedBy: &performedBy, PerformedByUsername: &performedByUsername},
	}}
	svc := NewService(stub)

	resp, err := svc.ListInspections(requireUser(), "vehicle-1", ListInspectionsRequest{Limit: 10, Offset: 0})

	require.NoError(t, err)
	require.Equal(t, 1, resp.Count)
	require.Equal(t, "guide", resp.Inspections[0].SourceType)
	require.Equal(t, 2, resp.Inspections[0].FaultCount)
	require.NotNil(t, resp.Inspections[0].PerformedBy)
	require.Equal(t, "user-1", *resp.Inspections[0].PerformedBy)
	require.NotNil(t, resp.Inspections[0].PerformedByUsername)
	require.Equal(t, "jsmith", *resp.Inspections[0].PerformedByUsername)
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
		return FaultRequest{GuideManual: "pmcs_sbs/hmmwv/file.json", PerformedDate: time.Now(), SectionID: "before", ItemIndex: 0, ItemNo: "1", Status: "X", FaultText: "leak"}
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
		{name: "invalid guide manual", equipmentID: "vehicle-1", pmcsID: samplePmcsIDStr, mutate: func(r FaultRequest) FaultRequest { r.GuideManual = "pmcs_sbs/../file.json"; return r }, want: ErrInvalidGuideManual},
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
				GuideManual: "pmcs_sbs/hmmwv/file.json", PerformedDate: time.Now(), SectionID: "before", ItemIndex: 0, ItemNo: "1", Status: tc.input, FaultText: "leak",
			})
			require.NoError(t, err)
			require.Equal(t, tc.want, stub.capturedFault.Status)
		})
	}
}

func TestUpsertFaultReturnsMappedResponse(t *testing.T) {
	now := time.Now().UTC()
	stub := &repoStub{savedFault: &model.PmcsSbsFaults{
		PmcsID: samplePmcsID(), SectionID: "before", ItemIndex: 0, ItemNo: "1", Status: "x", FaultText: "leak", CreatedAt: now, UpdatedAt: now,
	}}
	svc := NewService(stub)

	resp, err := svc.UpsertFault(requireUser(), "vehicle-1", samplePmcsIDStr, FaultRequest{
		GuideManual: "pmcs_sbs/hmmwv/file.json", PerformedDate: now, SectionID: "before", ItemIndex: 0, ItemNo: "1", Status: "X", FaultText: "leak",
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
		GuideManual: "pmcs_sbs/hmmwv/file.json", PerformedDate: time.Now(), Notes: &blank,
	})

	require.NoError(t, err)
	require.Nil(t, stub.capturedInspection.Notes)

	padded := "  looks fine  "
	_, err = svc.EnsureInspection(requireUser(), "vehicle-1", samplePmcsIDStr, InspectionRequest{
		GuideManual: "pmcs_sbs/hmmwv/file.json", PerformedDate: time.Now(), Notes: &padded,
	})

	require.NoError(t, err)
	require.NotNil(t, stub.capturedInspection.Notes)
	require.Equal(t, "looks fine", *stub.capturedInspection.Notes)
}

func TestEnsureInspectionRejectsOverlongNotes(t *testing.T) {
	svc := NewService(&repoStub{})
	tooLong := strings.Repeat("a", maxNotesLength+1)

	_, err := svc.EnsureInspection(requireUser(), "vehicle-1", samplePmcsIDStr, InspectionRequest{
		GuideManual: "pmcs_sbs/hmmwv/file.json", PerformedDate: time.Now(), Notes: &tooLong,
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

	_, err := svc.UpdateComment(requireUser(), samplePmcsIDStr, UpdateCommentRequest{Text: "edited"})

	requireServiceError(t, err, ErrForbidden)
}

func TestUpdateCommentSucceedsForAuthor(t *testing.T) {
	stub := &repoStub{existingComment: &CommentWithAuthor{
		PmcsSbsInspectionComments: model.PmcsSbsInspectionComments{ID: samplePmcsID(), AuthorID: "user-1", Text: "original"},
	}}
	svc := NewService(stub)

	_, err := svc.UpdateComment(requireUser(), samplePmcsIDStr, UpdateCommentRequest{Text: "edited"})

	require.NoError(t, err)
	require.Equal(t, "edited", stub.capturedCommentText)
}

func TestDeleteCommentRequiresAuthorshipAndUsesSentinelText(t *testing.T) {
	stub := &repoStub{existingComment: &CommentWithAuthor{
		PmcsSbsInspectionComments: model.PmcsSbsInspectionComments{ID: samplePmcsID(), AuthorID: "user-1", Text: "original"},
	}}
	svc := NewService(stub)

	_, err := svc.DeleteComment(requireUser(), samplePmcsIDStr)

	require.NoError(t, err)
	require.Equal(t, deletedCommentText, stub.capturedCommentText)
}

func TestDeleteCommentRejectsNonAuthor(t *testing.T) {
	stub := &repoStub{existingComment: &CommentWithAuthor{
		PmcsSbsInspectionComments: model.PmcsSbsInspectionComments{ID: samplePmcsID(), AuthorID: "someone-else", Text: "original"},
	}}
	svc := NewService(stub)

	_, err := svc.DeleteComment(requireUser(), samplePmcsIDStr)

	requireServiceError(t, err, ErrForbidden)
}
