package repository

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

type UserVehicleRepositoryImpl struct {
	Db *sql.DB
}

func NewUserVehicleRepositoryImpl(db *sql.DB) *UserVehicleRepositoryImpl {
	return &UserVehicleRepositoryImpl{Db: db}
}

// User Vehicle Operations

func (repo *UserVehicleRepositoryImpl) GetUserVehiclesByUserId(user *bootstrap.User) ([]model.UserVehicle, error) {
	var vehicles []model.UserVehicle

	if user == nil {
		return nil, errors.New("valid user not found")
	}

	stmt := SELECT(UserVehicle.AllColumns).
		FROM(UserVehicle).
		WHERE(UserVehicle.UserID.EQ(String(user.UserID)))

	err := stmt.Query(repo.Db, &vehicles)
	if err != nil {
		return nil, fmt.Errorf("error retrieving vehicles for user %s: %w", user.UserID, err)
	}

	slog.Info("vehicles retrieved for user", "user_id", user.UserID, "count", len(vehicles))
	return vehicles, nil
}

func (repo *UserVehicleRepositoryImpl) GetUserVehicleById(user *bootstrap.User, vehicleId string) (*model.UserVehicle, error) {
	var vehicle model.UserVehicle

	if user == nil {
		return nil, errors.New("valid user not found")
	}

	stmt := SELECT(UserVehicle.AllColumns).
		FROM(UserVehicle).
		WHERE(UserVehicle.UserID.EQ(String(user.UserID)).
			AND(UserVehicle.ID.EQ(String(vehicleId))))

	err := stmt.Query(repo.Db, &vehicle)
	if err != nil {
		return nil, fmt.Errorf("vehicle not found for user %s: %w", user.UserID, err)
	}

	slog.Info("vehicle retrieved", "user_id", user.UserID, "vehicle_id", vehicleId)
	return &vehicle, nil
}

func (repo *UserVehicleRepositoryImpl) UpsertUserVehicle(user *bootstrap.User, vehicle model.UserVehicle) error {
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

	err := stmt.Query(repo.Db, &vehicle)
	if err != nil {
		return fmt.Errorf("error saving vehicle: %w", err)
	}

	slog.Info("vehicle saved", "user_id", user.UserID, "vehicle_id", vehicle.ID)
	return nil
}

func (repo *UserVehicleRepositoryImpl) DeleteUserVehicle(user *bootstrap.User, vehicleId string) error {
	if user == nil {
		return errors.New("valid user not found")
	}

	stmt := UserVehicle.DELETE().
		WHERE(UserVehicle.UserID.EQ(String(user.UserID)).
			AND(UserVehicle.ID.EQ(String(vehicleId))))

	_, err := stmt.Exec(repo.Db)
	if err != nil {
		return fmt.Errorf("error deleting vehicle: %w", err)
	}

	slog.Info("vehicle deleted", "user_id", user.UserID, "vehicle_id", vehicleId)
	return nil
}

func (repo *UserVehicleRepositoryImpl) DeleteAllUserVehiclesByUserId(user *bootstrap.User) error {
	if user == nil {
		return errors.New("valid user not found")
	}

	stmt := UserVehicle.DELETE().
		WHERE(UserVehicle.UserID.EQ(String(user.UserID)))

	_, err := stmt.Exec(repo.Db)
	if err != nil {
		return fmt.Errorf("error deleting all vehicles: %w", err)
	}

	return nil
}

func (repo *UserVehicleRepositoryImpl) DeleteAllUserVehicles(user *bootstrap.User) error {
	if user == nil {
		return errors.New("valid user not found")
	}

	stmt := UserVehicle.DELETE().
		WHERE(UserVehicle.UserID.EQ(String(user.UserID)))

	_, err := stmt.Exec(repo.Db)
	if err != nil {
		return fmt.Errorf("error deleting all vehicles: %w", err)
	}

	slog.Info("all vehicles deleted", "user_id", user.UserID)
	return nil
}

// User Vehicle Notifications Operations

func (repo *UserVehicleRepositoryImpl) GetVehicleNotificationsByUserId(user *bootstrap.User) ([]model.UserVehicleNotifications, error) {
	var notifications []model.UserVehicleNotifications

	if user == nil {
		return nil, errors.New("valid user not found")
	}

	stmt := SELECT(UserVehicleNotifications.AllColumns).
		FROM(UserVehicleNotifications).
		WHERE(UserVehicleNotifications.UserID.EQ(String(user.UserID)))

	err := stmt.Query(repo.Db, &notifications)
	if err != nil {
		return nil, fmt.Errorf("error retrieving notifications for user %s: %w", user.UserID, err)
	}

	slog.Info("vehicle notifications retrieved for user", "user_id", user.UserID, "count", len(notifications))
	return notifications, nil
}

func (repo *UserVehicleRepositoryImpl) GetVehicleNotificationsByVehicleId(user *bootstrap.User, vehicleId string) ([]model.UserVehicleNotifications, error) {
	var notifications []model.UserVehicleNotifications

	if user == nil {
		return nil, errors.New("valid user not found")
	}

	stmt := SELECT(UserVehicleNotifications.AllColumns).
		FROM(UserVehicleNotifications).
		WHERE(UserVehicleNotifications.UserID.EQ(String(user.UserID)).
			AND(UserVehicleNotifications.VehicleID.EQ(String(vehicleId))))

	err := stmt.Query(repo.Db, &notifications)
	if err != nil {
		return nil, fmt.Errorf("error retrieving notifications for vehicle %s: %w", vehicleId, err)
	}

	slog.Info("vehicle notifications retrieved", "user_id", user.UserID, "vehicle_id", vehicleId, "count", len(notifications))
	return notifications, nil
}

func (repo *UserVehicleRepositoryImpl) GetVehicleNotificationById(user *bootstrap.User, notificationId string) (*model.UserVehicleNotifications, error) {
	var notification model.UserVehicleNotifications

	if user == nil {
		return nil, errors.New("valid user not found")
	}

	stmt := SELECT(UserVehicleNotifications.AllColumns).
		FROM(UserVehicleNotifications).
		WHERE(UserVehicleNotifications.UserID.EQ(String(user.UserID)).
			AND(UserVehicleNotifications.ID.EQ(String(notificationId))))

	err := stmt.Query(repo.Db, &notification)
	if err != nil {
		return nil, fmt.Errorf("notification not found for user %s: %w", user.UserID, err)
	}

	slog.Info("vehicle notification retrieved", "user_id", user.UserID, "notification_id", notificationId)
	return &notification, nil
}

func (repo *UserVehicleRepositoryImpl) UpsertVehicleNotification(user *bootstrap.User, notification model.UserVehicleNotifications) error {
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

	err := stmt.Query(repo.Db, &notification)
	if err != nil {
		return fmt.Errorf("error saving notification: %w", err)
	}

	slog.Info("vehicle notification saved", "user_id", user.UserID, "notification_id", notification.ID)
	return nil
}

func (repo *UserVehicleRepositoryImpl) DeleteVehicleNotification(user *bootstrap.User, notificationId string) error {
	if user == nil {
		return errors.New("valid user not found")
	}

	stmt := UserVehicleNotifications.DELETE().
		WHERE(UserVehicleNotifications.UserID.EQ(String(user.UserID)).
			AND(UserVehicleNotifications.ID.EQ(String(notificationId))))

	_, err := stmt.Exec(repo.Db)
	if err != nil {
		return fmt.Errorf("error deleting notification: %w", err)
	}

	slog.Info("vehicle notification deleted", "user_id", user.UserID, "notification_id", notificationId)
	return nil
}

func (repo *UserVehicleRepositoryImpl) DeleteAllVehicleNotificationsByVehicle(user *bootstrap.User, vehicleId string) error {
	if user == nil {
		return errors.New("valid user not found")
	}

	stmt := UserVehicleNotifications.DELETE().
		WHERE(UserVehicleNotifications.UserID.EQ(String(user.UserID)).
			AND(UserVehicleNotifications.VehicleID.EQ(String(vehicleId))))

	_, err := stmt.Exec(repo.Db)
	if err != nil {
		return fmt.Errorf("error deleting notifications for vehicle: %w", err)
	}

	slog.Info("all vehicle notifications deleted", "user_id", user.UserID, "vehicle_id", vehicleId)
	return nil
}

// User Vehicle Comments Operations

func (repo *UserVehicleRepositoryImpl) GetVehicleCommentsByUserId(user *bootstrap.User) ([]model.UserVehicleComments, error) {
	var comments []model.UserVehicleComments

	if user == nil {
		return nil, errors.New("valid user not found")
	}

	stmt := SELECT(UserVehicleComments.AllColumns).
		FROM(UserVehicleComments).
		WHERE(UserVehicleComments.UserID.EQ(String(user.UserID)))

	err := stmt.Query(repo.Db, &comments)
	if err != nil {
		return nil, fmt.Errorf("error retrieving comments for user %s: %w", user.UserID, err)
	}

	slog.Info("vehicle comments retrieved for user", "user_id", user.UserID, "count", len(comments))
	return comments, nil
}

func (repo *UserVehicleRepositoryImpl) GetVehicleCommentsByVehicleId(user *bootstrap.User, vehicleId string) ([]model.UserVehicleComments, error) {
	var comments []model.UserVehicleComments

	if user == nil {
		return nil, errors.New("valid user not found")
	}

	stmt := SELECT(UserVehicleComments.AllColumns).
		FROM(UserVehicleComments).
		WHERE(UserVehicleComments.UserID.EQ(String(user.UserID)).
			AND(UserVehicleComments.VehicleID.EQ(String(vehicleId))))

	err := stmt.Query(repo.Db, &comments)
	if err != nil {
		return nil, fmt.Errorf("error retrieving comments for vehicle %s: %w", vehicleId, err)
	}

	slog.Info("vehicle comments retrieved", "user_id", user.UserID, "vehicle_id", vehicleId, "count", len(comments))
	return comments, nil
}

func (repo *UserVehicleRepositoryImpl) GetVehicleCommentsByNotificationId(user *bootstrap.User, notificationId string) ([]model.UserVehicleComments, error) {
	var comments []model.UserVehicleComments

	if user == nil {
		return nil, errors.New("valid user not found")
	}

	stmt := SELECT(UserVehicleComments.AllColumns).
		FROM(UserVehicleComments).
		WHERE(UserVehicleComments.UserID.EQ(String(user.UserID)).
			AND(UserVehicleComments.NotificationID.EQ(String(notificationId))))

	err := stmt.Query(repo.Db, &comments)
	if err != nil {
		return nil, fmt.Errorf("error retrieving comments for notification %s: %w", notificationId, err)
	}

	slog.Info("vehicle comments retrieved", "user_id", user.UserID, "notification_id", notificationId, "count", len(comments))
	return comments, nil
}

func (repo *UserVehicleRepositoryImpl) GetVehicleCommentById(user *bootstrap.User, commentId string) (*model.UserVehicleComments, error) {
	var comment model.UserVehicleComments

	if user == nil {
		return nil, errors.New("valid user not found")
	}

	stmt := SELECT(UserVehicleComments.AllColumns).
		FROM(UserVehicleComments).
		WHERE(UserVehicleComments.UserID.EQ(String(user.UserID)).
			AND(UserVehicleComments.ID.EQ(String(commentId))))

	err := stmt.Query(repo.Db, &comment)
	if err != nil {
		return nil, fmt.Errorf("comment not found for user %s: %w", user.UserID, err)
	}

	slog.Info("vehicle comment retrieved", "user_id", user.UserID, "comment_id", commentId)
	return &comment, nil
}

func (repo *UserVehicleRepositoryImpl) UpsertVehicleComment(user *bootstrap.User, comment model.UserVehicleComments) error {
	if user == nil {
		return errors.New("valid user not found")
	}

	stmt := UserVehicleComments.INSERT(
		UserVehicleComments.ID, UserVehicleComments.UserID, UserVehicleComments.VehicleID,
		UserVehicleComments.NotificationID, UserVehicleComments.ParentID, UserVehicleComments.Message,
		UserVehicleComments.PostTime).
		MODEL(comment).
		ON_CONFLICT(UserVehicleComments.ID, UserVehicleComments.UserID).
		DO_UPDATE(
			SET(
				UserVehicleComments.VehicleID.SET(String(comment.VehicleID)),
				UserVehicleComments.Message.SET(String(comment.Message))).
				WHERE(UserVehicleComments.UserID.EQ(String(user.UserID)).
					AND(UserVehicleComments.ID.EQ(String(comment.ID))))).
		RETURNING(UserVehicleComments.AllColumns)

	err := stmt.Query(repo.Db, &comment)
	if err != nil {
		return fmt.Errorf("error saving comment: %w", err)
	}

	slog.Info("vehicle comment saved", "user_id", user.UserID, "comment_id", comment.ID)
	return nil
}

func (repo *UserVehicleRepositoryImpl) DeleteVehicleComment(user *bootstrap.User, commentId string) error {
	if user == nil {
		return errors.New("valid user not found")
	}

	stmt := UserVehicleComments.DELETE().
		WHERE(UserVehicleComments.UserID.EQ(String(user.UserID)).
			AND(UserVehicleComments.ID.EQ(String(commentId))))

	_, err := stmt.Exec(repo.Db)
	if err != nil {
		return fmt.Errorf("error deleting comment: %w", err)
	}

	slog.Info("vehicle comment deleted", "user_id", user.UserID, "comment_id", commentId)
	return nil
}

func (repo *UserVehicleRepositoryImpl) DeleteAllVehicleCommentsByVehicle(user *bootstrap.User, vehicleId string) error {
	if user == nil {
		return errors.New("valid user not found")
	}

	stmt := UserVehicleComments.DELETE().
		WHERE(UserVehicleComments.UserID.EQ(String(user.UserID)).
			AND(UserVehicleComments.VehicleID.EQ(String(vehicleId))))

	_, err := stmt.Exec(repo.Db)
	if err != nil {
		return fmt.Errorf("error deleting comments for vehicle: %w", err)
	}

	slog.Info("all vehicle comments deleted", "user_id", user.UserID, "vehicle_id", vehicleId)
	return nil
}

// User Notification Items Operations

func (repo *UserVehicleRepositoryImpl) GetNotificationItemsByUserId(user *bootstrap.User) ([]model.UserNotificationItems, error) {
	var items []model.UserNotificationItems

	if user == nil {
		return nil, errors.New("valid user not found")
	}

	stmt := SELECT(UserNotificationItems.AllColumns).
		FROM(UserNotificationItems).
		WHERE(UserNotificationItems.UserID.EQ(String(user.UserID)))

	err := stmt.Query(repo.Db, &items)
	if err != nil {
		return nil, fmt.Errorf("error retrieving notification items for user %s: %w", user.UserID, err)
	}

	slog.Info("notification items retrieved for user", "user_id", user.UserID, "count", len(items))
	return items, nil
}

func (repo *UserVehicleRepositoryImpl) GetNotificationItemsByNotificationId(user *bootstrap.User, notificationId string) ([]model.UserNotificationItems, error) {
	var items []model.UserNotificationItems

	if user == nil {
		return nil, errors.New("valid user not found")
	}

	stmt := SELECT(UserNotificationItems.AllColumns).
		FROM(UserNotificationItems).
		WHERE(UserNotificationItems.UserID.EQ(String(user.UserID)).
			AND(UserNotificationItems.NotificationID.EQ(String(notificationId))))

	err := stmt.Query(repo.Db, &items)
	if err != nil {
		return nil, fmt.Errorf("error retrieving items for notification %s: %w", notificationId, err)
	}

	slog.Info("notification items retrieved", "user_id", user.UserID, "notification_id", notificationId, "count", len(items))
	return items, nil
}

func (repo *UserVehicleRepositoryImpl) GetNotificationItemById(user *bootstrap.User, itemId string) (*model.UserNotificationItems, error) {
	var item model.UserNotificationItems

	if user == nil {
		return nil, errors.New("valid user not found")
	}

	stmt := SELECT(UserNotificationItems.AllColumns).
		FROM(UserNotificationItems).
		WHERE(UserNotificationItems.UserID.EQ(String(user.UserID)).
			AND(UserNotificationItems.ID.EQ(String(itemId))))

	err := stmt.Query(repo.Db, &item)
	if err != nil {
		return nil, fmt.Errorf("notification item not found for user %s: %w", user.UserID, err)
	}

	slog.Info("notification item retrieved", "user_id", user.UserID, "item_id", itemId)
	return &item, nil
}

func (repo *UserVehicleRepositoryImpl) UpsertNotificationItem(user *bootstrap.User, item model.UserNotificationItems) error {
	if user == nil {
		return errors.New("valid user not found")
	}

	stmt := UserNotificationItems.INSERT(
		UserNotificationItems.ID, UserNotificationItems.UserID, UserNotificationItems.NotificationID,
		UserNotificationItems.Niin, UserNotificationItems.Nomenclature, UserNotificationItems.Quantity,
		UserNotificationItems.SaveTime).
		MODEL(item).
		ON_CONFLICT(UserNotificationItems.ID).
		DO_UPDATE(
			SET(
				UserNotificationItems.NotificationID.SET(String(item.NotificationID)),
				UserNotificationItems.Niin.SET(String(item.Niin)),
				UserNotificationItems.Nomenclature.SET(String(item.Nomenclature)),
				UserNotificationItems.Quantity.SET(Int32(item.Quantity))).
				WHERE(UserNotificationItems.ID.EQ(String(item.ID)))).
		RETURNING(UserNotificationItems.AllColumns)

	err := stmt.Query(repo.Db, &item)
	if err != nil {
		return fmt.Errorf("error saving notification item: %w", err)
	}

	slog.Info("notification item saved", "user_id", user.UserID, "item_id", item.ID)
	return nil
}

func (repo *UserVehicleRepositoryImpl) UpsertNotificationItemList(user *bootstrap.User, items []model.UserNotificationItems) error {
	if user == nil {
		return errors.New("valid user not found")
	}

	var failedItems []string
	for _, item := range items {
		stmt := UserNotificationItems.INSERT(
			UserNotificationItems.ID, UserNotificationItems.UserID, UserNotificationItems.NotificationID,
			UserNotificationItems.Niin, UserNotificationItems.Nomenclature, UserNotificationItems.Quantity,
			UserNotificationItems.SaveTime).
			MODEL(item).
			ON_CONFLICT(UserNotificationItems.ID).
			DO_UPDATE(
				SET(
					UserNotificationItems.NotificationID.SET(String(item.NotificationID)),
					UserNotificationItems.Niin.SET(String(item.Niin)),
					UserNotificationItems.Nomenclature.SET(String(item.Nomenclature)),
					UserNotificationItems.Quantity.SET(Int32(item.Quantity))).
					WHERE(UserNotificationItems.ID.EQ(String(item.ID))))

		_, err := stmt.Exec(repo.Db)
		if err != nil {
			failedItems = append(failedItems, item.ID)
		}
	}

	if len(failedItems) > 0 {
		return fmt.Errorf("failed to save following notification items: %s", failedItems)
	}

	slog.Info("notification item list saved", "user_id", user.UserID, "count", len(items))
	return nil
}

func (repo *UserVehicleRepositoryImpl) DeleteNotificationItem(user *bootstrap.User, itemId string) error {
	if user == nil {
		return errors.New("valid user not found")
	}

	stmt := UserNotificationItems.DELETE().
		WHERE(UserNotificationItems.UserID.EQ(String(user.UserID)).
			AND(UserNotificationItems.ID.EQ(String(itemId))))

	_, err := stmt.Exec(repo.Db)
	if err != nil {
		return fmt.Errorf("error deleting notification item: %w", err)
	}

	slog.Info("notification item deleted", "user_id", user.UserID, "item_id", itemId)
	return nil
}

func (repo *UserVehicleRepositoryImpl) DeleteAllNotificationItemsByNotification(user *bootstrap.User, notificationId string) error {
	if user == nil {
		return errors.New("valid user not found")
	}

	stmt := UserNotificationItems.DELETE().
		WHERE(UserNotificationItems.UserID.EQ(String(user.UserID)).
			AND(UserNotificationItems.NotificationID.EQ(String(notificationId))))

	_, err := stmt.Exec(repo.Db)
	if err != nil {
		return fmt.Errorf("error deleting items for notification: %w", err)
	}

	slog.Info("all notification items deleted", "user_id", user.UserID, "notification_id", notificationId)
	return nil
}
