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
)

type RepositoryImpl struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *RepositoryImpl {
	return &RepositoryImpl{db: db}
}

func (repo *RepositoryImpl) ListFaults(user *bootstrap.User, equipmentID string, guideManual string) ([]model.PmcsSbsFaults, error) {
	if err := repo.requireVehicleAccess(user, equipmentID); err != nil {
		return nil, err
	}

	var rows []model.PmcsSbsFaults
	stmt := SELECT(PmcsSbsFaults.AllColumns).
		FROM(PmcsSbsFaults).
		WHERE(
			PmcsSbsFaults.EquipmentID.EQ(String(equipmentID)).
				AND(PmcsSbsFaults.GuideManual.EQ(String(guideManual))),
		).
		ORDER_BY(
			PmcsSbsFaults.SectionID.ASC(),
			PmcsSbsFaults.ItemIndex.ASC(),
		)

	if err := stmt.Query(repo.db, &rows); err != nil {
		return nil, fmt.Errorf("list pmcs sbs faults: %w", err)
	}
	return rows, nil
}

func (repo *RepositoryImpl) UpsertFault(user *bootstrap.User, fault model.PmcsSbsFaults) (*model.PmcsSbsFaults, error) {
	if err := repo.requireVehicleAccess(user, fault.EquipmentID); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	if fault.CreatedAt.IsZero() {
		fault.CreatedAt = now
	}
	fault.UpdatedAt = now

	stmt := PmcsSbsFaults.INSERT(
		PmcsSbsFaults.EquipmentID,
		PmcsSbsFaults.GuideManual,
		PmcsSbsFaults.SectionID,
		PmcsSbsFaults.ItemIndex,
		PmcsSbsFaults.ItemNo,
		PmcsSbsFaults.Status,
		PmcsSbsFaults.FaultText,
		PmcsSbsFaults.CorrectiveAction,
		PmcsSbsFaults.CreatedAt,
		PmcsSbsFaults.UpdatedAt,
	).VALUES(
		String(fault.EquipmentID),
		String(fault.GuideManual),
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
		PmcsSbsFaults.GuideManual,
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

func (repo *RepositoryImpl) DeleteFault(user *bootstrap.User, key FaultKey) error {
	if err := repo.requireVehicleAccess(user, key.EquipmentID); err != nil {
		return err
	}

	if _, err := PmcsSbsFaults.DELETE().
		WHERE(
			PmcsSbsFaults.EquipmentID.EQ(String(key.EquipmentID)).
				AND(PmcsSbsFaults.GuideManual.EQ(String(key.GuideManual))).
				AND(PmcsSbsFaults.SectionID.EQ(String(key.SectionID))).
				AND(PmcsSbsFaults.ItemIndex.EQ(Int32(key.ItemIndex))),
		).
		Exec(repo.db); err != nil {
		return fmt.Errorf("delete pmcs sbs fault: %w", err)
	}
	return nil
}

func (repo *RepositoryImpl) DeleteFaults(user *bootstrap.User, equipmentID string, guideManual string, keys []FaultKey) (int64, error) {
	return 0, errors.New("bulk delete pmcs sbs faults not implemented")
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
