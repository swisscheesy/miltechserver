package vehicles

import (
	"database/sql"
	"errors"
	"fmt"
	"miltechserver/.gen/miltech_ng/public/model"
	. "miltechserver/.gen/miltech_ng/public/table"
	"miltechserver/bootstrap"

	"github.com/go-jet/jet/v2/postgres"
	. "github.com/go-jet/jet/v2/postgres"
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
