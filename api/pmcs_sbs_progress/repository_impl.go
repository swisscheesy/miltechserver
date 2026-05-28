package pmcs_sbs_progress

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"miltechserver/.gen/miltech_ng/public/model"
	. "miltechserver/.gen/miltech_ng/public/table"
	"miltechserver/bootstrap"

	. "github.com/go-jet/jet/v2/postgres"
	"github.com/go-jet/jet/v2/qrm"
)

type RepositoryImpl struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *RepositoryImpl {
	return &RepositoryImpl{db: db}
}

func (repo *RepositoryImpl) ListEquipment(user *bootstrap.User) ([]model.PmcsSbsEquipment, error) {
	if user == nil {
		return nil, ErrUnauthorized
	}

	var rows []model.PmcsSbsEquipment
	stmt := SELECT(PmcsSbsEquipment.AllColumns).
		FROM(PmcsSbsEquipment).
		WHERE(PmcsSbsEquipment.UserUID.EQ(String(user.UserID))).
		ORDER_BY(PmcsSbsEquipment.UpdatedAt.DESC())

	if err := stmt.Query(repo.db, &rows); err != nil {
		return nil, fmt.Errorf("list pmcs sbs equipment: %w", err)
	}
	return rows, nil
}

func (repo *RepositoryImpl) GetEquipmentAggregate(user *bootstrap.User, equipmentID string) (*EquipmentAggregate, error) {
	equipment, err := repo.getEquipmentByID(repo.db, user, equipmentID)
	if err != nil {
		return nil, err
	}

	completions, err := repo.getCompletions(repo.db, equipmentID)
	if err != nil {
		return nil, err
	}

	faults, err := repo.getFaults(repo.db, equipmentID)
	if err != nil {
		return nil, err
	}

	return &EquipmentAggregate{
		Equipment:   *equipment,
		Completions: completions,
		Faults:      faults,
	}, nil
}

func (repo *RepositoryImpl) UpsertEquipment(user *bootstrap.User, equipment model.PmcsSbsEquipment) (*model.PmcsSbsEquipment, error) {
	if user == nil {
		return nil, ErrUnauthorized
	}

	now := time.Now().UTC()
	equipment.UserUID = user.UserID
	equipment.UpdatedAt = now
	if equipment.CreatedAt.IsZero() {
		equipment.CreatedAt = now
	}

	stmt := PmcsSbsEquipment.INSERT(
		PmcsSbsEquipment.ID,
		PmcsSbsEquipment.UserUID,
		PmcsSbsEquipment.EquipmentManual,
		PmcsSbsEquipment.Admin,
		PmcsSbsEquipment.Serial,
		PmcsSbsEquipment.Uic,
		PmcsSbsEquipment.CreatedAt,
		PmcsSbsEquipment.UpdatedAt,
	).MODEL(equipment).
		ON_CONFLICT(PmcsSbsEquipment.ID).
		DO_UPDATE(
			SET(
				PmcsSbsEquipment.EquipmentManual.SET(String(equipment.EquipmentManual)),
				PmcsSbsEquipment.Admin.SET(String(equipment.Admin)),
				PmcsSbsEquipment.Serial.SET(String(equipment.Serial)),
				PmcsSbsEquipment.Uic.SET(String(equipment.Uic)),
				PmcsSbsEquipment.UpdatedAt.SET(TimestampzT(now)),
			).WHERE(PmcsSbsEquipment.UserUID.EQ(String(user.UserID))),
		).
		RETURNING(PmcsSbsEquipment.AllColumns)

	var saved model.PmcsSbsEquipment
	if err := stmt.Query(repo.db, &saved); err != nil {
		if errors.Is(err, sql.ErrNoRows) || errors.Is(err, qrm.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("upsert pmcs sbs equipment: %w", err)
	}
	return &saved, nil
}

func (repo *RepositoryImpl) DeleteEquipment(user *bootstrap.User, equipmentID string) error {
	if user == nil {
		return ErrUnauthorized
	}

	tx, err := repo.db.Begin()
	if err != nil {
		return fmt.Errorf("begin delete pmcs sbs equipment: %w", err)
	}
	defer tx.Rollback()

	if _, err := repo.getEquipmentByID(tx, user, equipmentID); err != nil {
		return err
	}

	if _, err := PmcsSbsFaults.DELETE().
		WHERE(PmcsSbsFaults.EquipmentID.EQ(String(equipmentID))).
		Exec(tx); err != nil {
		return fmt.Errorf("delete pmcs sbs faults: %w", err)
	}

	if _, err := PmcsSbsCompletions.DELETE().
		WHERE(PmcsSbsCompletions.EquipmentID.EQ(String(equipmentID))).
		Exec(tx); err != nil {
		return fmt.Errorf("delete pmcs sbs completions: %w", err)
	}

	result, err := PmcsSbsEquipment.DELETE().
		WHERE(
			PmcsSbsEquipment.ID.EQ(String(equipmentID)).
				AND(PmcsSbsEquipment.UserUID.EQ(String(user.UserID))),
		).
		Exec(tx)
	if err != nil {
		return fmt.Errorf("delete pmcs sbs equipment: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete pmcs sbs equipment rows affected: %w", err)
	}
	if rows == 0 {
		return ErrNotFound
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit delete pmcs sbs equipment: %w", err)
	}
	slog.Info("pmcs sbs equipment deleted", "user_id", user.UserID, "equipment_id", equipmentID)
	return nil
}

func (repo *RepositoryImpl) getEquipmentByID(db qrm.Queryable, user *bootstrap.User, equipmentID string) (*model.PmcsSbsEquipment, error) {
	if user == nil {
		return nil, ErrUnauthorized
	}

	var row model.PmcsSbsEquipment
	stmt := SELECT(PmcsSbsEquipment.AllColumns).
		FROM(PmcsSbsEquipment).
		WHERE(
			PmcsSbsEquipment.ID.EQ(String(equipmentID)).
				AND(PmcsSbsEquipment.UserUID.EQ(String(user.UserID))),
		)

	if err := stmt.Query(db, &row); err != nil {
		if errors.Is(err, sql.ErrNoRows) || errors.Is(err, qrm.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get pmcs sbs equipment: %w", err)
	}
	if row.ID.String() == "00000000-0000-0000-0000-000000000000" {
		return nil, ErrNotFound
	}

	return &row, nil
}

func (repo *RepositoryImpl) getCompletions(db qrm.Queryable, equipmentID string) ([]model.PmcsSbsCompletions, error) {
	var rows []model.PmcsSbsCompletions
	stmt := SELECT(PmcsSbsCompletions.AllColumns).
		FROM(PmcsSbsCompletions).
		WHERE(PmcsSbsCompletions.EquipmentID.EQ(String(equipmentID))).
		ORDER_BY(
			PmcsSbsCompletions.SectionID.ASC(),
			PmcsSbsCompletions.ItemIndex.ASC(),
			PmcsSbsCompletions.StepID.ASC(),
		)

	if err := stmt.Query(db, &rows); err != nil {
		return nil, fmt.Errorf("get pmcs sbs completions: %w", err)
	}
	return rows, nil
}

func (repo *RepositoryImpl) getFaults(db qrm.Queryable, equipmentID string) ([]model.PmcsSbsFaults, error) {
	var rows []model.PmcsSbsFaults
	stmt := SELECT(PmcsSbsFaults.AllColumns).
		FROM(PmcsSbsFaults).
		WHERE(PmcsSbsFaults.EquipmentID.EQ(String(equipmentID))).
		ORDER_BY(
			PmcsSbsFaults.SectionID.ASC(),
			PmcsSbsFaults.ItemIndex.ASC(),
		)

	if err := stmt.Query(db, &rows); err != nil {
		return nil, fmt.Errorf("get pmcs sbs faults: %w", err)
	}
	return rows, nil
}

func (repo *RepositoryImpl) UpsertCompletion(user *bootstrap.User, completion model.PmcsSbsCompletions) (*model.PmcsSbsCompletions, error) {
	if _, err := repo.getEquipmentByID(repo.db, user, completion.EquipmentID.String()); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	completion.IsComplete = true
	completion.UpdatedAt = now

	stmt := PmcsSbsCompletions.INSERT(
		PmcsSbsCompletions.EquipmentID,
		PmcsSbsCompletions.SectionID,
		PmcsSbsCompletions.ItemIndex,
		PmcsSbsCompletions.ItemNo,
		PmcsSbsCompletions.StepID,
		PmcsSbsCompletions.IsComplete,
		PmcsSbsCompletions.UpdatedAt,
	).VALUES(
		String(completion.EquipmentID.String()),
		String(completion.SectionID),
		Int32(completion.ItemIndex),
		String(completion.ItemNo),
		String(completion.StepID),
		Bool(true),
		TimestampzT(now),
	).ON_CONFLICT(
		PmcsSbsCompletions.EquipmentID,
		PmcsSbsCompletions.SectionID,
		PmcsSbsCompletions.ItemIndex,
		PmcsSbsCompletions.StepID,
	).DO_UPDATE(SET(
		PmcsSbsCompletions.ItemNo.SET(String(completion.ItemNo)),
		PmcsSbsCompletions.IsComplete.SET(Bool(true)),
		PmcsSbsCompletions.UpdatedAt.SET(TimestampzT(now)),
	)).RETURNING(PmcsSbsCompletions.AllColumns)

	var saved model.PmcsSbsCompletions
	if err := stmt.Query(repo.db, &saved); err != nil {
		return nil, fmt.Errorf("upsert pmcs sbs completion: %w", err)
	}
	return &saved, nil
}

func (repo *RepositoryImpl) DeleteCompletion(user *bootstrap.User, equipmentID string, sectionID string, itemIndex int32, stepID string) error {
	if _, err := repo.getEquipmentByID(repo.db, user, equipmentID); err != nil {
		return err
	}

	_, err := PmcsSbsCompletions.DELETE().
		WHERE(
			PmcsSbsCompletions.EquipmentID.EQ(String(equipmentID)).
				AND(PmcsSbsCompletions.SectionID.EQ(String(sectionID))).
				AND(PmcsSbsCompletions.ItemIndex.EQ(Int32(itemIndex))).
				AND(PmcsSbsCompletions.StepID.EQ(String(stepID))),
		).
		Exec(repo.db)
	if err != nil {
		return fmt.Errorf("delete pmcs sbs completion: %w", err)
	}
	return nil
}

func (repo *RepositoryImpl) UpsertFault(user *bootstrap.User, fault model.PmcsSbsFaults) (*model.PmcsSbsFaults, error) {
	if _, err := repo.getEquipmentByID(repo.db, user, fault.EquipmentID.String()); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	if fault.CreatedAt.IsZero() {
		fault.CreatedAt = now
	}
	fault.UpdatedAt = now

	stmt := PmcsSbsFaults.INSERT(
		PmcsSbsFaults.EquipmentID,
		PmcsSbsFaults.SectionID,
		PmcsSbsFaults.ItemIndex,
		PmcsSbsFaults.ItemNo,
		PmcsSbsFaults.Status,
		PmcsSbsFaults.FaultText,
		PmcsSbsFaults.CorrectiveAction,
		PmcsSbsFaults.CreatedAt,
		PmcsSbsFaults.UpdatedAt,
	).VALUES(
		String(fault.EquipmentID.String()),
		String(fault.SectionID),
		Int32(fault.ItemIndex),
		String(fault.ItemNo),
		String(fault.Status),
		String(fault.FaultText),
		String(fault.CorrectiveAction),
		TimestampzT(fault.CreatedAt),
		TimestampzT(now),
	).ON_CONFLICT(
		PmcsSbsFaults.EquipmentID,
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
	if err := stmt.Query(repo.db, &saved); err != nil {
		return nil, fmt.Errorf("upsert pmcs sbs fault: %w", err)
	}
	return &saved, nil
}

func (repo *RepositoryImpl) DeleteFault(user *bootstrap.User, equipmentID string, sectionID string, itemIndex int32) error {
	if _, err := repo.getEquipmentByID(repo.db, user, equipmentID); err != nil {
		return err
	}

	_, err := PmcsSbsFaults.DELETE().
		WHERE(
			PmcsSbsFaults.EquipmentID.EQ(String(equipmentID)).
				AND(PmcsSbsFaults.SectionID.EQ(String(sectionID))).
				AND(PmcsSbsFaults.ItemIndex.EQ(Int32(itemIndex))),
		).
		Exec(repo.db)
	if err != nil {
		return fmt.Errorf("delete pmcs sbs fault: %w", err)
	}
	return nil
}

func (repo *RepositoryImpl) Sync(_ *bootstrap.User, _ SyncChangeSet) (*SyncResult, error) {
	return nil, errors.New("batch sync persistence starts in task 5")
}
