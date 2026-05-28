package pmcs_sbs_progress_test

import (
	"testing"

	"miltechserver/api/pmcs_sbs_progress"

	"github.com/stretchr/testify/require"
)

func TestRepositoryEquipmentLifecycle(t *testing.T) {
	clearPmcsSbsTables(t, testDB)
	user := testUser("pmcs-equipment-user")
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
	repo := pmcs_sbs_progress.NewRepository(testDB)
	equipment := sampleEquipment(owner)

	_, err := repo.UpsertEquipment(owner, equipment)
	require.NoError(t, err)

	_, err = repo.GetEquipmentAggregate(other, equipment.ID.String())
	require.ErrorIs(t, err, pmcs_sbs_progress.ErrNotFound)

	err = repo.DeleteEquipment(other, equipment.ID.String())
	require.ErrorIs(t, err, pmcs_sbs_progress.ErrNotFound)
}
