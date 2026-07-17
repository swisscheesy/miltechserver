package pmcs_sbs_progress

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"miltechserver/.gen/miltech_ng/public/model"
	. "miltechserver/.gen/miltech_ng/public/table"
	"miltechserver/bootstrap"

	. "github.com/go-jet/jet/v2/postgres"
	"github.com/go-jet/jet/v2/qrm"
	"github.com/google/uuid"
)

type RepositoryImpl struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *RepositoryImpl {
	return &RepositoryImpl{db: db}
}

func (repo *RepositoryImpl) EnsureInspection(user *bootstrap.User, inspection model.PmcsSbsInspections) (*model.PmcsSbsInspections, error) {
	if err := repo.requireVehicleAccess(user, inspection.EquipmentID); err != nil {
		return nil, err
	}
	return ensureInspection(repo.db, inspection)
}

func (repo *RepositoryImpl) GetInspection(user *bootstrap.User, equipmentID string, pmcsID uuid.UUID) (*model.PmcsSbsInspections, []model.PmcsSbsFaults, error) {
	if err := repo.requireVehicleAccess(user, equipmentID); err != nil {
		return nil, nil, err
	}

	var inspection model.PmcsSbsInspections
	stmt := SELECT(PmcsSbsInspections.AllColumns).
		FROM(PmcsSbsInspections).
		WHERE(
			PmcsSbsInspections.ID.EQ(UUID(pmcsID)).
				AND(PmcsSbsInspections.EquipmentID.EQ(String(equipmentID))),
		)

	if err := stmt.Query(repo.db, &inspection); err != nil {
		if errors.Is(err, sql.ErrNoRows) || errors.Is(err, qrm.ErrNoRows) {
			return nil, nil, ErrInspectionNotFound
		}
		return nil, nil, fmt.Errorf("get pmcs sbs inspection: %w", err)
	}

	var faults []model.PmcsSbsFaults
	faultsStmt := SELECT(PmcsSbsFaults.AllColumns).
		FROM(PmcsSbsFaults).
		WHERE(PmcsSbsFaults.PmcsID.EQ(UUID(pmcsID))).
		ORDER_BY(PmcsSbsFaults.SectionID.ASC(), PmcsSbsFaults.ItemIndex.ASC())

	if err := faultsStmt.Query(repo.db, &faults); err != nil {
		return nil, nil, fmt.Errorf("list pmcs sbs inspection faults: %w", err)
	}

	return &inspection, faults, nil
}

func (repo *RepositoryImpl) ListInspections(user *bootstrap.User, equipmentID string, guideManual string, limit int, offset int) ([]InspectionSummary, error) {
	if err := repo.requireVehicleAccess(user, equipmentID); err != nil {
		return nil, err
	}

	condition := PmcsSbsInspections.EquipmentID.EQ(String(equipmentID))
	if guideManual != "" {
		condition = condition.AND(PmcsSbsInspections.GuideManual.EQ(String(guideManual)))
	}

	var inspections []model.PmcsSbsInspections
	stmt := SELECT(PmcsSbsInspections.AllColumns).
		FROM(PmcsSbsInspections).
		WHERE(condition).
		ORDER_BY(PmcsSbsInspections.PerformedDate.DESC()).
		LIMIT(int64(limit)).
		OFFSET(int64(offset))

	if err := stmt.Query(repo.db, &inspections); err != nil {
		return nil, fmt.Errorf("list pmcs sbs inspections: %w", err)
	}
	if len(inspections) == 0 {
		return []InspectionSummary{}, nil
	}

	countByID := make(map[uuid.UUID]int, len(inspections))
	for _, inspection := range inspections {
		countByID[inspection.ID] = 0
	}

	// Count faults for each inspection
	for _, inspection := range inspections {
		var count int
		err := repo.db.QueryRow(
			`SELECT COUNT(*) FROM pmcs_sbs_faults WHERE pmcs_id = $1`,
			inspection.ID,
		).Scan(&count)
		if err != nil {
			return nil, fmt.Errorf("count pmcs sbs faults: %w", err)
		}
		countByID[inspection.ID] = count
	}

	summaries := make([]InspectionSummary, 0, len(inspections))
	for _, inspection := range inspections {
		summaries = append(summaries, InspectionSummary{
			ID:            inspection.ID,
			GuideManual:   inspection.GuideManual,
			PerformedDate: inspection.PerformedDate,
			FaultCount:    countByID[inspection.ID],
			CreatedAt:     inspection.CreatedAt,
		})
	}
	return summaries, nil
}

func (repo *RepositoryImpl) DeleteInspection(user *bootstrap.User, equipmentID string, pmcsID uuid.UUID) error {
	if err := repo.requireVehicleAccess(user, equipmentID); err != nil {
		return err
	}

	result, err := PmcsSbsInspections.DELETE().
		WHERE(
			PmcsSbsInspections.ID.EQ(UUID(pmcsID)).
				AND(PmcsSbsInspections.EquipmentID.EQ(String(equipmentID))),
		).
		Exec(repo.db)
	if err != nil {
		return fmt.Errorf("delete pmcs sbs inspection: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete pmcs sbs inspection rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrInspectionNotFound
	}
	return nil
}

func (repo *RepositoryImpl) UpsertFault(user *bootstrap.User, inspection model.PmcsSbsInspections, fault model.PmcsSbsFaults) (*model.PmcsSbsFaults, error) {
	if err := repo.requireVehicleAccess(user, inspection.EquipmentID); err != nil {
		return nil, err
	}

	tx, err := repo.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("begin upsert pmcs sbs fault transaction: %w", err)
	}
	defer tx.Rollback()

	savedInspection, err := ensureInspection(tx, inspection)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	fault.PmcsID = savedInspection.ID
	if fault.CreatedAt.IsZero() {
		fault.CreatedAt = now
	}
	fault.UpdatedAt = now

	stmt := PmcsSbsFaults.INSERT(
		PmcsSbsFaults.PmcsID,
		PmcsSbsFaults.SectionID,
		PmcsSbsFaults.ItemIndex,
		PmcsSbsFaults.ItemNo,
		PmcsSbsFaults.Status,
		PmcsSbsFaults.FaultText,
		PmcsSbsFaults.CorrectiveAction,
		PmcsSbsFaults.CreatedAt,
		PmcsSbsFaults.UpdatedAt,
	).VALUES(
		UUID(fault.PmcsID),
		String(fault.SectionID),
		Int32(fault.ItemIndex),
		String(fault.ItemNo),
		String(fault.Status),
		String(fault.FaultText),
		String(fault.CorrectiveAction),
		TimestampzT(fault.CreatedAt),
		TimestampzT(now),
	).ON_CONFLICT(
		PmcsSbsFaults.PmcsID,
		PmcsSbsFaults.SectionID,
		PmcsSbsFaults.ItemIndex,
	).DO_UPDATE(SET(
		PmcsSbsFaults.ItemNo.SET(String(fault.ItemNo)),
		PmcsSbsFaults.Status.SET(String(fault.Status)),
		PmcsSbsFaults.FaultText.SET(String(fault.FaultText)),
		PmcsSbsFaults.CorrectiveAction.SET(String(fault.CorrectiveAction)),
		PmcsSbsFaults.UpdatedAt.SET(TimestampzT(now)),
	)).RETURNING(PmcsSbsFaults.AllColumns)

	var saved model.PmcsSbsFaults
	if err := stmt.Query(tx, &saved); err != nil {
		return nil, fmt.Errorf("upsert pmcs sbs fault: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit upsert pmcs sbs fault transaction: %w", err)
	}
	return &saved, nil
}

func (repo *RepositoryImpl) DeleteFault(user *bootstrap.User, equipmentID string, key FaultKey) error {
	if err := repo.requireVehicleAccess(user, equipmentID); err != nil {
		return err
	}
	if err := repo.requireInspectionOwnership(repo.db, equipmentID, key.PmcsID); err != nil {
		return err
	}

	if _, err := PmcsSbsFaults.DELETE().
		WHERE(
			PmcsSbsFaults.PmcsID.EQ(UUID(key.PmcsID)).
				AND(PmcsSbsFaults.SectionID.EQ(String(key.SectionID))).
				AND(PmcsSbsFaults.ItemIndex.EQ(Int32(key.ItemIndex))),
		).
		Exec(repo.db); err != nil {
		return fmt.Errorf("delete pmcs sbs fault: %w", err)
	}
	return nil
}

func (repo *RepositoryImpl) DeleteFaults(user *bootstrap.User, equipmentID string, pmcsID uuid.UUID, keys []FaultKey) (int64, error) {
	if err := repo.requireVehicleAccess(user, equipmentID); err != nil {
		return 0, err
	}
	if err := repo.requireInspectionOwnership(repo.db, equipmentID, pmcsID); err != nil {
		return 0, err
	}
	if len(keys) == 0 {
		return 0, nil
	}

	keyRows := make([]Expression, 0, len(keys))
	for _, key := range keys {
		keyRows = append(keyRows, ROW(String(key.SectionID), Int32(key.ItemIndex)))
	}

	result, err := PmcsSbsFaults.DELETE().
		WHERE(
			PmcsSbsFaults.PmcsID.EQ(UUID(pmcsID)).
				AND(ROW(PmcsSbsFaults.SectionID, PmcsSbsFaults.ItemIndex).IN(keyRows...)),
		).
		Exec(repo.db)
	if err != nil {
		return 0, fmt.Errorf("bulk delete pmcs sbs faults: %w", err)
	}
	deletedCount, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("bulk delete pmcs sbs faults rows affected: %w", err)
	}
	return deletedCount, nil
}

func (repo *RepositoryImpl) requireVehicleAccess(user *bootstrap.User, equipmentID string) error {
	if user == nil {
		return ErrUnauthorized
	}

	stmt := SELECT(Int(1).AS("exists")).
		FROM(
			ShopVehicle.
				INNER_JOIN(ShopMembers, ShopMembers.ShopID.EQ(ShopVehicle.ShopID)),
		).
		WHERE(
			ShopVehicle.ID.EQ(String(equipmentID)).
				AND(ShopMembers.UserID.EQ(String(user.UserID))),
		).
		LIMIT(1)

	var rows []struct {
		Exists int `sql:"exists"`
	}
	if err := stmt.Query(repo.db, &rows); err != nil {
		if errors.Is(err, sql.ErrNoRows) || errors.Is(err, qrm.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("authorize pmcs sbs vehicle fault access: %w", err)
	}
	if len(rows) == 0 {
		return ErrNotFound
	}
	return nil
}

func (repo *RepositoryImpl) requireInspectionOwnership(queryable qrm.Queryable, equipmentID string, pmcsID uuid.UUID) error {
	stmt := SELECT(Int(1).AS("exists")).
		FROM(PmcsSbsInspections).
		WHERE(
			PmcsSbsInspections.ID.EQ(UUID(pmcsID)).
				AND(PmcsSbsInspections.EquipmentID.EQ(String(equipmentID))),
		).
		LIMIT(1)

	var rows []struct {
		Exists int `sql:"exists"`
	}
	if err := stmt.Query(queryable, &rows); err != nil {
		return fmt.Errorf("authorize pmcs sbs inspection access: %w", err)
	}
	if len(rows) == 0 {
		return ErrInspectionNotFound
	}
	return nil
}

// ensureInspection inserts the inspection if it doesn't exist yet, or, if a
// row with this id already exists, verifies equipment_id and guide_manual
// match and updates performed_date. A mismatch on either field returns
// ErrInspectionConflict. queryable is either *sql.DB (standalone calls) or
// *sql.Tx (the implicit-creation path inside UpsertFault) — both satisfy
// qrm.Queryable.
func ensureInspection(queryable qrm.Queryable, inspection model.PmcsSbsInspections) (*model.PmcsSbsInspections, error) {
	now := time.Now().UTC()
	var createdByExpr Expression = NULL
	if inspection.CreatedBy != nil {
		createdByExpr = String(*inspection.CreatedBy)
	}

	stmt := PmcsSbsInspections.INSERT(
		PmcsSbsInspections.ID,
		PmcsSbsInspections.EquipmentID,
		PmcsSbsInspections.GuideManual,
		PmcsSbsInspections.PerformedDate,
		PmcsSbsInspections.CreatedBy,
		PmcsSbsInspections.CreatedAt,
		PmcsSbsInspections.UpdatedAt,
	).VALUES(
		UUID(inspection.ID),
		String(inspection.EquipmentID),
		String(inspection.GuideManual),
		TimestampzT(inspection.PerformedDate),
		createdByExpr,
		TimestampzT(now),
		TimestampzT(now),
	).ON_CONFLICT(PmcsSbsInspections.ID).DO_UPDATE(
		SET(
			PmcsSbsInspections.PerformedDate.SET(TimestampzT(inspection.PerformedDate)),
			PmcsSbsInspections.UpdatedAt.SET(TimestampzT(now)),
		).WHERE(
			PmcsSbsInspections.EquipmentID.EQ(String(inspection.EquipmentID)).
				AND(PmcsSbsInspections.GuideManual.EQ(String(inspection.GuideManual))),
		),
	).RETURNING(PmcsSbsInspections.AllColumns)

	var saved model.PmcsSbsInspections
	if err := stmt.Query(queryable, &saved); err != nil {
		if errors.Is(err, sql.ErrNoRows) || errors.Is(err, qrm.ErrNoRows) {
			return nil, ErrInspectionConflict
		}
		return nil, fmt.Errorf("ensure pmcs sbs inspection: %w", err)
	}
	return &saved, nil
}
