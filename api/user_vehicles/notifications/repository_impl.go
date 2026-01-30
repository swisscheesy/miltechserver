package notifications

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

func (repo *RepositoryImpl) GetByUserID(user *bootstrap.User) ([]model.UserVehicleNotifications, error) {
	var notifications []model.UserVehicleNotifications

	if user == nil {
		return nil, errors.New("valid user not found")
	}

	stmt := SELECT(UserVehicleNotifications.AllColumns).
		FROM(UserVehicleNotifications).
		WHERE(UserVehicleNotifications.UserID.EQ(String(user.UserID)))

	err := stmt.Query(repo.db, &notifications)
	if err != nil {
		return nil, fmt.Errorf("error retrieving notifications for user %s: %w", user.UserID, err)
	}

	slog.Info("vehicle notifications retrieved for user", "user_id", user.UserID, "count", len(notifications))
	return notifications, nil
}

func (repo *RepositoryImpl) GetByVehicleID(user *bootstrap.User, vehicleID string) ([]model.UserVehicleNotifications, error) {
	var notifications []model.UserVehicleNotifications

	if user == nil {
		return nil, errors.New("valid user not found")
	}

	stmt := SELECT(UserVehicleNotifications.AllColumns).
		FROM(UserVehicleNotifications).
		WHERE(UserVehicleNotifications.UserID.EQ(String(user.UserID)).
			AND(UserVehicleNotifications.VehicleID.EQ(String(vehicleID))))

	err := stmt.Query(repo.db, &notifications)
	if err != nil {
		return nil, fmt.Errorf("error retrieving notifications for vehicle %s: %w", vehicleID, err)
	}

	slog.Info("vehicle notifications retrieved", "user_id", user.UserID, "vehicle_id", vehicleID, "count", len(notifications))
	return notifications, nil
}

func (repo *RepositoryImpl) GetByID(user *bootstrap.User, notificationID string) (*model.UserVehicleNotifications, error) {
	var notification model.UserVehicleNotifications

	if user == nil {
		return nil, errors.New("valid user not found")
	}

	stmt := SELECT(UserVehicleNotifications.AllColumns).
		FROM(UserVehicleNotifications).
		WHERE(UserVehicleNotifications.UserID.EQ(String(user.UserID)).
			AND(UserVehicleNotifications.ID.EQ(String(notificationID))))

	err := stmt.Query(repo.db, &notification)
	if err != nil {
		return nil, fmt.Errorf("notification not found for user %s: %w", user.UserID, err)
	}

	slog.Info("vehicle notification retrieved", "user_id", user.UserID, "notification_id", notificationID)
	return &notification, nil
}

func (repo *RepositoryImpl) Upsert(user *bootstrap.User, notification model.UserVehicleNotifications) error {
	if user == nil {
		return errors.New("valid user not found")
	}

	stmt := UserVehicleNotifications.INSERT(
		UserVehicleNotifications.ID, UserVehicleNotifications.UserID, UserVehicleNotifications.VehicleID,
		UserVehicleNotifications.Title, UserVehicleNotifications.Description, UserVehicleNotifications.Type,
		UserVehicleNotifications.Completed, UserVehicleNotifications.SaveTime, UserVehicleNotifications.LastUpdated).
		MODEL(notification).
		ON_CONFLICT(UserVehicleNotifications.ID).
		DO_UPDATE(
			SET(
				UserVehicleNotifications.VehicleID.SET(String(notification.VehicleID)),
				UserVehicleNotifications.Title.SET(String(notification.Title)),
				UserVehicleNotifications.Description.SET(String(notification.Description)),
				UserVehicleNotifications.Type.SET(String(notification.Type)),
				UserVehicleNotifications.Completed.SET(Bool(notification.Completed)),
				UserVehicleNotifications.LastUpdated.SET(TimestampzT(notification.LastUpdated))).
				WHERE(UserVehicleNotifications.ID.EQ(String(notification.ID)))).
		RETURNING(UserVehicleNotifications.AllColumns)

	err := stmt.Query(repo.db, &notification)
	if err != nil {
		return fmt.Errorf("error saving notification: %w", err)
	}

	slog.Info("vehicle notification saved", "user_id", user.UserID, "notification_id", notification.ID)
	return nil
}

func (repo *RepositoryImpl) Delete(user *bootstrap.User, notificationID string) error {
	if user == nil {
		return errors.New("valid user not found")
	}

	stmt := UserVehicleNotifications.DELETE().
		WHERE(UserVehicleNotifications.UserID.EQ(String(user.UserID)).
			AND(UserVehicleNotifications.ID.EQ(String(notificationID))))

	_, err := stmt.Exec(repo.db)
	if err != nil {
		return fmt.Errorf("error deleting notification: %w", err)
	}

	slog.Info("vehicle notification deleted", "user_id", user.UserID, "notification_id", notificationID)
	return nil
}

func (repo *RepositoryImpl) DeleteAllByVehicle(user *bootstrap.User, vehicleID string) error {
	if user == nil {
		return errors.New("valid user not found")
	}

	stmt := UserVehicleNotifications.DELETE().
		WHERE(UserVehicleNotifications.UserID.EQ(String(user.UserID)).
			AND(UserVehicleNotifications.VehicleID.EQ(String(vehicleID))))

	_, err := stmt.Exec(repo.db)
	if err != nil {
		return fmt.Errorf("error deleting notifications for vehicle: %w", err)
	}

	slog.Info("all vehicle notifications deleted", "user_id", user.UserID, "vehicle_id", vehicleID)
	return nil
}
