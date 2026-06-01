package pmcs_sbs_progress_test

import (
	"testing"

	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/pmcs_sbs_progress"

	"github.com/stretchr/testify/require"
)

func TestRepositoryEquipmentLifecycle(t *testing.T) {
	clearPmcsSbsTables(t, testDB)
	user := testUser("pmcs-equipment-user")
	ensureUser(t, testDB, user)
	repo := pmcs_sbs_progress.NewRepository(testDB)
	equipment := sampleEquipment(user)

	saved, err := repo.UpsertEquipment(user, equipment)
	require.NoError(t, err)
	require.Equal(t, equipment.ID, saved.ID)
	require.Equal(t, user.UserID, saved.UserUID)

	list, err := repo.ListEquipment(user)
	require.NoError(t, err)
	require.Len(t, list, 1)

	aggregate, err := repo.GetEquipmentAggregate(user, equipment.ID.String())
	require.NoError(t, err)
	require.Equal(t, equipment.ID, aggregate.Equipment.ID)
	require.Empty(t, aggregate.Completions)
	require.Empty(t, aggregate.Faults)

	require.NoError(t, repo.DeleteEquipment(user, equipment.ID.String()))
	list, err = repo.ListEquipment(user)
	require.NoError(t, err)
	require.Empty(t, list)
}

func TestRepositoryGetEquipmentAggregateIncludesChildren(t *testing.T) {
	clearPmcsSbsTables(t, testDB)
	user := testUser("pmcs-equipment-aggregate")
	ensureUser(t, testDB, user)
	repo := pmcs_sbs_progress.NewRepository(testDB)
	equipment := sampleEquipment(user)

	_, err := repo.UpsertEquipment(user, equipment)
	require.NoError(t, err)
	_, err = repo.UpsertCompletion(user, sampleCompletion(equipment.ID))
	require.NoError(t, err)
	_, err = repo.UpsertFault(user, sampleFault(equipment.ID))
	require.NoError(t, err)

	aggregate, err := repo.GetEquipmentAggregate(user, equipment.ID.String())
	require.NoError(t, err)
	require.Len(t, aggregate.Completions, 1)
	require.Len(t, aggregate.Faults, 1)
}

func TestRepositoryEquipmentUserIsolation(t *testing.T) {
	clearPmcsSbsTables(t, testDB)
	owner := testUser("pmcs-owner")
	other := testUser("pmcs-other")
	ensureUser(t, testDB, owner)
	ensureUser(t, testDB, other)
	repo := pmcs_sbs_progress.NewRepository(testDB)
	equipment := sampleEquipment(owner)

	_, err := repo.UpsertEquipment(owner, equipment)
	require.NoError(t, err)

	_, err = repo.GetEquipmentAggregate(other, equipment.ID.String())
	require.ErrorIs(t, err, pmcs_sbs_progress.ErrNotFound)

	err = repo.DeleteEquipment(other, equipment.ID.String())
	require.ErrorIs(t, err, pmcs_sbs_progress.ErrNotFound)
}

func TestRepositoryCompletionLifecycle(t *testing.T) {
	clearPmcsSbsTables(t, testDB)
	user := testUser("pmcs-completion-user")
	ensureUser(t, testDB, user)
	repo := pmcs_sbs_progress.NewRepository(testDB)
	equipment := sampleEquipment(user)
	_, err := repo.UpsertEquipment(user, equipment)
	require.NoError(t, err)

	completion := sampleCompletion(equipment.ID)
	saved, err := repo.UpsertCompletion(user, completion)
	require.NoError(t, err)
	require.True(t, saved.IsComplete)

	aggregate, err := repo.GetEquipmentAggregate(user, equipment.ID.String())
	require.NoError(t, err)
	require.Len(t, aggregate.Completions, 1)

	require.NoError(t, repo.DeleteCompletion(user, equipment.ID.String(), completion.SectionID, completion.ItemIndex, completion.StepID))
	aggregate, err = repo.GetEquipmentAggregate(user, equipment.ID.String())
	require.NoError(t, err)
	require.Empty(t, aggregate.Completions)
}

func TestRepositoryBatchCompletions(t *testing.T) {
	clearPmcsSbsTables(t, testDB)
	user := testUser("pmcs-batch-completions")
	ensureUser(t, testDB, user)
	repo := pmcs_sbs_progress.NewRepository(testDB)
	equipment := sampleEquipment(user)
	_, err := repo.UpsertEquipment(user, equipment)
	require.NoError(t, err)

	first := sampleCompletion(equipment.ID)
	second := sampleCompletion(equipment.ID)
	second.StepID = "1-b"
	result, err := repo.BatchCompletions(user, equipment.ID.String(), []model.PmcsSbsCompletions{first, second}, nil)
	require.NoError(t, err)
	require.Equal(t, int64(2), result.UpsertedCount)
	require.Equal(t, int64(0), result.DeletedCount)

	aggregate, err := repo.GetEquipmentAggregate(user, equipment.ID.String())
	require.NoError(t, err)
	require.Len(t, aggregate.Completions, 2)

	second.ItemNo = "1 updated"
	result, err = repo.BatchCompletions(user, equipment.ID.String(), []model.PmcsSbsCompletions{second}, []pmcs_sbs_progress.CompletionKey{{
		EquipmentID: equipment.ID.String(),
		SectionID:   first.SectionID,
		ItemIndex:   first.ItemIndex,
		StepID:      first.StepID,
	}})
	require.NoError(t, err)
	require.Equal(t, int64(1), result.UpsertedCount)
	require.Equal(t, int64(1), result.DeletedCount)

	aggregate, err = repo.GetEquipmentAggregate(user, equipment.ID.String())
	require.NoError(t, err)
	require.Len(t, aggregate.Completions, 1)
	require.Equal(t, "1 updated", aggregate.Completions[0].ItemNo)
	require.Equal(t, "1-b", aggregate.Completions[0].StepID)
}

func TestRepositoryBatchCompletionsRequiresOwnedEquipment(t *testing.T) {
	clearPmcsSbsTables(t, testDB)
	owner := testUser("pmcs-batch-owner")
	other := testUser("pmcs-batch-other")
	ensureUser(t, testDB, owner)
	ensureUser(t, testDB, other)
	repo := pmcs_sbs_progress.NewRepository(testDB)
	equipment := sampleEquipment(owner)
	_, err := repo.UpsertEquipment(owner, equipment)
	require.NoError(t, err)

	_, err = repo.BatchCompletions(other, equipment.ID.String(), []model.PmcsSbsCompletions{sampleCompletion(equipment.ID)}, nil)
	require.ErrorIs(t, err, pmcs_sbs_progress.ErrNotFound)
}

func TestRepositoryFaultLifecycle(t *testing.T) {
	clearPmcsSbsTables(t, testDB)
	user := testUser("pmcs-fault-user")
	ensureUser(t, testDB, user)
	repo := pmcs_sbs_progress.NewRepository(testDB)
	equipment := sampleEquipment(user)
	_, err := repo.UpsertEquipment(user, equipment)
	require.NoError(t, err)

	fault := sampleFault(equipment.ID)
	saved, err := repo.UpsertFault(user, fault)
	require.NoError(t, err)
	require.Equal(t, "leak", saved.FaultText)

	fault.FaultText = "updated leak"
	updated, err := repo.UpsertFault(user, fault)
	require.NoError(t, err)
	require.Equal(t, "updated leak", updated.FaultText)
	require.Equal(t, saved.CreatedAt, updated.CreatedAt)

	aggregate, err := repo.GetEquipmentAggregate(user, equipment.ID.String())
	require.NoError(t, err)
	require.Len(t, aggregate.Faults, 1)

	require.NoError(t, repo.DeleteFault(user, equipment.ID.String(), fault.SectionID, fault.ItemIndex))
	aggregate, err = repo.GetEquipmentAggregate(user, equipment.ID.String())
	require.NoError(t, err)
	require.Empty(t, aggregate.Faults)
}

func TestRepositoryChildWritesRequireOwnedEquipment(t *testing.T) {
	clearPmcsSbsTables(t, testDB)
	owner := testUser("pmcs-child-owner")
	other := testUser("pmcs-child-other")
	ensureUser(t, testDB, owner)
	ensureUser(t, testDB, other)
	repo := pmcs_sbs_progress.NewRepository(testDB)
	equipment := sampleEquipment(owner)
	_, err := repo.UpsertEquipment(owner, equipment)
	require.NoError(t, err)

	_, err = repo.UpsertCompletion(other, sampleCompletion(equipment.ID))
	require.ErrorIs(t, err, pmcs_sbs_progress.ErrNotFound)

	_, err = repo.UpsertFault(other, sampleFault(equipment.ID))
	require.ErrorIs(t, err, pmcs_sbs_progress.ErrNotFound)
}

func TestRepositorySyncAppliesChangeSet(t *testing.T) {
	clearPmcsSbsTables(t, testDB)
	user := testUser("pmcs-sync-user")
	ensureUser(t, testDB, user)
	repo := pmcs_sbs_progress.NewRepository(testDB)
	equipment := sampleEquipment(user)
	completion := sampleCompletion(equipment.ID)
	fault := sampleFault(equipment.ID)

	result, err := repo.Sync(user, pmcs_sbs_progress.SyncChangeSet{
		UpsertEquipment:   []model.PmcsSbsEquipment{equipment},
		UpsertCompletions: []model.PmcsSbsCompletions{completion},
		UpsertFaults:      []model.PmcsSbsFaults{fault},
	})

	require.NoError(t, err)
	require.Len(t, result.Equipment, 1)
	require.Len(t, result.Equipment[0].Completions, 1)
	require.Len(t, result.Equipment[0].Faults, 1)
}

func TestRepositorySyncDeletesRows(t *testing.T) {
	clearPmcsSbsTables(t, testDB)
	user := testUser("pmcs-sync-delete")
	ensureUser(t, testDB, user)
	repo := pmcs_sbs_progress.NewRepository(testDB)
	equipment := sampleEquipment(user)
	_, err := repo.UpsertEquipment(user, equipment)
	require.NoError(t, err)
	_, err = repo.UpsertCompletion(user, sampleCompletion(equipment.ID))
	require.NoError(t, err)
	_, err = repo.UpsertFault(user, sampleFault(equipment.ID))
	require.NoError(t, err)

	result, err := repo.Sync(user, pmcs_sbs_progress.SyncChangeSet{
		DeleteCompletions: []pmcs_sbs_progress.CompletionKey{{
			EquipmentID: equipment.ID.String(),
			SectionID:   "before",
			ItemIndex:   0,
			StepID:      "1-a",
		}},
		DeleteFaults: []pmcs_sbs_progress.FaultKey{{
			EquipmentID: equipment.ID.String(),
			SectionID:   "before",
			ItemIndex:   0,
		}},
	})

	require.NoError(t, err)
	require.Len(t, result.Equipment, 1)
	require.Empty(t, result.Equipment[0].Completions)
	require.Empty(t, result.Equipment[0].Faults)
}

func TestRepositorySyncDeletesEquipment(t *testing.T) {
	clearPmcsSbsTables(t, testDB)
	user := testUser("pmcs-sync-del-equip")
	ensureUser(t, testDB, user)
	repo := pmcs_sbs_progress.NewRepository(testDB)
	equipment := sampleEquipment(user)
	_, err := repo.UpsertEquipment(user, equipment)
	require.NoError(t, err)
	_, err = repo.UpsertCompletion(user, sampleCompletion(equipment.ID))
	require.NoError(t, err)

	result, err := repo.Sync(user, pmcs_sbs_progress.SyncChangeSet{
		DeleteEquipmentIDs: []string{equipment.ID.String()},
	})

	require.NoError(t, err)
	require.Empty(t, result.Equipment)
	require.Equal(t, []string{equipment.ID.String()}, result.DeletedEquipmentIDs)

	list, err := repo.ListEquipment(user)
	require.NoError(t, err)
	require.Empty(t, list)
}
