package pmcs_sbs_progress

import (
	"errors"
	"testing"
	"time"

	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/bootstrap"

	"github.com/stretchr/testify/require"
)

type repoStub struct {
	listFaults []model.PmcsSbsFaults
	savedFault *model.PmcsSbsFaults
	err        error

	capturedUser        *bootstrap.User
	capturedEquipmentID string
	capturedGuideManual string
	capturedFault       model.PmcsSbsFaults
	capturedDelete      FaultKey
}

func (repo *repoStub) ListFaults(user *bootstrap.User, equipmentID string, guideManual string) ([]model.PmcsSbsFaults, error) {
	repo.capturedUser = user
	repo.capturedEquipmentID = equipmentID
	repo.capturedGuideManual = guideManual
	return repo.listFaults, repo.err
}

func (repo *repoStub) UpsertFault(user *bootstrap.User, fault model.PmcsSbsFaults) (*model.PmcsSbsFaults, error) {
	repo.capturedUser = user
	repo.capturedFault = fault
	if repo.savedFault != nil {
		return repo.savedFault, repo.err
	}
	return &fault, repo.err
}

func (repo *repoStub) DeleteFault(user *bootstrap.User, key FaultKey) error {
	repo.capturedUser = user
	repo.capturedDelete = key
	return repo.err
}

func requireUser() *bootstrap.User {
	return &bootstrap.User{UserID: "user-1", Email: "user-1@example.com", Username: "user-1"}
}

func requireServiceError(t *testing.T, err error, target error) {
	t.Helper()
	require.Error(t, err)
	require.Truef(t, errors.Is(err, target), "expected %v, got %v", target, err)
}

func TestListFaultsRequiresAuth(t *testing.T) {
	svc := NewService(&repoStub{})

	_, err := svc.ListFaults(nil, "vehicle-1", "pmcs_sbs/hmmwv/file.json")

	requireServiceError(t, err, ErrUnauthorized)
}

func TestListFaultsRejectsBlankEquipmentID(t *testing.T) {
	svc := NewService(&repoStub{})

	_, err := svc.ListFaults(requireUser(), " ", "pmcs_sbs/hmmwv/file.json")

	requireServiceError(t, err, ErrInvalidID)
}

func TestListFaultsRejectsInvalidGuideManual(t *testing.T) {
	svc := NewService(&repoStub{})

	_, err := svc.ListFaults(requireUser(), "vehicle-1", "pmcs_sbs/../file.json")

	requireServiceError(t, err, ErrInvalidGuideManual)
}

func TestListFaultsMapsRows(t *testing.T) {
	now := time.Now().UTC()
	stub := &repoStub{listFaults: []model.PmcsSbsFaults{{
		EquipmentID:      "vehicle-1",
		GuideManual:      "pmcs_sbs/hmmwv/file.json",
		SectionID:        "before",
		ItemIndex:        0,
		ItemNo:           "1",
		Status:           "x",
		FaultText:        "leak",
		CorrectiveAction: "tightened",
		CreatedAt:        now,
		UpdatedAt:        now,
	}}}
	svc := NewService(stub)

	resp, err := svc.ListFaults(requireUser(), " vehicle-1 ", " pmcs_sbs/hmmwv/file.json ")

	require.NoError(t, err)
	require.Equal(t, "vehicle-1", stub.capturedEquipmentID)
	require.Equal(t, "pmcs_sbs/hmmwv/file.json", stub.capturedGuideManual)
	require.Len(t, resp.Faults, 1)
	require.Equal(t, "vehicle-1", resp.Faults[0].EquipmentID)
	require.Equal(t, "pmcs_sbs/hmmwv/file.json", resp.Faults[0].GuideManual)
	require.Equal(t, "before", resp.Faults[0].SectionID)
	require.Equal(t, "x", resp.Faults[0].Status)
	require.Equal(t, 1, resp.Count)
}

func TestValidateFaultRequest(t *testing.T) {
	svc := NewService(&repoStub{})

	fault, err := svc.validateFaultRequest(" vehicle-1 ", FaultRequest{
		GuideManual:      " pmcs_sbs/hmmwv/file.json ",
		SectionID:        " before ",
		ItemIndex:        0,
		ItemNo:           " 1 ",
		Status:           " X ",
		FaultText:        " leak ",
		CorrectiveAction: " tightened ",
	})

	require.NoError(t, err)
	require.Equal(t, "vehicle-1", fault.EquipmentID)
	require.Equal(t, "pmcs_sbs/hmmwv/file.json", fault.GuideManual)
	require.Equal(t, "before", fault.SectionID)
	require.Equal(t, int32(0), fault.ItemIndex)
	require.Equal(t, "1", fault.ItemNo)
	require.Equal(t, "x", fault.Status)
	require.Equal(t, "leak", fault.FaultText)
	require.Equal(t, "tightened", fault.CorrectiveAction)
	require.False(t, fault.CreatedAt.IsZero())
	require.False(t, fault.UpdatedAt.IsZero())
}

func TestValidateFaultRequestAcceptsAllowedStatuses(t *testing.T) {
	svc := NewService(&repoStub{})
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
			fault, err := svc.validateFaultRequest("vehicle-1", FaultRequest{
				GuideManual: "pmcs_sbs/hmmwv/file.json",
				SectionID:   "before",
				ItemIndex:   0,
				ItemNo:      "1",
				Status:      tc.input,
				FaultText:   "leak",
			})
			require.NoError(t, err)
			require.Equal(t, tc.want, fault.Status)
		})
	}
}

func TestValidateFaultRequestRejectsInvalidValues(t *testing.T) {
	svc := NewService(&repoStub{})

	cases := []struct {
		name        string
		equipmentID string
		req         FaultRequest
		want        error
	}{
		{
			name:        "blank equipment",
			equipmentID: " ",
			req:         FaultRequest{GuideManual: "pmcs_sbs/hmmwv/file.json", SectionID: "before", ItemIndex: 0, ItemNo: "1", Status: "X", FaultText: "leak"},
			want:        ErrInvalidID,
		},
		{
			name:        "blank guide manual",
			equipmentID: "vehicle-1",
			req:         FaultRequest{GuideManual: " ", SectionID: "before", ItemIndex: 0, ItemNo: "1", Status: "X", FaultText: "leak"},
			want:        ErrInvalidGuideManual,
		},
		{
			name:        "guide manual traversal",
			equipmentID: "vehicle-1",
			req:         FaultRequest{GuideManual: "pmcs_sbs/hmmwv/../file.json", SectionID: "before", ItemIndex: 0, ItemNo: "1", Status: "X", FaultText: "leak"},
			want:        ErrInvalidGuideManual,
		},
		{
			name:        "guide manual outside pmcs sbs",
			equipmentID: "vehicle-1",
			req:         FaultRequest{GuideManual: "pmcs/hmmwv/file.json", SectionID: "before", ItemIndex: 0, ItemNo: "1", Status: "X", FaultText: "leak"},
			want:        ErrInvalidGuideManual,
		},
		{
			name:        "blank section",
			equipmentID: "vehicle-1",
			req:         FaultRequest{GuideManual: "pmcs_sbs/hmmwv/file.json", SectionID: " ", ItemIndex: 0, ItemNo: "1", Status: "X", FaultText: "leak"},
			want:        ErrInvalidRequest,
		},
		{
			name:        "negative item index",
			equipmentID: "vehicle-1",
			req:         FaultRequest{GuideManual: "pmcs_sbs/hmmwv/file.json", SectionID: "before", ItemIndex: -1, ItemNo: "1", Status: "X", FaultText: "leak"},
			want:        ErrInvalidRequest,
		},
		{
			name:        "blank item no",
			equipmentID: "vehicle-1",
			req:         FaultRequest{GuideManual: "pmcs_sbs/hmmwv/file.json", SectionID: "before", ItemIndex: 0, ItemNo: " ", Status: "X", FaultText: "leak"},
			want:        ErrInvalidRequest,
		},
		{
			name:        "invalid status",
			equipmentID: "vehicle-1",
			req:         FaultRequest{GuideManual: "pmcs_sbs/hmmwv/file.json", SectionID: "before", ItemIndex: 0, ItemNo: "1", Status: "BAD", FaultText: "leak"},
			want:        ErrInvalidStatus,
		},
		{
			name:        "blank fault text",
			equipmentID: "vehicle-1",
			req:         FaultRequest{GuideManual: "pmcs_sbs/hmmwv/file.json", SectionID: "before", ItemIndex: 0, ItemNo: "1", Status: "X", FaultText: " "},
			want:        ErrInvalidRequest,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.validateFaultRequest(tc.equipmentID, tc.req)
			requireServiceError(t, err, tc.want)
		})
	}
}

func TestValidateDeleteFaultRequest(t *testing.T) {
	svc := NewService(&repoStub{})

	key, err := svc.validateDeleteFaultRequest(" vehicle-1 ", DeleteFaultRequest{
		GuideManual: " pmcs_sbs/hmmwv/file.json ",
		SectionID:   " before ",
		ItemIndex:   0,
	})

	require.NoError(t, err)
	require.Equal(t, "vehicle-1", key.EquipmentID)
	require.Equal(t, "pmcs_sbs/hmmwv/file.json", key.GuideManual)
	require.Equal(t, "before", key.SectionID)
	require.Equal(t, int32(0), key.ItemIndex)
}

func TestValidateDeleteFaultRequestRejectsInvalidValues(t *testing.T) {
	svc := NewService(&repoStub{})

	_, err := svc.validateDeleteFaultRequest(" ", DeleteFaultRequest{GuideManual: "pmcs_sbs/hmmwv/file.json", SectionID: "before", ItemIndex: 0})
	requireServiceError(t, err, ErrInvalidID)

	_, err = svc.validateDeleteFaultRequest("vehicle-1", DeleteFaultRequest{GuideManual: " ", SectionID: "before", ItemIndex: 0})
	requireServiceError(t, err, ErrInvalidGuideManual)

	_, err = svc.validateDeleteFaultRequest("vehicle-1", DeleteFaultRequest{GuideManual: "pmcs_sbs/hmmwv/file.json", SectionID: " ", ItemIndex: 0})
	requireServiceError(t, err, ErrInvalidRequest)

	_, err = svc.validateDeleteFaultRequest("vehicle-1", DeleteFaultRequest{GuideManual: "pmcs_sbs/hmmwv/file.json", SectionID: "before", ItemIndex: -1})
	requireServiceError(t, err, ErrInvalidRequest)
}

func TestUpsertFaultReturnsMappedResponse(t *testing.T) {
	now := time.Now().UTC()
	stub := &repoStub{savedFault: &model.PmcsSbsFaults{
		EquipmentID:      "vehicle-1",
		GuideManual:      "pmcs_sbs/hmmwv/file.json",
		SectionID:        "before",
		ItemIndex:        0,
		ItemNo:           "1",
		Status:           "x",
		FaultText:        "leak",
		CorrectiveAction: "",
		CreatedAt:        now,
		UpdatedAt:        now,
	}}
	svc := NewService(stub)

	resp, err := svc.UpsertFault(requireUser(), "vehicle-1", FaultRequest{
		GuideManual: "pmcs_sbs/hmmwv/file.json",
		SectionID:   "before",
		ItemIndex:   0,
		ItemNo:      "1",
		Status:      "X",
		FaultText:   "leak",
	})

	require.NoError(t, err)
	require.Equal(t, "vehicle-1", stub.capturedFault.EquipmentID)
	require.Equal(t, "pmcs_sbs/hmmwv/file.json", stub.capturedFault.GuideManual)
	require.Equal(t, "x", stub.capturedFault.Status)
	require.Equal(t, "vehicle-1", resp.EquipmentID)
	require.Equal(t, "pmcs_sbs/hmmwv/file.json", resp.GuideManual)
	require.Equal(t, "x", resp.Status)
}

func TestDeleteFaultPassesValidatedKey(t *testing.T) {
	stub := &repoStub{}
	svc := NewService(stub)

	err := svc.DeleteFault(requireUser(), " vehicle-1 ", DeleteFaultRequest{GuideManual: " pmcs_sbs/hmmwv/file.json ", SectionID: " before ", ItemIndex: 0})

	require.NoError(t, err)
	require.Equal(t, "vehicle-1", stub.capturedDelete.EquipmentID)
	require.Equal(t, "pmcs_sbs/hmmwv/file.json", stub.capturedDelete.GuideManual)
	require.Equal(t, "before", stub.capturedDelete.SectionID)
	require.Equal(t, int32(0), stub.capturedDelete.ItemIndex)
}
