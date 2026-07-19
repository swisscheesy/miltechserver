package pmcs_sbs_progress_test

import (
	"testing"
	"time"

	"miltechserver/api/pmcs_sbs_progress"

	"github.com/stretchr/testify/require"
)

func TestRepositoryEnsureInspectionCreatesRecord(t *testing.T) {
	clearPmcsSbsTables(t, testDB)
	user := testUser("pmcs-insp-create")
	ensureUser(t, testDB, user)
	shopID := createShopWithMember(t, testDB, user, "member")
	vehicleID := createShopVehicle(t, testDB, shopID, user, "B1")
	repo := pmcs_sbs_progress.NewRepository(testDB)

	inspection := sampleInspection(vehicleID, user.UserID)
	saved, err := repo.EnsureInspection(user, inspection)

	require.NoError(t, err)
	require.Equal(t, inspection.ID, saved.ID)
	require.Equal(t, vehicleID, saved.EquipmentID)
	require.Equal(t, "pmcs_sbs/hmmwv/file.json", saved.GuideManual)
	require.NotNil(t, saved.PerformedBy)
	require.Equal(t, user.UserID, *saved.PerformedBy)
}

func TestRepositoryEnsureInspectionIsIdempotentAndUpdatesPerformedDate(t *testing.T) {
	clearPmcsSbsTables(t, testDB)
	user := testUser("pmcs-insp-idem")
	ensureUser(t, testDB, user)
	shopID := createShopWithMember(t, testDB, user, "member")
	vehicleID := createShopVehicle(t, testDB, shopID, user, "B2")
	repo := pmcs_sbs_progress.NewRepository(testDB)

	inspection := sampleInspection(vehicleID, user.UserID)
	first, err := repo.EnsureInspection(user, inspection)
	require.NoError(t, err)

	corrected := inspection
	corrected.PerformedDate = first.PerformedDate.Add(-24 * time.Hour)
	second, err := repo.EnsureInspection(user, corrected)

	require.NoError(t, err)
	require.Equal(t, first.ID, second.ID)
	require.True(t, second.PerformedDate.Before(first.PerformedDate))
}

func TestRepositoryEnsureInspectionRejectsGuideManualMismatch(t *testing.T) {
	clearPmcsSbsTables(t, testDB)
	user := testUser("pmcs-insp-conflict")
	ensureUser(t, testDB, user)
	shopID := createShopWithMember(t, testDB, user, "member")
	vehicleID := createShopVehicle(t, testDB, shopID, user, "B3")
	repo := pmcs_sbs_progress.NewRepository(testDB)

	inspection := sampleInspection(vehicleID, user.UserID)
	_, err := repo.EnsureInspection(user, inspection)
	require.NoError(t, err)

	mismatched := inspection
	mismatched.GuideManual = "pmcs_sbs/hmmwv/other.json"
	_, err = repo.EnsureInspection(user, mismatched)

	require.ErrorIs(t, err, pmcs_sbs_progress.ErrInspectionConflict)
}

func TestRepositoryEnsureInspectionDeniesNonMember(t *testing.T) {
	clearPmcsSbsTables(t, testDB)
	owner := testUser("pmcs-insp-owner")
	other := testUser("pmcs-insp-other")
	ensureUser(t, testDB, owner)
	ensureUser(t, testDB, other)
	shopID := createShopWithMember(t, testDB, owner, "admin")
	vehicleID := createShopVehicle(t, testDB, shopID, owner, "B4")
	repo := pmcs_sbs_progress.NewRepository(testDB)

	_, err := repo.EnsureInspection(other, sampleInspection(vehicleID, other.UserID))

	require.ErrorIs(t, err, pmcs_sbs_progress.ErrNotFound)
}

func TestRepositoryUpsertFaultCreatesInspectionImplicitly(t *testing.T) {
	clearPmcsSbsTables(t, testDB)
	user := testUser("pmcs-fault-implicit")
	ensureUser(t, testDB, user)
	shopID := createShopWithMember(t, testDB, user, "member")
	vehicleID := createShopVehicle(t, testDB, shopID, user, "B5")
	repo := pmcs_sbs_progress.NewRepository(testDB)

	inspection := sampleInspection(vehicleID, user.UserID)
	fault := sampleFault(inspection.ID)

	saved, err := repo.UpsertFault(user, inspection, fault)
	require.NoError(t, err)
	require.Equal(t, inspection.ID, saved.PmcsID)

	fetched, faults, err := repo.GetInspection(user, vehicleID, inspection.ID)
	require.NoError(t, err)
	require.Equal(t, inspection.GuideManual, fetched.GuideManual)
	require.Len(t, faults, 1)
	require.Equal(t, "leak", faults[0].FaultText)
}

func TestRepositoryUpsertFaultReusesExistingInspectionForSamePmcsID(t *testing.T) {
	clearPmcsSbsTables(t, testDB)
	user := testUser("pmcs-fault-reuse")
	ensureUser(t, testDB, user)
	shopID := createShopWithMember(t, testDB, user, "member")
	vehicleID := createShopVehicle(t, testDB, shopID, user, "B6")
	repo := pmcs_sbs_progress.NewRepository(testDB)

	inspection := sampleInspection(vehicleID, user.UserID)
	first := sampleFault(inspection.ID)
	_, err := repo.UpsertFault(user, inspection, first)
	require.NoError(t, err)

	second := sampleFault(inspection.ID)
	second.SectionID = "during"
	second.ItemIndex = 1
	second.FaultText = "second fault"
	_, err = repo.UpsertFault(user, inspection, second)
	require.NoError(t, err)

	_, faults, err := repo.GetInspection(user, vehicleID, inspection.ID)
	require.NoError(t, err)
	require.Len(t, faults, 2)
}

func TestRepositoryUpsertFaultRejectsGuideManualMismatch(t *testing.T) {
	clearPmcsSbsTables(t, testDB)
	user := testUser("pmcs-fault-conflict")
	ensureUser(t, testDB, user)
	shopID := createShopWithMember(t, testDB, user, "member")
	vehicleID := createShopVehicle(t, testDB, shopID, user, "B7")
	repo := pmcs_sbs_progress.NewRepository(testDB)

	inspection := sampleInspection(vehicleID, user.UserID)
	_, err := repo.UpsertFault(user, inspection, sampleFault(inspection.ID))
	require.NoError(t, err)

	mismatched := inspection
	mismatched.GuideManual = "pmcs_sbs/hmmwv/other.json"
	_, err = repo.UpsertFault(user, mismatched, sampleFault(inspection.ID))

	require.ErrorIs(t, err, pmcs_sbs_progress.ErrInspectionConflict)
}

func TestRepositoryGetInspectionReturnsFaultsOrderedBySectionAndItem(t *testing.T) {
	clearPmcsSbsTables(t, testDB)
	user := testUser("pmcs-get-order")
	ensureUser(t, testDB, user)
	shopID := createShopWithMember(t, testDB, user, "member")
	vehicleID := createShopVehicle(t, testDB, shopID, user, "B8")
	repo := pmcs_sbs_progress.NewRepository(testDB)

	inspection := sampleInspection(vehicleID, user.UserID)
	late := sampleFault(inspection.ID)
	late.SectionID = "during"
	late.ItemIndex = 2
	_, err := repo.UpsertFault(user, inspection, late)
	require.NoError(t, err)

	early := sampleFault(inspection.ID)
	early.SectionID = "before"
	early.ItemIndex = 0
	_, err = repo.UpsertFault(user, inspection, early)
	require.NoError(t, err)

	_, faults, err := repo.GetInspection(user, vehicleID, inspection.ID)
	require.NoError(t, err)
	require.Len(t, faults, 2)
	require.Equal(t, "before", faults[0].SectionID)
	require.Equal(t, "during", faults[1].SectionID)
}

func TestRepositoryGetInspectionReturnsCleanInspectionWithEmptyFaults(t *testing.T) {
	clearPmcsSbsTables(t, testDB)
	user := testUser("pmcs-get-clean")
	ensureUser(t, testDB, user)
	shopID := createShopWithMember(t, testDB, user, "member")
	vehicleID := createShopVehicle(t, testDB, shopID, user, "B9")
	repo := pmcs_sbs_progress.NewRepository(testDB)

	inspection := sampleInspection(vehicleID, user.UserID)
	_, err := repo.EnsureInspection(user, inspection)
	require.NoError(t, err)

	fetched, faults, err := repo.GetInspection(user, vehicleID, inspection.ID)
	require.NoError(t, err)
	require.Equal(t, inspection.ID, fetched.ID)
	require.Empty(t, faults)
}

func TestRepositoryGetInspectionRejectsCrossVehiclePmcsID(t *testing.T) {
	clearPmcsSbsTables(t, testDB)
	user := testUser("pmcs-get-crossveh")
	ensureUser(t, testDB, user)
	shopID := createShopWithMember(t, testDB, user, "member")
	vehicleID := createShopVehicle(t, testDB, shopID, user, "B10")
	otherVehicleID := createShopVehicle(t, testDB, shopID, user, "B11")
	repo := pmcs_sbs_progress.NewRepository(testDB)

	inspection := sampleInspection(vehicleID, user.UserID)
	_, err := repo.EnsureInspection(user, inspection)
	require.NoError(t, err)

	_, _, err = repo.GetInspection(user, otherVehicleID, inspection.ID)

	require.ErrorIs(t, err, pmcs_sbs_progress.ErrInspectionNotFound)
}

func TestRepositoryGetInspectionIncludesPerformedByUsername(t *testing.T) {
	clearPmcsSbsTables(t, testDB)
	user := testUser("pmcs-get-username")
	user.Username = "jsmith"
	ensureUser(t, testDB, user)
	shopID := createShopWithMember(t, testDB, user, "member")
	vehicleID := createShopVehicle(t, testDB, shopID, user, "B17")
	repo := pmcs_sbs_progress.NewRepository(testDB)

	inspection := sampleInspection(vehicleID, user.UserID)
	_, err := repo.EnsureInspection(user, inspection)
	require.NoError(t, err)

	detail, _, err := repo.GetInspection(user, vehicleID, inspection.ID)
	require.NoError(t, err)
	require.NotNil(t, detail.PerformedBy)
	require.Equal(t, user.UserID, *detail.PerformedBy)
	require.NotNil(t, detail.PerformedByUsername)
	require.Equal(t, "jsmith", *detail.PerformedByUsername)
}

func TestRepositoryGetInspectionReturnsNilUsernameWhenPerformerDeleted(t *testing.T) {
	clearPmcsSbsTables(t, testDB)
	performer := testUser("pmcs-del-performer")
	viewer := testUser("pmcs-get-deleted-viewer")
	ensureUser(t, testDB, performer)
	ensureUser(t, testDB, viewer)
	// viewer, not performer, creates the shop and the vehicle: shops.created_by
	// is a NO ACTION (restrict) foreign key to users(uid), and
	// shop_vehicle.creator_id is NOT NULL despite its ON DELETE SET NULL
	// action, so the row we're about to delete below can't be the creator of
	// either. performer only appears as a shop_members row (CASCADE) and as
	// the inspection's performed_by (SET NULL).
	shopID := createShopWithMember(t, testDB, viewer, "member")
	addShopMember(t, testDB, shopID, performer, "member")
	vehicleID := createShopVehicle(t, testDB, shopID, viewer, "B18")
	repo := pmcs_sbs_progress.NewRepository(testDB)

	inspection := sampleInspection(vehicleID, performer.UserID)
	_, err := repo.EnsureInspection(performer, inspection)
	require.NoError(t, err)

	_, err = testDB.Exec(`DELETE FROM users WHERE uid=$1`, performer.UserID)
	require.NoError(t, err)

	detail, _, err := repo.GetInspection(viewer, vehicleID, inspection.ID)
	require.NoError(t, err)
	require.Nil(t, detail.PerformedBy)
	require.Nil(t, detail.PerformedByUsername)
}

func TestRepositoryListInspectionsIncludesPerformedByUsername(t *testing.T) {
	clearPmcsSbsTables(t, testDB)
	user := testUser("pmcs-list-username")
	user.Username = "jsmith"
	ensureUser(t, testDB, user)
	shopID := createShopWithMember(t, testDB, user, "member")
	vehicleID := createShopVehicle(t, testDB, shopID, user, "B19")
	repo := pmcs_sbs_progress.NewRepository(testDB)

	inspection := sampleInspection(vehicleID, user.UserID)
	_, err := repo.EnsureInspection(user, inspection)
	require.NoError(t, err)

	summaries, err := repo.ListInspections(user, vehicleID, "", 10, 0)
	require.NoError(t, err)
	require.Len(t, summaries, 1)
	require.NotNil(t, summaries[0].PerformedBy)
	require.Equal(t, user.UserID, *summaries[0].PerformedBy)
	require.NotNil(t, summaries[0].PerformedByUsername)
	require.Equal(t, "jsmith", *summaries[0].PerformedByUsername)
}

func TestRepositoryListInspectionsOrdersByPerformedDateDescWithFaultCounts(t *testing.T) {
	clearPmcsSbsTables(t, testDB)
	user := testUser("pmcs-list-order")
	ensureUser(t, testDB, user)
	shopID := createShopWithMember(t, testDB, user, "member")
	vehicleID := createShopVehicle(t, testDB, shopID, user, "B12")
	repo := pmcs_sbs_progress.NewRepository(testDB)

	older := sampleInspection(vehicleID, user.UserID)
	older.PerformedDate = time.Now().UTC().Add(-48 * time.Hour)
	_, err := repo.EnsureInspection(user, older)
	require.NoError(t, err)

	newer := sampleInspection(vehicleID, user.UserID)
	newer.PerformedDate = time.Now().UTC()
	_, err = repo.UpsertFault(user, newer, sampleFault(newer.ID))
	require.NoError(t, err)

	summaries, err := repo.ListInspections(user, vehicleID, "", 10, 0)
	require.NoError(t, err)
	require.Len(t, summaries, 2)
	require.Equal(t, newer.ID, summaries[0].ID)
	require.Equal(t, 1, summaries[0].FaultCount)
	require.Equal(t, older.ID, summaries[1].ID)
	require.Equal(t, 0, summaries[1].FaultCount)
}

func TestRepositoryListInspectionsFiltersByGuideManual(t *testing.T) {
	clearPmcsSbsTables(t, testDB)
	user := testUser("pmcs-list-filter")
	ensureUser(t, testDB, user)
	shopID := createShopWithMember(t, testDB, user, "member")
	vehicleID := createShopVehicle(t, testDB, shopID, user, "B13")
	repo := pmcs_sbs_progress.NewRepository(testDB)

	first := sampleInspection(vehicleID, user.UserID)
	first.GuideManual = "pmcs_sbs/hmmwv/first.json"
	_, err := repo.EnsureInspection(user, first)
	require.NoError(t, err)

	second := sampleInspection(vehicleID, user.UserID)
	second.GuideManual = "pmcs_sbs/hmmwv/second.json"
	_, err = repo.EnsureInspection(user, second)
	require.NoError(t, err)

	summaries, err := repo.ListInspections(user, vehicleID, "pmcs_sbs/hmmwv/first.json", 10, 0)
	require.NoError(t, err)
	require.Len(t, summaries, 1)
	require.Equal(t, first.ID, summaries[0].ID)
}

func TestRepositoryDeleteInspectionCascadesFaultsButNotSiblings(t *testing.T) {
	clearPmcsSbsTables(t, testDB)
	user := testUser("pmcs-delete-cascade")
	ensureUser(t, testDB, user)
	shopID := createShopWithMember(t, testDB, user, "member")
	vehicleID := createShopVehicle(t, testDB, shopID, user, "B14")
	repo := pmcs_sbs_progress.NewRepository(testDB)

	toDelete := sampleInspection(vehicleID, user.UserID)
	_, err := repo.UpsertFault(user, toDelete, sampleFault(toDelete.ID))
	require.NoError(t, err)

	sibling := sampleInspection(vehicleID, user.UserID)
	_, err = repo.UpsertFault(user, sibling, sampleFault(sibling.ID))
	require.NoError(t, err)

	err = repo.DeleteInspection(user, vehicleID, toDelete.ID)
	require.NoError(t, err)

	var faultCount int
	err = testDB.QueryRow(`SELECT COUNT(*) FROM pmcs_sbs_faults WHERE pmcs_id=$1`, toDelete.ID).Scan(&faultCount)
	require.NoError(t, err)
	require.Equal(t, 0, faultCount)

	_, siblingFaults, err := repo.GetInspection(user, vehicleID, sibling.ID)
	require.NoError(t, err)
	require.Len(t, siblingFaults, 1)
}

func TestRepositoryDeleteFaultAndBulkDeleteFaultsScopedToInspection(t *testing.T) {
	clearPmcsSbsTables(t, testDB)
	user := testUser("pmcs-delete-fault")
	ensureUser(t, testDB, user)
	shopID := createShopWithMember(t, testDB, user, "member")
	vehicleID := createShopVehicle(t, testDB, shopID, user, "B15")
	repo := pmcs_sbs_progress.NewRepository(testDB)

	inspection := sampleInspection(vehicleID, user.UserID)
	first := sampleFault(inspection.ID)
	first.SectionID = "before"
	first.ItemIndex = 0
	_, err := repo.UpsertFault(user, inspection, first)
	require.NoError(t, err)

	second := sampleFault(inspection.ID)
	second.SectionID = "during"
	second.ItemIndex = 1
	_, err = repo.UpsertFault(user, inspection, second)
	require.NoError(t, err)

	err = repo.DeleteFault(user, vehicleID, pmcs_sbs_progress.FaultKey{PmcsID: inspection.ID, SectionID: "before", ItemIndex: 0})
	require.NoError(t, err)

	deletedCount, err := repo.DeleteFaults(user, vehicleID, inspection.ID, []pmcs_sbs_progress.FaultKey{
		{PmcsID: inspection.ID, SectionID: "during", ItemIndex: 1},
		{PmcsID: inspection.ID, SectionID: "missing", ItemIndex: 99},
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), deletedCount)

	_, faults, err := repo.GetInspection(user, vehicleID, inspection.ID)
	require.NoError(t, err)
	require.Empty(t, faults)
}

func TestRepositoryVehicleDeleteCascadesInspectionsAndFaults(t *testing.T) {
	clearPmcsSbsTables(t, testDB)
	user := testUser("pmcs-vehicle-cascade")
	ensureUser(t, testDB, user)
	shopID := createShopWithMember(t, testDB, user, "member")
	vehicleID := createShopVehicle(t, testDB, shopID, user, "B16")
	repo := pmcs_sbs_progress.NewRepository(testDB)

	inspection := sampleInspection(vehicleID, user.UserID)
	_, err := repo.UpsertFault(user, inspection, sampleFault(inspection.ID))
	require.NoError(t, err)

	_, err = testDB.Exec(`DELETE FROM shop_vehicle WHERE id=$1`, vehicleID)
	require.NoError(t, err)

	var inspectionCount, faultCount int
	err = testDB.QueryRow(`SELECT COUNT(*) FROM pmcs_sbs_inspections WHERE equipment_id=$1`, vehicleID).Scan(&inspectionCount)
	require.NoError(t, err)
	require.Equal(t, 0, inspectionCount)

	err = testDB.QueryRow(`SELECT COUNT(*) FROM pmcs_sbs_faults WHERE pmcs_id=$1`, inspection.ID).Scan(&faultCount)
	require.NoError(t, err)
	require.Equal(t, 0, faultCount)
}
