package pmcs_sbs_progress_test

import (
	"testing"
	"time"

	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/pmcs_sbs_progress"

	"github.com/google/uuid"
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
	require.NotNil(t, saved.GuideManual)
	require.Equal(t, "pmcs_sbs/hmmwv/file.json", *saved.GuideManual)
	require.NotNil(t, saved.PerformedBy)
	require.Equal(t, user.UserID, *saved.PerformedBy)
}

func TestInspectionSourceConstraint(t *testing.T) {
	clearPmcsSbsTables(t, testDB)
	user := testUser("pmcs-source-constraint")
	ensureUser(t, testDB, user)
	shopID := createShopWithMember(t, testDB, user, "member")
	vehicleID := createShopVehicle(t, testDB, shopID, user, "S1")
	performedDate := time.Now().UTC()
	_, err := testDB.Exec(
		`INSERT INTO shop_vehicle (
			id, creator_id, niin, admin, model, serial, uoc, mileage, hours, comment,
			save_time, last_updated, shop_id
		) VALUES ('', $1, '', 'BLANK', 'M1152A1', 'BLANK-EQUIPMENT', 'UNK', 0, 0, '', $2, $2, $3)`,
		user.UserID, performedDate, shopID,
	)
	require.NoError(t, err)

	_, err = testDB.Exec(
		`INSERT INTO pmcs_sbs_inspections
		  (id, equipment_id, source_type, guide_manual, performed_date, performed_by)
		 VALUES ($1, $2, 'guide', 'pmcs_sbs/hmmwv/file.json', $3, $4)`,
		uuid.New(), vehicleID, performedDate, user.UserID,
	)
	require.NoError(t, err)

	customChecklistID := uuid.New()
	customRevisionID := uuid.New()
	_, err = testDB.Exec(
		`INSERT INTO pmcs_sbs_inspections
		  (id, equipment_id, source_type, guide_manual,
		   custom_checklist_id, custom_revision_id,
		   custom_revision_number, custom_checklist_name,
		   performed_date, performed_by)
		 VALUES ($1, $2, 'custom', NULL, $3, $4, 0, 'Device Checklist', $5, $6)`,
		uuid.New(), vehicleID, customChecklistID, customRevisionID, performedDate, user.UserID,
	)
	require.NoError(t, err)

	invalidRows := []struct {
		name  string
		query string
		args  []any
	}{
		{
			name: "inspection equipment cannot be blank",
			query: `INSERT INTO pmcs_sbs_inspections
			  (id, equipment_id, source_type, guide_manual, performed_date, performed_by)
			 VALUES ($1, '', 'guide', 'pmcs_sbs/hmmwv/file.json', $2, $3)`,
			args: []any{uuid.New(), performedDate, user.UserID},
		},
		{
			name: "custom source cannot include guide manual",
			query: `INSERT INTO pmcs_sbs_inspections
			  (id, equipment_id, source_type, guide_manual,
			   custom_checklist_id, custom_revision_id,
			   custom_revision_number, custom_checklist_name,
			   performed_date, performed_by)
			 VALUES ($1, $2, 'custom', 'pmcs_sbs/hmmwv/file.json', $3, $4, 0, 'Device Checklist', $5, $6)`,
			args: []any{uuid.New(), vehicleID, customChecklistID, customRevisionID, performedDate, user.UserID},
		},
		{
			name: "guide source cannot include custom provenance",
			query: `INSERT INTO pmcs_sbs_inspections
			  (id, equipment_id, source_type, guide_manual, custom_checklist_id,
			   performed_date, performed_by)
			 VALUES ($1, $2, 'guide', 'pmcs_sbs/hmmwv/file.json', $3, $4, $5)`,
			args: []any{uuid.New(), vehicleID, customChecklistID, performedDate, user.UserID},
		},
		{
			name: "custom source requires revision number",
			query: `INSERT INTO pmcs_sbs_inspections
			  (id, equipment_id, source_type, guide_manual,
			   custom_checklist_id, custom_revision_id, custom_checklist_name,
			   performed_date, performed_by)
			 VALUES ($1, $2, 'custom', NULL, $3, $4, 'Device Checklist', $5, $6)`,
			args: []any{uuid.New(), vehicleID, customChecklistID, customRevisionID, performedDate, user.UserID},
		},
		{
			name: "custom source requires checklist name",
			query: `INSERT INTO pmcs_sbs_inspections
			  (id, equipment_id, source_type, guide_manual,
			   custom_checklist_id, custom_revision_id, custom_revision_number,
			   performed_date, performed_by)
			 VALUES ($1, $2, 'custom', NULL, $3, $4, 0, $5, $6)`,
			args: []any{uuid.New(), vehicleID, customChecklistID, customRevisionID, performedDate, user.UserID},
		},
	}

	for _, invalidRow := range invalidRows {
		t.Run(invalidRow.name, func(t *testing.T) {
			_, err := testDB.Exec(invalidRow.query, invalidRow.args...)
			require.Error(t, err)
		})
	}
}

func TestRepositoryCustomListAndDetailReturnNullableSourceProvenanceCountsAndOrder(t *testing.T) {
	clearPmcsSbsTables(t, testDB)
	user := testUser("pmcs-custom-read")
	ensureUser(t, testDB, user)
	shopID := createShopWithMember(t, testDB, user, "member")
	vehicleID := createShopVehicle(t, testDB, shopID, user, "C1")
	repo := pmcs_sbs_progress.NewRepository(testDB)

	guide := sampleInspection(vehicleID, user.UserID)
	guide.PerformedDate = time.Now().UTC().Add(-24 * time.Hour)
	_, err := repo.EnsureInspection(user, guide)
	require.NoError(t, err)

	custom := sampleCustomInspection(vehicleID, user.UserID)
	custom.PerformedDate = time.Now().UTC()
	_, err = repo.UpsertFault(user, custom, sampleFault(custom.ID))
	require.NoError(t, err)
	_, err = repo.CreateComment(user, vehicleID, custom.ID, "custom comment")
	require.NoError(t, err)

	detail, faults, comments, err := repo.GetInspection(user, vehicleID, custom.ID)
	require.NoError(t, err)
	require.Equal(t, "custom", detail.SourceType)
	require.Nil(t, detail.GuideManual)
	require.Equal(t, custom.CustomChecklistID, detail.CustomChecklistID)
	require.Equal(t, custom.CustomRevisionID, detail.CustomRevisionID)
	require.Equal(t, custom.CustomRevisionNumber, detail.CustomRevisionNumber)
	require.Equal(t, custom.CustomChecklistName, detail.CustomChecklistName)
	require.Len(t, faults, 1)
	require.Len(t, comments, 1)

	summaries, err := repo.ListInspections(user, vehicleID, "", 10, 0)
	require.NoError(t, err)
	require.Len(t, summaries, 2)
	require.Equal(t, custom.ID, summaries[0].ID)
	require.Equal(t, 1, summaries[0].FaultCount)
	require.Equal(t, 1, summaries[0].CommentCount)
	require.Equal(t, guide.ID, summaries[1].ID)

	require.Equal(t, "custom", summaries[0].SourceType)
	require.Nil(t, summaries[0].GuideManual)
	require.Equal(t, custom.CustomChecklistID, summaries[0].CustomChecklistID)
	require.Equal(t, custom.CustomRevisionID, summaries[0].CustomRevisionID)
	require.Equal(t, custom.CustomRevisionNumber, summaries[0].CustomRevisionNumber)
	require.Equal(t, custom.CustomChecklistName, summaries[0].CustomChecklistName)

	guideSummaries, err := repo.ListInspections(user, vehicleID, *guide.GuideManual, 10, 0)
	require.NoError(t, err)
	require.Len(t, guideSummaries, 1)
	require.Equal(t, guide.ID, guideSummaries[0].ID)
}

func TestRepositoryCustomUpsertFaultCreatesInspectionImplicitly(t *testing.T) {
	clearPmcsSbsTables(t, testDB)
	user := testUser("cst-fault-implicit")
	ensureUser(t, testDB, user)
	shopID := createShopWithMember(t, testDB, user, "member")
	vehicleID := createShopVehicle(t, testDB, shopID, user, "C2")
	repo := pmcs_sbs_progress.NewRepository(testDB)

	inspection := sampleCustomInspection(vehicleID, user.UserID)
	saved, err := repo.UpsertFault(user, inspection, sampleFault(inspection.ID))

	require.NoError(t, err)
	require.Equal(t, inspection.ID, saved.PmcsID)
	detail, faults, _, err := repo.GetInspection(user, vehicleID, inspection.ID)
	require.NoError(t, err)
	require.Equal(t, "custom", detail.SourceType)
	require.Nil(t, detail.GuideManual)
	require.Equal(t, inspection.CustomChecklistID, detail.CustomChecklistID)
	require.Equal(t, inspection.CustomRevisionID, detail.CustomRevisionID)
	require.Equal(t, inspection.CustomRevisionNumber, detail.CustomRevisionNumber)
	require.Equal(t, inspection.CustomChecklistName, detail.CustomChecklistName)
	require.Len(t, faults, 1)
}

func TestRepositoryCustomEnsureInspectionCreatesCleanInspection(t *testing.T) {
	clearPmcsSbsTables(t, testDB)
	user := testUser("pmcs-custom-clean")
	ensureUser(t, testDB, user)
	shopID := createShopWithMember(t, testDB, user, "member")
	vehicleID := createShopVehicle(t, testDB, shopID, user, "C3")
	repo := pmcs_sbs_progress.NewRepository(testDB)

	inspection := sampleCustomInspection(vehicleID, user.UserID)
	saved, err := repo.EnsureInspection(user, inspection)

	require.NoError(t, err)
	require.Equal(t, inspection.ID, saved.ID)
	require.Equal(t, "custom", saved.SourceType)
	require.Nil(t, saved.GuideManual)
	_, faults, _, err := repo.GetInspection(user, vehicleID, inspection.ID)
	require.NoError(t, err)
	require.Empty(t, faults)
}

func TestRepositoryCustomEnsureInspectionRetryUpdatesOnlyMutableFields(t *testing.T) {
	clearPmcsSbsTables(t, testDB)
	creator := testUser("pmcs-custom-retry-creator")
	retryingMember := testUser("pmcs-custom-retry-member")
	ensureUser(t, testDB, creator)
	ensureUser(t, testDB, retryingMember)
	shopID := createShopWithMember(t, testDB, creator, "admin")
	addShopMember(t, testDB, shopID, retryingMember, "member")
	vehicleID := createShopVehicle(t, testDB, shopID, creator, "C4")
	repo := pmcs_sbs_progress.NewRepository(testDB)

	inspection := sampleCustomInspection(vehicleID, creator.UserID)
	firstNotes := "first notes"
	inspection.Notes = &firstNotes
	first, err := repo.EnsureInspection(creator, inspection)
	require.NoError(t, err)

	retry := inspection
	retryDate := inspection.PerformedDate.Add(-time.Hour)
	retry.PerformedDate = retryDate
	retryingMemberID := retryingMember.UserID
	retry.PerformedBy = &retryingMemberID
	retryNotes := "corrected notes"
	retry.Notes = &retryNotes
	second, err := repo.EnsureInspection(retryingMember, retry)

	require.NoError(t, err)
	require.Equal(t, first.ID, second.ID)
	require.True(t, retryDate.Equal(second.PerformedDate))
	require.Equal(t, &retryNotes, second.Notes)
	require.Equal(t, &creator.UserID, second.PerformedBy)
	require.Equal(t, inspection.CustomChecklistID, second.CustomChecklistID)

	var count int
	err = testDB.QueryRow(`SELECT COUNT(*) FROM pmcs_sbs_inspections WHERE id=$1`, inspection.ID).Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 1, count)
}

func TestRepositoryCustomEnsureInspectionRejectsSourceMutation(t *testing.T) {
	clearPmcsSbsTables(t, testDB)
	user := testUser("cst-source-conflict")
	ensureUser(t, testDB, user)
	shopID := createShopWithMember(t, testDB, user, "member")
	vehicleID := createShopVehicle(t, testDB, shopID, user, "C5")
	repo := pmcs_sbs_progress.NewRepository(testDB)

	inspection := sampleCustomInspection(vehicleID, user.UserID)
	_, err := repo.EnsureInspection(user, inspection)
	require.NoError(t, err)

	mutations := []struct {
		name   string
		mutate func(model model.PmcsSbsInspections) model.PmcsSbsInspections
	}{
		{
			name: "source type",
			mutate: func(model model.PmcsSbsInspections) model.PmcsSbsInspections {
				guideManual := "pmcs_sbs/hmmwv/file.json"
				model.SourceType = "guide"
				model.GuideManual = &guideManual
				model.CustomChecklistID = nil
				model.CustomRevisionID = nil
				model.CustomRevisionNumber = nil
				model.CustomChecklistName = nil
				return model
			},
		},
		{
			name: "checklist id",
			mutate: func(model model.PmcsSbsInspections) model.PmcsSbsInspections {
				value := uuid.New()
				model.CustomChecklistID = &value
				return model
			},
		},
		{
			name: "revision id",
			mutate: func(model model.PmcsSbsInspections) model.PmcsSbsInspections {
				value := uuid.New()
				model.CustomRevisionID = &value
				return model
			},
		},
		{
			name: "revision number",
			mutate: func(model model.PmcsSbsInspections) model.PmcsSbsInspections {
				value := *model.CustomRevisionNumber + 1
				model.CustomRevisionNumber = &value
				return model
			},
		},
		{
			name: "checklist name",
			mutate: func(model model.PmcsSbsInspections) model.PmcsSbsInspections {
				value := "Different Checklist"
				model.CustomChecklistName = &value
				return model
			},
		},
	}

	for _, mutation := range mutations {
		t.Run(mutation.name, func(t *testing.T) {
			_, err := repo.EnsureInspection(user, mutation.mutate(inspection))
			require.ErrorIs(t, err, pmcs_sbs_progress.ErrInspectionConflict)
		})
	}
}

func TestRepositoryCustomEnsureInspectionRejectsEquipmentMutation(t *testing.T) {
	clearPmcsSbsTables(t, testDB)
	user := testUser("cst-equipment-conflict")
	ensureUser(t, testDB, user)
	shopID := createShopWithMember(t, testDB, user, "member")
	vehicleID := createShopVehicle(t, testDB, shopID, user, "C6")
	otherVehicleID := createShopVehicle(t, testDB, shopID, user, "C7")
	repo := pmcs_sbs_progress.NewRepository(testDB)

	inspection := sampleCustomInspection(vehicleID, user.UserID)
	_, err := repo.EnsureInspection(user, inspection)
	require.NoError(t, err)

	mutated := inspection
	mutated.EquipmentID = otherVehicleID
	_, err = repo.EnsureInspection(user, mutated)

	require.ErrorIs(t, err, pmcs_sbs_progress.ErrInspectionConflict)
}

func TestRepositoryCustomInspectionAllowsShopMemberAccess(t *testing.T) {
	clearPmcsSbsTables(t, testDB)
	owner := testUser("pmcs-custom-access-owner")
	member := testUser("pmcs-custom-access-member")
	ensureUser(t, testDB, owner)
	ensureUser(t, testDB, member)
	shopID := createShopWithMember(t, testDB, owner, "admin")
	addShopMember(t, testDB, shopID, member, "member")
	vehicleID := createShopVehicle(t, testDB, shopID, owner, "C8")
	repo := pmcs_sbs_progress.NewRepository(testDB)

	inspection := sampleCustomInspection(vehicleID, member.UserID)
	_, err := repo.EnsureInspection(member, inspection)
	require.NoError(t, err)

	detail, _, _, err := repo.GetInspection(owner, vehicleID, inspection.ID)
	require.NoError(t, err)
	require.Equal(t, inspection.ID, detail.ID)
}

func TestRepositoryCustomInspectionHidesVehicleFromNonmember(t *testing.T) {
	clearPmcsSbsTables(t, testDB)
	owner := testUser("pmcs-custom-hidden-owner")
	nonmember := testUser("cst-hidden-nonmember")
	ensureUser(t, testDB, owner)
	ensureUser(t, testDB, nonmember)
	shopID := createShopWithMember(t, testDB, owner, "admin")
	vehicleID := createShopVehicle(t, testDB, shopID, owner, "C9")
	repo := pmcs_sbs_progress.NewRepository(testDB)

	inspection := sampleCustomInspection(vehicleID, nonmember.UserID)
	_, err := repo.EnsureInspection(nonmember, inspection)
	require.ErrorIs(t, err, pmcs_sbs_progress.ErrNotFound)

	inspection.PerformedBy = &owner.UserID
	_, err = repo.EnsureInspection(owner, inspection)
	require.NoError(t, err)
	_, _, _, err = repo.GetInspection(nonmember, vehicleID, inspection.ID)
	require.ErrorIs(t, err, pmcs_sbs_progress.ErrNotFound)
}

func TestRepositoryUpsertFaultPersistsAndUpdatesSectionTitle(t *testing.T) {
	clearPmcsSbsTables(t, testDB)
	user := testUser("pmcs-custom-section-title")
	ensureUser(t, testDB, user)
	shopID := createShopWithMember(t, testDB, user, "member")
	vehicleID := createShopVehicle(t, testDB, shopID, user, "C10")
	repo := pmcs_sbs_progress.NewRepository(testDB)

	inspection := sampleCustomInspection(vehicleID, user.UserID)
	fault := sampleFault(inspection.ID)
	firstTitle := "Before Operation"
	fault.SectionTitle = &firstTitle
	first, err := repo.UpsertFault(user, inspection, fault)
	require.NoError(t, err)
	require.Equal(t, &firstTitle, first.SectionTitle)

	updatedTitle := "Before Checks"
	fault.SectionTitle = &updatedTitle
	second, err := repo.UpsertFault(user, inspection, fault)
	require.NoError(t, err)
	require.Equal(t, &updatedTitle, second.SectionTitle)

	_, faults, _, err := repo.GetInspection(user, vehicleID, inspection.ID)
	require.NoError(t, err)
	require.Len(t, faults, 1)
	require.Equal(t, &updatedTitle, faults[0].SectionTitle)
}

func TestRepositoryCustomVehicleDeleteCascadesInspectionFaultAndComment(t *testing.T) {
	clearPmcsSbsTables(t, testDB)
	user := testUser("pmcs-custom-cascade")
	ensureUser(t, testDB, user)
	shopID := createShopWithMember(t, testDB, user, "member")
	vehicleID := createShopVehicle(t, testDB, shopID, user, "C11")
	repo := pmcs_sbs_progress.NewRepository(testDB)

	inspection := sampleCustomInspection(vehicleID, user.UserID)
	_, err := repo.UpsertFault(user, inspection, sampleFault(inspection.ID))
	require.NoError(t, err)
	_, err = repo.CreateComment(user, vehicleID, inspection.ID, "cascade comment")
	require.NoError(t, err)

	_, err = testDB.Exec(`DELETE FROM shop_vehicle WHERE id=$1`, vehicleID)
	require.NoError(t, err)

	var inspectionCount, faultCount, commentCount int
	err = testDB.QueryRow(`SELECT COUNT(*) FROM pmcs_sbs_inspections WHERE id=$1`, inspection.ID).Scan(&inspectionCount)
	require.NoError(t, err)
	err = testDB.QueryRow(`SELECT COUNT(*) FROM pmcs_sbs_faults WHERE pmcs_id=$1`, inspection.ID).Scan(&faultCount)
	require.NoError(t, err)
	err = testDB.QueryRow(`SELECT COUNT(*) FROM pmcs_sbs_inspection_comments WHERE pmcs_id=$1`, inspection.ID).Scan(&commentCount)
	require.NoError(t, err)
	require.Zero(t, inspectionCount)
	require.Zero(t, faultCount)
	require.Zero(t, commentCount)
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
	mismatched.GuideManual = stringPointer("pmcs_sbs/hmmwv/other.json")
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

	fetched, faults, _, err := repo.GetInspection(user, vehicleID, inspection.ID)
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

	_, faults, _, err := repo.GetInspection(user, vehicleID, inspection.ID)
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
	mismatched.GuideManual = stringPointer("pmcs_sbs/hmmwv/other.json")
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

	_, faults, _, err := repo.GetInspection(user, vehicleID, inspection.ID)
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

	fetched, faults, _, err := repo.GetInspection(user, vehicleID, inspection.ID)
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

	_, _, _, err = repo.GetInspection(user, otherVehicleID, inspection.ID)

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

	detail, _, _, err := repo.GetInspection(user, vehicleID, inspection.ID)
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

	detail, _, _, err := repo.GetInspection(viewer, vehicleID, inspection.ID)
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
	first.GuideManual = stringPointer("pmcs_sbs/hmmwv/first.json")
	_, err := repo.EnsureInspection(user, first)
	require.NoError(t, err)

	second := sampleInspection(vehicleID, user.UserID)
	second.GuideManual = stringPointer("pmcs_sbs/hmmwv/second.json")
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

	_, siblingFaults, _, err := repo.GetInspection(user, vehicleID, sibling.ID)
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

	_, faults, _, err := repo.GetInspection(user, vehicleID, inspection.ID)
	require.NoError(t, err)
	require.Empty(t, faults)
}

func TestRepositoryEnsureInspectionPersistsAndClearsNotes(t *testing.T) {
	clearPmcsSbsTables(t, testDB)
	user := testUser("pmcs-notes-persist")
	ensureUser(t, testDB, user)
	shopID := createShopWithMember(t, testDB, user, "member")
	vehicleID := createShopVehicle(t, testDB, shopID, user, "N1")
	repo := pmcs_sbs_progress.NewRepository(testDB)

	inspection := sampleInspection(vehicleID, user.UserID)
	notes := "clean inspection, no issues found"
	inspection.Notes = &notes
	saved, err := repo.EnsureInspection(user, inspection)
	require.NoError(t, err)
	require.NotNil(t, saved.Notes)
	require.Equal(t, notes, *saved.Notes)

	cleared := inspection
	cleared.Notes = nil
	saved, err = repo.EnsureInspection(user, cleared)
	require.NoError(t, err)
	require.Nil(t, saved.Notes)
}

func TestRepositoryCreateCommentAndGetInspectionReturnsOrderedWithAuthor(t *testing.T) {
	clearPmcsSbsTables(t, testDB)
	author := testUser("pmcs-comment-author")
	author.Username = "jsmith"
	ensureUser(t, testDB, author)
	shopID := createShopWithMember(t, testDB, author, "member")
	vehicleID := createShopVehicle(t, testDB, shopID, author, "N2")
	repo := pmcs_sbs_progress.NewRepository(testDB)

	inspection := sampleInspection(vehicleID, author.UserID)
	_, err := repo.EnsureInspection(author, inspection)
	require.NoError(t, err)

	first, err := repo.CreateComment(author, vehicleID, inspection.ID, "first comment")
	require.NoError(t, err)
	require.Equal(t, inspection.ID, first.PmcsID)
	require.Equal(t, author.UserID, first.AuthorID)
	require.NotNil(t, first.AuthorUsername)
	require.Equal(t, "jsmith", *first.AuthorUsername)

	time.Sleep(10 * time.Millisecond)
	_, err = repo.CreateComment(author, vehicleID, inspection.ID, "second comment")
	require.NoError(t, err)

	_, _, comments, err := repo.GetInspection(author, vehicleID, inspection.ID)
	require.NoError(t, err)
	require.Len(t, comments, 2)
	require.Equal(t, "first comment", comments[0].Text)
	require.Equal(t, "second comment", comments[1].Text)
}

func TestRepositoryCreateCommentRejectsCrossVehiclePmcsID(t *testing.T) {
	clearPmcsSbsTables(t, testDB)
	user := testUser("pmcs-comment-crossveh")
	ensureUser(t, testDB, user)
	shopID := createShopWithMember(t, testDB, user, "member")
	vehicleID := createShopVehicle(t, testDB, shopID, user, "N3")
	otherVehicleID := createShopVehicle(t, testDB, shopID, user, "N4")
	repo := pmcs_sbs_progress.NewRepository(testDB)

	inspection := sampleInspection(vehicleID, user.UserID)
	_, err := repo.EnsureInspection(user, inspection)
	require.NoError(t, err)

	_, err = repo.CreateComment(user, otherVehicleID, inspection.ID, "should fail")

	require.ErrorIs(t, err, pmcs_sbs_progress.ErrInspectionNotFound)
}

func TestRepositoryUpdateCommentPersistsTextAndSetsUpdatedAt(t *testing.T) {
	clearPmcsSbsTables(t, testDB)
	author := testUser("pmcs-comment-update")
	ensureUser(t, testDB, author)
	shopID := createShopWithMember(t, testDB, author, "member")
	vehicleID := createShopVehicle(t, testDB, shopID, author, "N5")
	repo := pmcs_sbs_progress.NewRepository(testDB)

	inspection := sampleInspection(vehicleID, author.UserID)
	_, err := repo.EnsureInspection(author, inspection)
	require.NoError(t, err)

	created, err := repo.CreateComment(author, vehicleID, inspection.ID, "original text")
	require.NoError(t, err)
	require.Nil(t, created.UpdatedAt)

	updated, err := repo.UpdateComment(created.ID, "Deleted by user")
	require.NoError(t, err)
	require.Equal(t, "Deleted by user", updated.Text)
	require.NotNil(t, updated.UpdatedAt)
}

func TestRepositoryGetCommentReturnsNotFoundForMissingID(t *testing.T) {
	clearPmcsSbsTables(t, testDB)
	repo := pmcs_sbs_progress.NewRepository(testDB)

	_, err := repo.GetComment(uuid.New())

	require.ErrorIs(t, err, pmcs_sbs_progress.ErrCommentNotFound)
}

func TestRepositoryListInspectionsIncludesCommentCount(t *testing.T) {
	clearPmcsSbsTables(t, testDB)
	user := testUser("pmcs-list-comment-count")
	ensureUser(t, testDB, user)
	shopID := createShopWithMember(t, testDB, user, "member")
	vehicleID := createShopVehicle(t, testDB, shopID, user, "N6")
	repo := pmcs_sbs_progress.NewRepository(testDB)

	inspection := sampleInspection(vehicleID, user.UserID)
	_, err := repo.EnsureInspection(user, inspection)
	require.NoError(t, err)

	_, err = repo.CreateComment(user, vehicleID, inspection.ID, "one")
	require.NoError(t, err)
	_, err = repo.CreateComment(user, vehicleID, inspection.ID, "two")
	require.NoError(t, err)

	summaries, err := repo.ListInspections(user, vehicleID, "", 10, 0)
	require.NoError(t, err)
	require.Len(t, summaries, 1)
	require.Equal(t, 2, summaries[0].CommentCount)
}

func TestRepositoryDeleteInspectionCascadesComments(t *testing.T) {
	clearPmcsSbsTables(t, testDB)
	user := testUser("pmcs-comment-cascade")
	ensureUser(t, testDB, user)
	shopID := createShopWithMember(t, testDB, user, "member")
	vehicleID := createShopVehicle(t, testDB, shopID, user, "N7")
	repo := pmcs_sbs_progress.NewRepository(testDB)

	inspection := sampleInspection(vehicleID, user.UserID)
	_, err := repo.EnsureInspection(user, inspection)
	require.NoError(t, err)
	_, err = repo.CreateComment(user, vehicleID, inspection.ID, "will be cascaded")
	require.NoError(t, err)

	err = repo.DeleteInspection(user, vehicleID, inspection.ID)
	require.NoError(t, err)

	var commentCount int
	err = testDB.QueryRow(`SELECT COUNT(*) FROM pmcs_sbs_inspection_comments WHERE pmcs_id=$1`, inspection.ID).Scan(&commentCount)
	require.NoError(t, err)
	require.Equal(t, 0, commentCount)
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
