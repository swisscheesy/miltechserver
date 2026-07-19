package pmcs_sbs_progress

import (
	"errors"
	"testing"
	"time"

	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/bootstrap"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type repoStub struct {
	inspection    *model.PmcsSbsInspections
	faults        []model.PmcsSbsFaults
	summaries     []InspectionSummary
	savedFault    *model.PmcsSbsFaults
	deletedCount  int64
	err           error

	capturedUser        *bootstrap.User
	capturedEquipmentID string
	capturedPmcsID      uuid.UUID
	capturedGuideManual string
	capturedLimit       int
	capturedOffset      int
	capturedInspection  model.PmcsSbsInspections
	capturedFault       model.PmcsSbsFaults
	capturedDelete      FaultKey
	capturedBulkKeys    []FaultKey
}

func (repo *repoStub) EnsureInspection(user *bootstrap.User, inspection model.PmcsSbsInspections) (*model.PmcsSbsInspections, error) {
	repo.capturedUser = user
	repo.capturedInspection = inspection
	if repo.inspection != nil {
		return repo.inspection, repo.err
	}
	return &inspection, repo.err
}

func (repo *repoStub) GetInspection(user *bootstrap.User, equipmentID string, pmcsID uuid.UUID) (*model.PmcsSbsInspections, []model.PmcsSbsFaults, error) {
	repo.capturedUser = user
	repo.capturedEquipmentID = equipmentID
	repo.capturedPmcsID = pmcsID
	return repo.inspection, repo.faults, repo.err
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
		GuideManual:   "pmcs_sbs/hmmwv/file.json",
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
	require.Equal(t, "vehicle-1", stub.capturedInspection.EquipmentID)
	require.NotNil(t, stub.capturedInspection.PerformedBy)
	require.Equal(t, "user-1", *stub.capturedInspection.PerformedBy)
	require.Empty(t, resp.Faults)
}

func TestGetInspectionRejectsInvalidPmcsID(t *testing.T) {
	svc := NewService(&repoStub{})

	_, err := svc.GetInspection(requireUser(), "vehicle-1", "not-a-uuid")

	requireServiceError(t, err, ErrInvalidPmcsID)
}

func TestGetInspectionMapsFaults(t *testing.T) {
	now := time.Now().UTC()
	stub := &repoStub{
		inspection: &model.PmcsSbsInspections{ID: samplePmcsID(), EquipmentID: "vehicle-1", GuideManual: "pmcs_sbs/hmmwv/file.json", PerformedDate: now},
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
	stub := &repoStub{summaries: []InspectionSummary{
		{ID: samplePmcsID(), GuideManual: "pmcs_sbs/hmmwv/file.json", PerformedDate: now, FaultCount: 2, CreatedAt: now},
	}}
	svc := NewService(stub)

	resp, err := svc.ListInspections(requireUser(), "vehicle-1", ListInspectionsRequest{Limit: 10, Offset: 0})

	require.NoError(t, err)
	require.Equal(t, 1, resp.Count)
	require.Equal(t, 2, resp.Inspections[0].FaultCount)
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
