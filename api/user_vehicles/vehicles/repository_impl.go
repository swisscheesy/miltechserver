package vehicles

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"miltechserver/.gen/miltech_ng/public/model"
	. "miltechserver/.gen/miltech_ng/public/table"
	"miltechserver/bootstrap"

	. "github.com/go-jet/jet/v2/postgres"
)

type RepositoryImpl struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *RepositoryImpl {
	return &RepositoryImpl{db: db}
}

func (repo *RepositoryImpl) GetByUserID(user *bootstrap.User) ([]model.UserVehicle, error) {
	var vehicles []model.UserVehicle

	if user == nil {
		return nil, errors.New("valid user not found")
	}

	stmt := SELECT(UserVehicle.AllColumns).
		FROM(UserVehicle).
		WHERE(UserVehicle.UserID.EQ(String(user.UserID)))

	err := stmt.Query(repo.db, &vehicles)
	if err != nil {
		return nil, fmt.Errorf("error retrieving vehicles for user %s: %w", user.UserID, err)
	}

	slog.Info("vehicles retrieved for user", "user_id", user.UserID, "count", len(vehicles))
	return vehicles, nil
}

func (repo *RepositoryImpl) GetByID(user *bootstrap.User, vehicleID string) (*model.UserVehicle, error) {
	var vehicle model.UserVehicle

	if user == nil {
		return nil, errors.New("valid user not found")
	}

	stmt := SELECT(UserVehicle.AllColumns).
		FROM(UserVehicle).
		WHERE(UserVehicle.UserID.EQ(String(user.UserID)).
			AND(UserVehicle.ID.EQ(String(vehicleID))))

	err := stmt.Query(repo.db, &vehicle)
	if err != nil {
		return nil, fmt.Errorf("vehicle not found for user %s: %w", user.UserID, err)
	}

	slog.Info("vehicle retrieved", "user_id", user.UserID, "vehicle_id", vehicleID)
	return &vehicle, nil
}

func (repo *RepositoryImpl) Upsert(user *bootstrap.User, vehicle model.UserVehicle) error {
	if user == nil {
		return errors.New("valid user not found")
	}

	stmt := UserVehicle.INSERT(
		UserVehicle.ID, UserVehicle.UserID, UserVehicle.Niin, UserVehicle.Admin,
		UserVehicle.Model, UserVehicle.Serial, UserVehicle.Uoc, UserVehicle.Mileage,
		UserVehicle.Hours, UserVehicle.Comment, UserVehicle.SaveTime, UserVehicle.LastUpdated).
		MODEL(vehicle).
		ON_CONFLICT(UserVehicle.ID).
		DO_UPDATE(
			SET(
				UserVehicle.Niin.SET(String(vehicle.Niin)),
				UserVehicle.Admin.SET(String(vehicle.Admin)),
				UserVehicle.Model.SET(String(vehicle.Model)),
				UserVehicle.Serial.SET(String(vehicle.Serial)),
				UserVehicle.Uoc.SET(String(vehicle.Uoc)),
				UserVehicle.Mileage.SET(Int32(vehicle.Mileage)),
				UserVehicle.Hours.SET(Int32(vehicle.Hours)),
				UserVehicle.Comment.SET(String(vehicle.Comment)),
				UserVehicle.LastUpdated.SET(TimestampzT(vehicle.LastUpdated))).
				WHERE(UserVehicle.ID.EQ(String(vehicle.ID)))).
		RETURNING(UserVehicle.AllColumns)

	err := stmt.Query(repo.db, &vehicle)
	if err != nil {
		return fmt.Errorf("error saving vehicle: %w", err)
	}

	slog.Info("vehicle saved", "user_id", user.UserID, "vehicle_id", vehicle.ID)
	return nil
}

func (repo *RepositoryImpl) Delete(user *bootstrap.User, vehicleID string) error {
	if user == nil {
		return errors.New("valid user not found")
	}

	stmt := UserVehicle.DELETE().
		WHERE(UserVehicle.UserID.EQ(String(user.UserID)).
			AND(UserVehicle.ID.EQ(String(vehicleID))))

	_, err := stmt.Exec(repo.db)
	if err != nil {
		return fmt.Errorf("error deleting vehicle: %w", err)
	}

	slog.Info("vehicle deleted", "user_id", user.UserID, "vehicle_id", vehicleID)
	return nil
}

func (repo *RepositoryImpl) DeleteAll(user *bootstrap.User) error {
	if user == nil {
		return errors.New("valid user not found")
	}

	stmt := UserVehicle.DELETE().
		WHERE(UserVehicle.UserID.EQ(String(user.UserID)))

	_, err := stmt.Exec(repo.db)
	if err != nil {
		return fmt.Errorf("error deleting all vehicles: %w", err)
	}

	slog.Info("all vehicles deleted", "user_id", user.UserID)
	return nil
}
