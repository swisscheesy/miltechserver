package pmcs_sbs_progress_test

import (
	"testing"
	"time"

	"miltechserver/api/pmcs_sbs_progress"

	"github.com/stretchr/testify/require"
)

func TestRepositoryMemberCanListSaveAndDeleteFaults(t *testing.T) {
	clearPmcsSbsTables(t, testDB)
	user := testUser("pmcs-fault-member")
	ensureUser(t, testDB, user)
	shopID := createShopWithMember(t, testDB, user, "member")
	vehicleID := createShopVehicle(t, testDB, shopID, user, "A1")
	repo := pmcs_sbs_progress.NewRepository(testDB)

	saved, err := repo.UpsertFault(user, sampleFault(vehicleID))
	require.NoError(t, err)
	require.Equal(t, vehicleID, saved.EquipmentID)
	require.Equal(t, "leak", saved.FaultText)

	list, err := repo.ListFaults(user, vehicleID, "pmcs_sbs/hmmwv/file.json")
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, "before", list[0].SectionID)

	err = repo.DeleteFault(user, pmcs_sbs_progress.FaultKey{
		EquipmentID: vehicleID,
		GuideManual: "pmcs_sbs/hmmwv/file.json",
		SectionID:   "before",
		ItemIndex:   0,
	})
	require.NoError(t, err)

	list, err = repo.ListFaults(user, vehicleID, "pmcs_sbs/hmmwv/file.json")
	require.NoError(t, err)
	require.Empty(t, list)
}

func TestRepositoryNonMemberCannotAccessFaults(t *testing.T) {
	clearPmcsSbsTables(t, testDB)
	owner := testUser("pmcs-fault-owner")
	other := testUser("pmcs-fault-other")
	ensureUser(t, testDB, owner)
	ensureUser(t, testDB, other)
	shopID := createShopWithMember(t, testDB, owner, "admin")
	vehicleID := createShopVehicle(t, testDB, shopID, owner, "A2")
	repo := pmcs_sbs_progress.NewRepository(testDB)

	_, err := repo.ListFaults(other, vehicleID, "pmcs_sbs/hmmwv/file.json")
	require.ErrorIs(t, err, pmcs_sbs_progress.ErrNotFound)

	_, err = repo.UpsertFault(other, sampleFault(vehicleID))
	require.ErrorIs(t, err, pmcs_sbs_progress.ErrNotFound)

	err = repo.DeleteFault(other, pmcs_sbs_progress.FaultKey{EquipmentID: vehicleID, GuideManual: "pmcs_sbs/hmmwv/file.json", SectionID: "before", ItemIndex: 0})
	require.ErrorIs(t, err, pmcs_sbs_progress.ErrNotFound)
}

func TestRepositoryMissingVehicleReturnsNotFound(t *testing.T) {
	clearPmcsSbsTables(t, testDB)
	user := testUser("pmcs-fault-missing")
	ensureUser(t, testDB, user)
	repo := pmcs_sbs_progress.NewRepository(testDB)

	_, err := repo.ListFaults(user, "missing-vehicle", "pmcs_sbs/hmmwv/file.json")
	require.ErrorIs(t, err, pmcs_sbs_progress.ErrNotFound)

	_, err = repo.UpsertFault(user, sampleFault("missing-vehicle"))
	require.ErrorIs(t, err, pmcs_sbs_progress.ErrNotFound)

	err = repo.DeleteFault(user, pmcs_sbs_progress.FaultKey{EquipmentID: "missing-vehicle", GuideManual: "pmcs_sbs/hmmwv/file.json", SectionID: "before", ItemIndex: 0})
	require.ErrorIs(t, err, pmcs_sbs_progress.ErrNotFound)
}

func TestRepositoryAnyShopMemberCanManageFaults(t *testing.T) {
	clearPmcsSbsTables(t, testDB)
	owner := testUser("pmcs-fault-shop-owner")
	member := testUser("pmcs-fault-shop-member")
	ensureUser(t, testDB, owner)
	ensureUser(t, testDB, member)
	shopID := createShopWithMember(t, testDB, owner, "admin")
	addShopMember(t, testDB, shopID, member, "member")
	vehicleID := createShopVehicle(t, testDB, shopID, owner, "A3")
	repo := pmcs_sbs_progress.NewRepository(testDB)

	_, err := repo.UpsertFault(member, sampleFault(vehicleID))
	require.NoError(t, err)

	list, err := repo.ListFaults(member, vehicleID, "pmcs_sbs/hmmwv/file.json")
	require.NoError(t, err)
	require.Len(t, list, 1)
}

func TestRepositoryFaultsAreScopedByGuideManual(t *testing.T) {
	clearPmcsSbsTables(t, testDB)
	user := testUser("pmcs-fault-guide-scope")
	ensureUser(t, testDB, user)
	shopID := createShopWithMember(t, testDB, user, "member")
	vehicleID := createShopVehicle(t, testDB, shopID, user, "A7")
	repo := pmcs_sbs_progress.NewRepository(testDB)

	firstManualFault := sampleFault(vehicleID)
	firstManualFault.GuideManual = "pmcs_sbs/hmmwv/first.json"
	firstManualFault.FaultText = "first manual leak"
	_, err := repo.UpsertFault(user, firstManualFault)
	require.NoError(t, err)

	secondManualFault := sampleFault(vehicleID)
	secondManualFault.GuideManual = "pmcs_sbs/hmmwv/second.json"
	secondManualFault.FaultText = "second manual leak"
	_, err = repo.UpsertFault(user, secondManualFault)
	require.NoError(t, err)

	firstList, err := repo.ListFaults(user, vehicleID, "pmcs_sbs/hmmwv/first.json")
	require.NoError(t, err)
	require.Len(t, firstList, 1)
	require.Equal(t, "first manual leak", firstList[0].FaultText)

	secondList, err := repo.ListFaults(user, vehicleID, "pmcs_sbs/hmmwv/second.json")
	require.NoError(t, err)
	require.Len(t, secondList, 1)
	require.Equal(t, "second manual leak", secondList[0].FaultText)

	err = repo.DeleteFault(user, pmcs_sbs_progress.FaultKey{
		EquipmentID: vehicleID,
		GuideManual: "pmcs_sbs/hmmwv/first.json",
		SectionID:   "before",
		ItemIndex:   0,
	})
	require.NoError(t, err)

	firstList, err = repo.ListFaults(user, vehicleID, "pmcs_sbs/hmmwv/first.json")
	require.NoError(t, err)
	require.Empty(t, firstList)

	secondList, err = repo.ListFaults(user, vehicleID, "pmcs_sbs/hmmwv/second.json")
	require.NoError(t, err)
	require.Len(t, secondList, 1)
	require.Equal(t, "second manual leak", secondList[0].FaultText)
}

func TestRepositoryFaultUpsertPreservesCreatedAt(t *testing.T) {
	clearPmcsSbsTables(t, testDB)
	user := testUser("pmcs-fault-update")
	ensureUser(t, testDB, user)
	shopID := createShopWithMember(t, testDB, user, "member")
	vehicleID := createShopVehicle(t, testDB, shopID, user, "A4")
	repo := pmcs_sbs_progress.NewRepository(testDB)

	first, err := repo.UpsertFault(user, sampleFault(vehicleID))
	require.NoError(t, err)
	time.Sleep(time.Millisecond)

	updatedFault := sampleFault(vehicleID)
	updatedFault.FaultText = "updated leak"
	updatedFault.CorrectiveAction = "tightened"
	second, err := repo.UpsertFault(user, updatedFault)
	require.NoError(t, err)

	require.Equal(t, first.CreatedAt, second.CreatedAt)
	require.True(t, second.UpdatedAt.After(first.UpdatedAt))
	require.Equal(t, "updated leak", second.FaultText)
	require.Equal(t, "tightened", second.CorrectiveAction)
}

func TestRepositoryDeleteFaultIsIdempotentForAccessibleVehicle(t *testing.T) {
	clearPmcsSbsTables(t, testDB)
	user := testUser("pmcs-fault-delete-missing")
	ensureUser(t, testDB, user)
	shopID := createShopWithMember(t, testDB, user, "member")
	vehicleID := createShopVehicle(t, testDB, shopID, user, "A5")
	repo := pmcs_sbs_progress.NewRepository(testDB)

	err := repo.DeleteFault(user, pmcs_sbs_progress.FaultKey{EquipmentID: vehicleID, GuideManual: "pmcs_sbs/hmmwv/file.json", SectionID: "before", ItemIndex: 0})

	require.NoError(t, err)
}

func TestRepositoryVehicleDeleteCascadesFaults(t *testing.T) {
	clearPmcsSbsTables(t, testDB)
	user := testUser("pmcs-fault-cascade")
	ensureUser(t, testDB, user)
	shopID := createShopWithMember(t, testDB, user, "member")
	vehicleID := createShopVehicle(t, testDB, shopID, user, "A6")
	repo := pmcs_sbs_progress.NewRepository(testDB)
	_, err := repo.UpsertFault(user, sampleFault(vehicleID))
	require.NoError(t, err)

	_, err = testDB.Exec(`DELETE FROM shop_vehicle WHERE id=$1`, vehicleID)
	require.NoError(t, err)

	var count int
	err = testDB.QueryRow(`SELECT COUNT(*) FROM pmcs_sbs_faults WHERE equipment_id=$1`, vehicleID).Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 0, count)
}

func TestRepositoryDeleteFaultsDeletesMultipleAndReportsCount(t *testing.T) {
	clearPmcsSbsTables(t, testDB)
	user := testUser("pmcs-fault-bulk-delete")
	ensureUser(t, testDB, user)
	shopID := createShopWithMember(t, testDB, user, "member")
	vehicleID := createShopVehicle(t, testDB, shopID, user, "A8")
	repo := pmcs_sbs_progress.NewRepository(testDB)

	first := sampleFault(vehicleID)
	first.SectionID = "before"
	first.ItemIndex = 0
	first.ItemNo = "1"
	_, err := repo.UpsertFault(user, first)
	require.NoError(t, err)

	second := sampleFault(vehicleID)
	second.SectionID = "during"
	second.ItemIndex = 3
	second.ItemNo = "4"
	second.FaultText = "during leak"
	_, err = repo.UpsertFault(user, second)
	require.NoError(t, err)

	third := sampleFault(vehicleID)
	third.SectionID = "after"
	third.ItemIndex = 1
	third.ItemNo = "2"
	third.FaultText = "after leak"
	_, err = repo.UpsertFault(user, third)
	require.NoError(t, err)

	deletedCount, err := repo.DeleteFaults(user, vehicleID, "pmcs_sbs/hmmwv/file.json", []pmcs_sbs_progress.FaultKey{
		{EquipmentID: vehicleID, GuideManual: "pmcs_sbs/hmmwv/file.json", SectionID: "before", ItemIndex: 0},
		{EquipmentID: vehicleID, GuideManual: "pmcs_sbs/hmmwv/file.json", SectionID: "during", ItemIndex: 3},
	})
	require.NoError(t, err)
	require.Equal(t, int64(2), deletedCount)

	list, err := repo.ListFaults(user, vehicleID, "pmcs_sbs/hmmwv/file.json")
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, "after", list[0].SectionID)
}

func TestRepositoryDeleteFaultsIsIdempotentAndReportsExistingRowsOnly(t *testing.T) {
	clearPmcsSbsTables(t, testDB)
	user := testUser("pmcs-fault-bulk-idem")
	ensureUser(t, testDB, user)
	shopID := createShopWithMember(t, testDB, user, "member")
	vehicleID := createShopVehicle(t, testDB, shopID, user, "A9")
	repo := pmcs_sbs_progress.NewRepository(testDB)

	_, err := repo.UpsertFault(user, sampleFault(vehicleID))
	require.NoError(t, err)

	deletedCount, err := repo.DeleteFaults(user, vehicleID, "pmcs_sbs/hmmwv/file.json", []pmcs_sbs_progress.FaultKey{
		{EquipmentID: vehicleID, GuideManual: "pmcs_sbs/hmmwv/file.json", SectionID: "before", ItemIndex: 0},
		{EquipmentID: vehicleID, GuideManual: "pmcs_sbs/hmmwv/file.json", SectionID: "missing", ItemIndex: 99},
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), deletedCount)

	deletedCount, err = repo.DeleteFaults(user, vehicleID, "pmcs_sbs/hmmwv/file.json", []pmcs_sbs_progress.FaultKey{
		{EquipmentID: vehicleID, GuideManual: "pmcs_sbs/hmmwv/file.json", SectionID: "before", ItemIndex: 0},
		{EquipmentID: vehicleID, GuideManual: "pmcs_sbs/hmmwv/file.json", SectionID: "missing", ItemIndex: 99},
	})
	require.NoError(t, err)
	require.Equal(t, int64(0), deletedCount)
}

func TestRepositoryDeleteFaultsPreservesOtherManualsAndVehicles(t *testing.T) {
	clearPmcsSbsTables(t, testDB)
	user := testUser("pmcs-fault-bulk-scope")
	ensureUser(t, testDB, user)
	shopID := createShopWithMember(t, testDB, user, "member")
	vehicleID := createShopVehicle(t, testDB, shopID, user, "A10")
	otherVehicleID := createShopVehicle(t, testDB, shopID, user, "A11")
	repo := pmcs_sbs_progress.NewRepository(testDB)

	firstManualFault := sampleFault(vehicleID)
	firstManualFault.GuideManual = "pmcs_sbs/hmmwv/first.json"
	_, err := repo.UpsertFault(user, firstManualFault)
	require.NoError(t, err)

	secondManualFault := sampleFault(vehicleID)
	secondManualFault.GuideManual = "pmcs_sbs/hmmwv/second.json"
	secondManualFault.FaultText = "second manual leak"
	_, err = repo.UpsertFault(user, secondManualFault)
	require.NoError(t, err)

	otherVehicleFault := sampleFault(otherVehicleID)
	otherVehicleFault.GuideManual = "pmcs_sbs/hmmwv/first.json"
	otherVehicleFault.FaultText = "other vehicle leak"
	_, err = repo.UpsertFault(user, otherVehicleFault)
	require.NoError(t, err)

	deletedCount, err := repo.DeleteFaults(user, vehicleID, "pmcs_sbs/hmmwv/first.json", []pmcs_sbs_progress.FaultKey{
		{EquipmentID: vehicleID, GuideManual: "pmcs_sbs/hmmwv/first.json", SectionID: "before", ItemIndex: 0},
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), deletedCount)

	firstList, err := repo.ListFaults(user, vehicleID, "pmcs_sbs/hmmwv/first.json")
	require.NoError(t, err)
	require.Empty(t, firstList)

	secondList, err := repo.ListFaults(user, vehicleID, "pmcs_sbs/hmmwv/second.json")
	require.NoError(t, err)
	require.Len(t, secondList, 1)
	require.Equal(t, "second manual leak", secondList[0].FaultText)

	otherVehicleList, err := repo.ListFaults(user, otherVehicleID, "pmcs_sbs/hmmwv/first.json")
	require.NoError(t, err)
	require.Len(t, otherVehicleList, 1)
	require.Equal(t, "other vehicle leak", otherVehicleList[0].FaultText)
}

func TestRepositoryDeleteFaultsDeniesNonMemberAndMissingVehicle(t *testing.T) {
	clearPmcsSbsTables(t, testDB)
	owner := testUser("pmcs-fault-bulk-owner")
	other := testUser("pmcs-fault-bulk-other")
	ensureUser(t, testDB, owner)
	ensureUser(t, testDB, other)
	shopID := createShopWithMember(t, testDB, owner, "admin")
	vehicleID := createShopVehicle(t, testDB, shopID, owner, "A12")
	repo := pmcs_sbs_progress.NewRepository(testDB)
	keys := []pmcs_sbs_progress.FaultKey{{EquipmentID: vehicleID, GuideManual: "pmcs_sbs/hmmwv/file.json", SectionID: "before", ItemIndex: 0}}

	_, err := repo.DeleteFaults(other, vehicleID, "pmcs_sbs/hmmwv/file.json", keys)
	require.ErrorIs(t, err, pmcs_sbs_progress.ErrNotFound)

	_, err = repo.DeleteFaults(owner, "missing-vehicle", "pmcs_sbs/hmmwv/file.json", []pmcs_sbs_progress.FaultKey{{EquipmentID: "missing-vehicle", GuideManual: "pmcs_sbs/hmmwv/file.json", SectionID: "before", ItemIndex: 0}})
	require.ErrorIs(t, err, pmcs_sbs_progress.ErrNotFound)
}
