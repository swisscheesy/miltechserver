package pmcs_sbs_progress

import (
	"errors"
	"testing"

	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/bootstrap"

	"github.com/stretchr/testify/require"
)

type repoStub struct{}

func (repo *repoStub) ListEquipment(user *bootstrap.User) ([]model.PmcsSbsEquipment, error) {
	return nil, nil
}

func (repo *repoStub) GetEquipmentAggregate(user *bootstrap.User, equipmentID string) (*EquipmentAggregate, error) {
	return nil, nil
}

func (repo *repoStub) UpsertEquipment(user *bootstrap.User, equipment model.PmcsSbsEquipment) (*model.PmcsSbsEquipment, error) {
	return &equipment, nil
}

func (repo *repoStub) DeleteEquipment(user *bootstrap.User, equipmentID string) error {
	return nil
}

func (repo *repoStub) UpsertCompletion(user *bootstrap.User, completion model.PmcsSbsCompletions) (*model.PmcsSbsCompletions, error) {
	return &completion, nil
}

func (repo *repoStub) BatchCompletions(user *bootstrap.User, equipmentID string, upserts []model.PmcsSbsCompletions, deletes []CompletionKey) (*BatchCompletionsResult, error) {
	return &BatchCompletionsResult{UpsertedCount: int64(len(upserts)), DeletedCount: int64(len(deletes))}, nil
}

func (repo *repoStub) DeleteCompletion(user *bootstrap.User, equipmentID string, sectionID string, itemIndex int32, stepID string) error {
	return nil
}

func (repo *repoStub) UpsertFault(user *bootstrap.User, fault model.PmcsSbsFaults) (*model.PmcsSbsFaults, error) {
	return &fault, nil
}

func (repo *repoStub) DeleteFault(user *bootstrap.User, equipmentID string, sectionID string, itemIndex int32) error {
	return nil
}

func (repo *repoStub) Sync(user *bootstrap.User, changeSet SyncChangeSet) (*SyncResult, error) {
	return &SyncResult{}, nil
}

func TestValidateEquipmentRequest(t *testing.T) {
	svc := NewService(&repoStub{})

	req, err := svc.validateEquipmentRequest("550e8400-e29b-41d4-a716-446655440000", EquipmentRequest{
		EquipmentManual: " pmcs_sbs/hmmwv/basic.json ",
		Admin:           " A12 ",
		Serial:          " SER ",
		Uic:             " UIC ",
	})

	require.NoError(t, err)
	require.Equal(t, "550e8400-e29b-41d4-a716-446655440000", req.ID.String())
	require.Equal(t, "pmcs_sbs/hmmwv/basic.json", req.EquipmentManual)
	require.Equal(t, "A12", req.Admin)
	require.Equal(t, "SER", req.Serial)
	require.Equal(t, "UIC", req.Uic)
}

func TestValidateEquipmentRequestRejectsInvalidValues(t *testing.T) {
	svc := NewService(&repoStub{})

	cases := []struct {
		name        string
		equipmentID string
		req         EquipmentRequest
		want        error
	}{
		{
			name:        "invalid uuid",
			equipmentID: "not-a-uuid",
			req:         EquipmentRequest{EquipmentManual: "pmcs_sbs/hmmwv/basic.json", Admin: "A12"},
			want:        ErrInvalidID,
		},
		{
			name:        "bad blob prefix",
			equipmentID: "550e8400-e29b-41d4-a716-446655440000",
			req:         EquipmentRequest{EquipmentManual: "pmcs/hmmwv/basic.json", Admin: "A12"},
			want:        ErrInvalidBlobPath,
		},
		{
			name:        "bad blob extension",
			equipmentID: "550e8400-e29b-41d4-a716-446655440000",
			req:         EquipmentRequest{EquipmentManual: "pmcs_sbs/hmmwv/basic.pdf", Admin: "A12"},
			want:        ErrInvalidBlobPath,
		},
		{
			name:        "path traversal",
			equipmentID: "550e8400-e29b-41d4-a716-446655440000",
			req:         EquipmentRequest{EquipmentManual: "pmcs_sbs/../secret.json", Admin: "A12"},
			want:        ErrInvalidBlobPath,
		},
		{
			name:        "missing admin",
			equipmentID: "550e8400-e29b-41d4-a716-446655440000",
			req:         EquipmentRequest{EquipmentManual: "pmcs_sbs/hmmwv/basic.json", Admin: " "},
			want:        ErrInvalidRequest,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.validateEquipmentRequest(tc.equipmentID, tc.req)
			requireServiceError(t, err, tc.want)
		})
	}
}

func TestValidateCompletionRequest(t *testing.T) {
	svc := NewService(&repoStub{})

	completion, err := svc.validateCompletionRequest("550e8400-e29b-41d4-a716-446655440000", CompletionRequest{
		SectionID: " before ",
		ItemIndex: 0,
		ItemNo:    " 1 ",
		StepID:    " 1-a ",
	})

	require.NoError(t, err)
	require.Equal(t, "550e8400-e29b-41d4-a716-446655440000", completion.EquipmentID.String())
	require.Equal(t, "before", completion.SectionID)
	require.Equal(t, int32(0), completion.ItemIndex)
	require.Equal(t, "1", completion.ItemNo)
	require.Equal(t, "1-a", completion.StepID)
	require.True(t, completion.IsComplete)
}

func TestValidateCompletionRequestRejectsInvalidValues(t *testing.T) {
	svc := NewService(&repoStub{})

	_, err := svc.validateCompletionRequest("bad", CompletionRequest{SectionID: "before", ItemIndex: 0, ItemNo: "1", StepID: "1-a"})
	requireServiceError(t, err, ErrInvalidID)

	_, err = svc.validateCompletionRequest("550e8400-e29b-41d4-a716-446655440000", CompletionRequest{SectionID: "", ItemIndex: 0, ItemNo: "1", StepID: "1-a"})
	requireServiceError(t, err, ErrInvalidRequest)

	_, err = svc.validateCompletionRequest("550e8400-e29b-41d4-a716-446655440000", CompletionRequest{SectionID: "before", ItemIndex: 0, ItemNo: "", StepID: "1-a"})
	requireServiceError(t, err, ErrInvalidRequest)

	_, err = svc.validateCompletionRequest("550e8400-e29b-41d4-a716-446655440000", CompletionRequest{SectionID: "before", ItemIndex: 0, ItemNo: "1", StepID: ""})
	requireServiceError(t, err, ErrInvalidRequest)

	_, err = svc.validateCompletionRequest("550e8400-e29b-41d4-a716-446655440000", CompletionRequest{SectionID: "before", ItemIndex: -1, ItemNo: "1", StepID: "1-a"})
	requireServiceError(t, err, ErrInvalidRequest)
}

func TestBuildBatchCompletionsChangeSet(t *testing.T) {
	svc := NewService(&repoStub{})

	upserts, deletes, err := svc.buildBatchCompletionsChangeSet("550E8400-E29B-41D4-A716-446655440000", BatchCompletionsRequest{
		UpsertCompletions: []CompletionRequest{{
			SectionID: " before ",
			ItemIndex: 0,
			ItemNo:    " 1 ",
			StepID:    " 1-a ",
		}},
		DeleteCompletions: []DeleteCompletionRequest{{
			SectionID: " before ",
			ItemIndex: 0,
			StepID:    " 1-b ",
		}},
	})

	require.NoError(t, err)
	require.Len(t, upserts, 1)
	require.Len(t, deletes, 1)
	require.Equal(t, "550e8400-e29b-41d4-a716-446655440000", upserts[0].EquipmentID.String())
	require.Equal(t, "550e8400-e29b-41d4-a716-446655440000", deletes[0].EquipmentID)
	require.Equal(t, "before", upserts[0].SectionID)
	require.Equal(t, "before", deletes[0].SectionID)
}

func TestBuildBatchCompletionsChangeSetAllowsNoOp(t *testing.T) {
	svc := NewService(&repoStub{})

	upserts, deletes, err := svc.buildBatchCompletionsChangeSet("550e8400-e29b-41d4-a716-446655440000", BatchCompletionsRequest{})

	require.NoError(t, err)
	require.Empty(t, upserts)
	require.Empty(t, deletes)
}

func TestBuildBatchCompletionsChangeSetRejectsInvalidValues(t *testing.T) {
	svc := NewService(&repoStub{})

	_, _, err := svc.buildBatchCompletionsChangeSet("bad", BatchCompletionsRequest{})
	requireServiceError(t, err, ErrInvalidID)

	tooMany := BatchCompletionsRequest{UpsertCompletions: make([]CompletionRequest, maxBatchCompletionChanges+1)}
	for i := range tooMany.UpsertCompletions {
		tooMany.UpsertCompletions[i] = CompletionRequest{SectionID: "before", ItemIndex: int32(i), ItemNo: "1", StepID: "a"}
	}
	_, _, err = svc.buildBatchCompletionsChangeSet("550e8400-e29b-41d4-a716-446655440000", tooMany)
	requireServiceError(t, err, ErrInvalidRequest)

	_, _, err = svc.buildBatchCompletionsChangeSet("550e8400-e29b-41d4-a716-446655440000", BatchCompletionsRequest{
		UpsertCompletions: []CompletionRequest{{
			SectionID: "before",
			ItemIndex: 0,
			ItemNo:    "1",
			StepID:    "1-a",
		}},
		DeleteCompletions: []DeleteCompletionRequest{{
			SectionID: "before",
			ItemIndex: 0,
			StepID:    "1-a",
		}},
	})
	requireServiceError(t, err, ErrInvalidSyncRequest)

	_, _, err = svc.buildBatchCompletionsChangeSet("550e8400-e29b-41d4-a716-446655440000", BatchCompletionsRequest{
		UpsertCompletions: []CompletionRequest{
			{SectionID: "before", ItemIndex: 0, ItemNo: "1", StepID: "1-a"},
			{SectionID: " before ", ItemIndex: 0, ItemNo: "1", StepID: " 1-a "},
		},
	})
	requireServiceError(t, err, ErrInvalidSyncRequest)
}

func TestBatchCompletionsReturnsCounts(t *testing.T) {
	svc := NewService(&repoStub{})

	resp, err := svc.BatchCompletions(requireUser(), "550e8400-e29b-41d4-a716-446655440000", BatchCompletionsRequest{
		UpsertCompletions: []CompletionRequest{
			{SectionID: "before", ItemIndex: 0, ItemNo: "1", StepID: "1-a"},
			{SectionID: "before", ItemIndex: 0, ItemNo: "1", StepID: "1-b"},
		},
		DeleteCompletions: []DeleteCompletionRequest{
			{SectionID: "before", ItemIndex: 0, StepID: "1-c"},
		},
	})

	require.NoError(t, err)
	require.Equal(t, int64(2), resp.UpsertedCount)
	require.Equal(t, int64(1), resp.DeletedCount)
}

func TestValidateFaultRequest(t *testing.T) {
	svc := NewService(&repoStub{})

	fault, err := svc.validateFaultRequest("550e8400-e29b-41d4-a716-446655440000", FaultRequest{
		SectionID:        " before ",
		ItemIndex:        0,
		ItemNo:           " 1 ",
		Status:           "X",
		FaultText:        " leak ",
		CorrectiveAction: " tighten ",
	})

	require.NoError(t, err)
	require.Equal(t, "550e8400-e29b-41d4-a716-446655440000", fault.EquipmentID.String())
	require.Equal(t, "before", fault.SectionID)
	require.Equal(t, "1", fault.ItemNo)
	require.Equal(t, "x", fault.Status)
	require.Equal(t, "leak", fault.FaultText)
	require.Equal(t, "tighten", fault.CorrectiveAction)
	require.False(t, fault.CreatedAt.IsZero())
	require.False(t, fault.UpdatedAt.IsZero())
}

func TestValidateFaultRequestAcceptsAllowedStatuses(t *testing.T) {
	svc := NewService(&repoStub{})

	testCases := map[string]string{
		"X":     "x",
		"x":     "x",
		"/":     "slash",
		"slash": "slash",
		"-":     "dash",
		"dash":  "dash",
	}
	for status, expected := range testCases {
		t.Run(status, func(t *testing.T) {
			fault, err := svc.validateFaultRequest("550e8400-e29b-41d4-a716-446655440000", FaultRequest{
				SectionID: "before",
				ItemIndex: 0,
				ItemNo:    "1",
				Status:    status,
				FaultText: "leak",
			})
			require.NoError(t, err)
			require.Equal(t, expected, fault.Status)
		})
	}
}

func TestValidateFaultRequestRejectsInvalidValues(t *testing.T) {
	svc := NewService(&repoStub{})

	_, err := svc.validateFaultRequest("bad", FaultRequest{SectionID: "before", ItemIndex: 0, ItemNo: "1", Status: "X", FaultText: "leak"})
	requireServiceError(t, err, ErrInvalidID)

	_, err = svc.validateFaultRequest("550e8400-e29b-41d4-a716-446655440000", FaultRequest{SectionID: "before", ItemIndex: 0, ItemNo: "1", Status: "BAD", FaultText: "leak"})
	requireServiceError(t, err, ErrInvalidStatus)

	_, err = svc.validateFaultRequest("550e8400-e29b-41d4-a716-446655440000", FaultRequest{SectionID: "before", ItemIndex: 0, ItemNo: "1", Status: "X", FaultText: " "})
	requireServiceError(t, err, ErrInvalidRequest)

	_, err = svc.validateFaultRequest("550e8400-e29b-41d4-a716-446655440000", FaultRequest{SectionID: "", ItemIndex: 0, ItemNo: "1", Status: "X", FaultText: "leak"})
	requireServiceError(t, err, ErrInvalidRequest)

	_, err = svc.validateFaultRequest("550e8400-e29b-41d4-a716-446655440000", FaultRequest{SectionID: "before", ItemIndex: -1, ItemNo: "1", Status: "X", FaultText: "leak"})
	requireServiceError(t, err, ErrInvalidRequest)
}

func TestValidateSyncRequestRejectsContradictions(t *testing.T) {
	svc := NewService(&repoStub{})

	err := svc.validateSyncRequest(SyncRequest{
		UpsertEquipment: []SyncEquipmentRequest{{
			ID:              "550E8400-E29B-41D4-A716-446655440000",
			EquipmentManual: "pmcs_sbs/hmmwv/basic.json",
			Admin:           "A12",
		}},
		DeleteEquipmentIDs: []string{"550e8400-e29b-41d4-a716-446655440000"},
	})
	requireServiceError(t, err, ErrInvalidSyncRequest)

	err = svc.validateSyncRequest(SyncRequest{
		DeleteEquipmentIDs: []string{"550e8400-e29b-41d4-a716-446655440000"},
		UpsertCompletions: []SyncCompletionRequest{{
			EquipmentID: "550e8400-e29b-41d4-a716-446655440000",
			SectionID:   "before",
			ItemIndex:   0,
			ItemNo:      "1",
			StepID:      "1-a",
		}},
	})
	requireServiceError(t, err, ErrInvalidSyncRequest)

	err = svc.validateSyncRequest(SyncRequest{
		DeleteEquipmentIDs: []string{"550E8400-E29B-41D4-A716-446655440000"},
		UpsertCompletions: []SyncCompletionRequest{{
			EquipmentID: "550e8400-e29b-41d4-a716-446655440000",
			SectionID:   "before",
			ItemIndex:   0,
			ItemNo:      "1",
			StepID:      "1-a",
		}},
	})
	requireServiceError(t, err, ErrInvalidSyncRequest)

	err = svc.validateSyncRequest(SyncRequest{
		DeleteEquipmentIDs: []string{"550e8400-e29b-41d4-a716-446655440000"},
		UpsertFaults: []SyncFaultRequest{{
			EquipmentID: "550e8400-e29b-41d4-a716-446655440000",
			SectionID:   "before",
			ItemIndex:   0,
			ItemNo:      "1",
			Status:      "X",
			FaultText:   "leak",
		}},
	})
	requireServiceError(t, err, ErrInvalidSyncRequest)

	err = svc.validateSyncRequest(SyncRequest{
		DeleteEquipmentIDs: []string{"550E8400-E29B-41D4-A716-446655440000"},
		UpsertFaults: []SyncFaultRequest{{
			EquipmentID: "550e8400-e29b-41d4-a716-446655440000",
			SectionID:   "before",
			ItemIndex:   0,
			ItemNo:      "1",
			Status:      "X",
			FaultText:   "leak",
		}},
	})
	requireServiceError(t, err, ErrInvalidSyncRequest)

	err = svc.validateSyncRequest(SyncRequest{
		UpsertCompletions: []SyncCompletionRequest{{
			EquipmentID: "550e8400-e29b-41d4-a716-446655440000",
			SectionID:   "before",
			ItemIndex:   0,
			ItemNo:      "1",
			StepID:      "1-a",
		}},
		DeleteCompletions: []SyncDeleteCompletionRequest{{
			EquipmentID: "550e8400-e29b-41d4-a716-446655440000",
			SectionID:   "before",
			ItemIndex:   0,
			StepID:      "1-a",
		}},
	})
	requireServiceError(t, err, ErrInvalidSyncRequest)

	err = svc.validateSyncRequest(SyncRequest{
		UpsertCompletions: []SyncCompletionRequest{{
			EquipmentID: "550E8400-E29B-41D4-A716-446655440000",
			SectionID:   "before",
			ItemIndex:   0,
			ItemNo:      "1",
			StepID:      "1-a",
		}},
		DeleteCompletions: []SyncDeleteCompletionRequest{{
			EquipmentID: "550e8400-e29b-41d4-a716-446655440000",
			SectionID:   "before",
			ItemIndex:   0,
			StepID:      "1-a",
		}},
	})
	requireServiceError(t, err, ErrInvalidSyncRequest)

	err = svc.validateSyncRequest(SyncRequest{
		UpsertFaults: []SyncFaultRequest{{
			EquipmentID: "550e8400-e29b-41d4-a716-446655440000",
			SectionID:   "before",
			ItemIndex:   0,
			ItemNo:      "1",
			Status:      "X",
			FaultText:   "leak",
		}},
		DeleteFaults: []SyncDeleteFaultRequest{{
			EquipmentID: "550e8400-e29b-41d4-a716-446655440000",
			SectionID:   "before",
			ItemIndex:   0,
		}},
	})
	requireServiceError(t, err, ErrInvalidSyncRequest)

	err = svc.validateSyncRequest(SyncRequest{
		UpsertFaults: []SyncFaultRequest{{
			EquipmentID: "550E8400-E29B-41D4-A716-446655440000",
			SectionID:   "before",
			ItemIndex:   0,
			ItemNo:      "1",
			Status:      "X",
			FaultText:   "leak",
		}},
		DeleteFaults: []SyncDeleteFaultRequest{{
			EquipmentID: "550e8400-e29b-41d4-a716-446655440000",
			SectionID:   "before",
			ItemIndex:   0,
		}},
	})
	requireServiceError(t, err, ErrInvalidSyncRequest)
}

func TestBuildSyncChangeSet(t *testing.T) {
	svc := NewService(&repoStub{})
	user := requireUser()

	changeSet, err := svc.buildSyncChangeSet(user, SyncRequest{
		UpsertEquipment: []SyncEquipmentRequest{{
			ID:              "550e8400-e29b-41d4-a716-446655440000",
			EquipmentManual: "pmcs_sbs/hmmwv/basic.json",
			Admin:           "A12",
		}},
		UpsertCompletions: []SyncCompletionRequest{{
			EquipmentID: "550e8400-e29b-41d4-a716-446655440000",
			SectionID:   "before",
			ItemIndex:   0,
			ItemNo:      "1",
			StepID:      "1-a",
		}},
		UpsertFaults: []SyncFaultRequest{{
			EquipmentID: "550e8400-e29b-41d4-a716-446655440000",
			SectionID:   "before",
			ItemIndex:   0,
			ItemNo:      "1",
			Status:      "X",
			FaultText:   "leak",
		}},
	})

	require.NoError(t, err)
	require.Len(t, changeSet.UpsertEquipment, 1)
	require.Len(t, changeSet.UpsertCompletions, 1)
	require.Len(t, changeSet.UpsertFaults, 1)
	require.Equal(t, user.UserID, changeSet.UpsertEquipment[0].UserUID)
}

func requireUser() *bootstrap.User {
	return &bootstrap.User{UserID: "user-1", Username: "tester", Email: "user-1@example.com"}
}

func requireServiceError(t *testing.T, err error, target error) {
	t.Helper()
	require.True(t, errors.Is(err, target), "expected %v, got %v", target, err)
}
