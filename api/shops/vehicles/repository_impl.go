package vehicles

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"miltechserver/.gen/miltech_ng/public/model"
	. "miltechserver/.gen/miltech_ng/public/table"
	"miltechserver/api/shops/shared"
	"miltechserver/bootstrap"

	"github.com/go-jet/jet/v2/postgres"
	. "github.com/go-jet/jet/v2/postgres"
	"github.com/go-jet/jet/v2/qrm"
)

type RepositoryImpl struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *RepositoryImpl {
	return &RepositoryImpl{db: db}
}

func (repo *RepositoryImpl) CreateShopVehicle(user *bootstrap.User, vehicle model.ShopVehicle) (*model.ShopVehicle, error) {
	stmt := ShopVehicle.INSERT(
		ShopVehicle.ID,
		ShopVehicle.CreatorID,
		ShopVehicle.Niin,
		ShopVehicle.Admin,
		ShopVehicle.Model,
		ShopVehicle.Serial,
		ShopVehicle.Uoc,
		ShopVehicle.Mileage,
		ShopVehicle.Hours,
		ShopVehicle.Comment,
		ShopVehicle.SaveTime,
		ShopVehicle.LastUpdated,
		ShopVehicle.ShopID,
	).MODEL(vehicle).RETURNING(ShopVehicle.AllColumns)

	var createdVehicle model.ShopVehicle
	err := stmt.Query(repo.db, &createdVehicle)
	if err != nil {
		return nil, fmt.Errorf("failed to create shop vehicle: %w", err)
	}

	return &createdVehicle, nil
}

func (repo *RepositoryImpl) GetShopVehicles(user *bootstrap.User, shopID string) ([]model.ShopVehicle, error) {
	stmt := SELECT(ShopVehicle.AllColumns).
		FROM(ShopVehicle).
		WHERE(ShopVehicle.ShopID.EQ(String(shopID))).
		ORDER_BY(ShopVehicle.SaveTime.DESC())

	var vehicles []model.ShopVehicle
	err := stmt.Query(repo.db, &vehicles)
	if err != nil {
		return nil, fmt.Errorf("failed to get shop vehicles: %w", err)
	}

	return vehicles, nil
}

func (repo *RepositoryImpl) GetShopVehicleByID(user *bootstrap.User, vehicleID string) (*model.ShopVehicle, error) {
	stmt := SELECT(ShopVehicle.AllColumns).
		FROM(ShopVehicle).
		WHERE(ShopVehicle.ID.EQ(String(vehicleID)))

	var vehicle model.ShopVehicle
	err := stmt.Query(repo.db, &vehicle)
	if err != nil {
		return nil, fmt.Errorf("shop vehicle not found: %w", err)
	}

	return &vehicle, nil
}

func (repo *RepositoryImpl) UpdateShopVehicle(user *bootstrap.User, vehicle model.ShopVehicle) error {
	setClauses := []postgres.ColumnAssigment{
		ShopVehicle.Model.SET(String(vehicle.Model)),
		ShopVehicle.Serial.SET(String(vehicle.Serial)),
		ShopVehicle.Niin.SET(String(vehicle.Niin)),
		ShopVehicle.Uoc.SET(String(vehicle.Uoc)),
		ShopVehicle.Mileage.SET(Int32(vehicle.Mileage)),
		ShopVehicle.Hours.SET(Int32(vehicle.Hours)),
		ShopVehicle.Comment.SET(String(vehicle.Comment)),
		ShopVehicle.LastUpdated.SET(TimestampzT(vehicle.LastUpdated)),
	}

	if vehicle.TrackedMileage != nil {
		setClauses = append(setClauses, ShopVehicle.TrackedMileage.SET(Int32(*vehicle.TrackedMileage)))
	}

	if vehicle.TrackedHours != nil {
		setClauses = append(setClauses, ShopVehicle.TrackedHours.SET(Int32(*vehicle.TrackedHours)))
	}

	setArgs := make([]interface{}, len(setClauses))
	for i, clause := range setClauses {
		setArgs[i] = clause
	}

	stmt := ShopVehicle.UPDATE().SET(setArgs[0], setArgs[1:]...).WHERE(ShopVehicle.ID.EQ(String(vehicle.ID)))

	result, err := stmt.Exec(repo.db)
	if err != nil {
		return fmt.Errorf("failed to update shop vehicle: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return errors.New("vehicle not found")
	}

	return nil
}

func (repo *RepositoryImpl) UpdateShopVehicleUsage(user *bootstrap.User, update ShopVehicleUsageUpdate) error {
	setClauses := []postgres.ColumnAssigment{
		ShopVehicle.LastUpdated.SET(TimestampzT(update.LastUpdated)),
	}

	if update.TrackedMileage != nil {
		setClauses = append(setClauses, ShopVehicle.TrackedMileage.SET(Int32(*update.TrackedMileage)))
	}

	if update.TrackedHours != nil {
		setClauses = append(setClauses, ShopVehicle.TrackedHours.SET(Int32(*update.TrackedHours)))
	}

	setArgs := make([]interface{}, len(setClauses))
	for i, clause := range setClauses {
		setArgs[i] = clause
	}

	stmt := ShopVehicle.UPDATE().SET(setArgs[0], setArgs[1:]...).WHERE(ShopVehicle.ID.EQ(String(update.VehicleID)))

	result, err := stmt.Exec(repo.db)
	if err != nil {
		return fmt.Errorf("failed to update shop vehicle usage: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return errors.New("vehicle not found")
	}

	return nil
}

func (repo *RepositoryImpl) AdjustShopVehicleUsage(ctx context.Context, adjustment UsageAdjustment) (*model.ShopVehicle, error) {
	tx, err := repo.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin vehicle usage adjustment transaction: %w", err)
	}
	defer tx.Rollback()

	lockedVehicleStmt := SELECT(ShopVehicle.AllColumns).
		FROM(ShopVehicle).
		WHERE(ShopVehicle.ID.EQ(String(adjustment.VehicleID))).
		FOR(UPDATE())

	var currentVehicle model.ShopVehicle
	if err := lockedVehicleStmt.QueryContext(ctx, tx, &currentVehicle); err != nil {
		if errors.Is(err, sql.ErrNoRows) || errors.Is(err, qrm.ErrNoRows) {
			return nil, shared.ErrVehicleNotFound
		}
		return nil, fmt.Errorf("lock shop vehicle for usage adjustment: %w", err)
	}

	setClauses := []postgres.ColumnAssigment{
		ShopVehicle.LastUpdated.SET(TimestampzT(adjustment.LastUpdated)),
	}
	if adjustment.MileageAdjustment != nil {
		currentMileage := currentVehicle.Mileage
		if currentVehicle.TrackedMileage != nil {
			currentMileage = *currentVehicle.TrackedMileage
		}

		adjustedMileage, err := applyUsageAdjustment(currentMileage, *adjustment.MileageAdjustment, adjustment.Operation)
		if err != nil {
			return nil, err
		}
		setClauses = append(setClauses, ShopVehicle.TrackedMileage.SET(Int32(adjustedMileage)))
	}
	if adjustment.HoursAdjustment != nil {
		currentHours := currentVehicle.Hours
		if currentVehicle.TrackedHours != nil {
			currentHours = *currentVehicle.TrackedHours
		}

		adjustedHours, err := applyUsageAdjustment(currentHours, *adjustment.HoursAdjustment, adjustment.Operation)
		if err != nil {
			return nil, err
		}
		setClauses = append(setClauses, ShopVehicle.TrackedHours.SET(Int32(adjustedHours)))
	}

	setArgs := make([]interface{}, len(setClauses))
	for index, clause := range setClauses {
		setArgs[index] = clause
	}

	stmt := ShopVehicle.UPDATE().
		SET(setArgs[0], setArgs[1:]...).
		WHERE(ShopVehicle.ID.EQ(String(adjustment.VehicleID))).
		RETURNING(ShopVehicle.AllColumns)

	var updatedVehicle model.ShopVehicle
	if err := stmt.QueryContext(ctx, tx, &updatedVehicle); err != nil {
		if errors.Is(err, sql.ErrNoRows) || errors.Is(err, qrm.ErrNoRows) {
			return nil, shared.ErrVehicleNotFound
		}
		return nil, fmt.Errorf("update shop vehicle usage: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit vehicle usage adjustment transaction: %w", err)
	}

	return &updatedVehicle, nil
}

func applyUsageAdjustment(current int32, magnitude int32, operation UsageAdjustmentOperation) (int32, error) {
	result := int64(current)
	if operation == UsageOperationSubtract {
		result -= int64(magnitude)
	} else {
		result += int64(magnitude)
	}
	if result < 0 || result > math.MaxInt32 {
		return 0, ErrUsageOutOfRange
	}
	return int32(result), nil
}

func (repo *RepositoryImpl) DeleteShopVehicle(user *bootstrap.User, vehicleID string) error {
	stmt := ShopVehicle.DELETE().
		WHERE(ShopVehicle.ID.EQ(String(vehicleID)))

	result, err := stmt.Exec(repo.db)
	if err != nil {
		return fmt.Errorf("failed to delete shop vehicle: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return errors.New("vehicle not found")
	}

	return nil
}

func (repo *RepositoryImpl) CreateNotificationChange(user *bootstrap.User, change model.ShopVehicleNotificationChanges) error {
	rawSQL := `
		INSERT INTO shop_vehicle_notification_changes (
			notification_id,
			shop_id,
			vehicle_id,
			changed_by,
			change_type,
			field_changes,
			notification_title,
			notification_type,
			vehicle_admin
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err := repo.db.Exec(
		rawSQL,
		change.NotificationID,
		change.ShopID,
		change.VehicleID,
		change.ChangedBy,
		change.ChangeType,
		change.FieldChanges,
		change.NotificationTitle,
		change.NotificationType,
		change.VehicleAdmin,
	)
	if err != nil {
		return fmt.Errorf("failed to create notification change record: %w", err)
	}

	return nil
}
