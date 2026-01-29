package notifications

import (
	"database/sql"
	"errors"
	"fmt"
	"miltechserver/.gen/miltech_ng/public/model"
	. "miltechserver/.gen/miltech_ng/public/table"
	"miltechserver/api/response"
	"miltechserver/bootstrap"

	. "github.com/go-jet/jet/v2/postgres"
)

type RepositoryImpl struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *RepositoryImpl {
	return &RepositoryImpl{db: db}
}

func (repo *RepositoryImpl) CreateVehicleNotification(user *bootstrap.User, notification model.ShopVehicleNotifications) (*model.ShopVehicleNotifications, error) {
	stmt := ShopVehicleNotifications.INSERT(
		ShopVehicleNotifications.ID,
		ShopVehicleNotifications.ShopID,
		ShopVehicleNotifications.VehicleID,
		ShopVehicleNotifications.Title,
		ShopVehicleNotifications.Description,
		ShopVehicleNotifications.Type,
		ShopVehicleNotifications.Completed,
		ShopVehicleNotifications.SaveTime,
		ShopVehicleNotifications.LastUpdated,
	).MODEL(notification).RETURNING(ShopVehicleNotifications.AllColumns)

	var createdNotification model.ShopVehicleNotifications
	err := stmt.Query(repo.db, &createdNotification)
	if err != nil {
		return nil, fmt.Errorf("failed to create vehicle notification: %w", err)
	}

	return &createdNotification, nil
}

func (repo *RepositoryImpl) GetVehicleNotifications(user *bootstrap.User, vehicleID string) ([]model.ShopVehicleNotifications, error) {
	stmt := SELECT(ShopVehicleNotifications.AllColumns).
		FROM(ShopVehicleNotifications).
		WHERE(ShopVehicleNotifications.VehicleID.EQ(String(vehicleID))).
		ORDER_BY(ShopVehicleNotifications.SaveTime.DESC())

	var notifications []model.ShopVehicleNotifications
	err := stmt.Query(repo.db, &notifications)
	if err != nil {
		return nil, fmt.Errorf("failed to get vehicle notifications: %w", err)
	}

	return notifications, nil
}

func (repo *RepositoryImpl) GetVehicleNotificationsWithItems(user *bootstrap.User, vehicleID string) ([]response.VehicleNotificationWithItems, error) {
	notifications, err := repo.GetVehicleNotifications(user, vehicleID)
	if err != nil {
		return nil, fmt.Errorf("failed to get vehicle notifications: %w", err)
	}

	var result []response.VehicleNotificationWithItems
	for _, notification := range notifications {
		items, err := repo.GetNotificationItems(user, notification.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get items for notification %s: %w", notification.ID, err)
		}

		if items == nil {
			items = []model.ShopNotificationItems{}
		}

		result = append(result, response.VehicleNotificationWithItems{
			Notification: notification,
			Items:        items,
		})
	}

	return result, nil
}

func (repo *RepositoryImpl) GetShopNotifications(user *bootstrap.User, shopID string) ([]model.ShopVehicleNotifications, error) {
	stmt := SELECT(ShopVehicleNotifications.AllColumns).
		FROM(ShopVehicleNotifications).
		WHERE(ShopVehicleNotifications.ShopID.EQ(String(shopID))).
		ORDER_BY(ShopVehicleNotifications.SaveTime.DESC())

	var notifications []model.ShopVehicleNotifications
	err := stmt.Query(repo.db, &notifications)
	if err != nil {
		return nil, fmt.Errorf("failed to get shop notifications: %w", err)
	}

	return notifications, nil
}

func (repo *RepositoryImpl) GetVehicleNotificationByID(user *bootstrap.User, notificationID string) (*model.ShopVehicleNotifications, error) {
	stmt := SELECT(ShopVehicleNotifications.AllColumns).
		FROM(ShopVehicleNotifications).
		WHERE(ShopVehicleNotifications.ID.EQ(String(notificationID)))

	var notification model.ShopVehicleNotifications
	err := stmt.Query(repo.db, &notification)
	if err != nil {
		return nil, fmt.Errorf("vehicle notification not found: %w", err)
	}

	return &notification, nil
}

func (repo *RepositoryImpl) UpdateVehicleNotification(user *bootstrap.User, notification model.ShopVehicleNotifications) error {
	stmt := ShopVehicleNotifications.UPDATE(
		ShopVehicleNotifications.Title,
		ShopVehicleNotifications.Description,
		ShopVehicleNotifications.Type,
		ShopVehicleNotifications.Completed,
		ShopVehicleNotifications.LastUpdated,
	).SET(
		ShopVehicleNotifications.Title.SET(String(notification.Title)),
		ShopVehicleNotifications.Description.SET(String(notification.Description)),
		ShopVehicleNotifications.Type.SET(String(notification.Type)),
		ShopVehicleNotifications.Completed.SET(Bool(notification.Completed)),
		ShopVehicleNotifications.LastUpdated.SET(TimestampzT(notification.LastUpdated)),
	).WHERE(ShopVehicleNotifications.ID.EQ(String(notification.ID)))

	result, err := stmt.Exec(repo.db)
	if err != nil {
		return fmt.Errorf("failed to update vehicle notification: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return errors.New("notification not found")
	}

	return nil
}

func (repo *RepositoryImpl) DeleteVehicleNotification(user *bootstrap.User, notificationID string) error {
	stmt := ShopVehicleNotifications.DELETE().
		WHERE(ShopVehicleNotifications.ID.EQ(String(notificationID)))

	result, err := stmt.Exec(repo.db)
	if err != nil {
		return fmt.Errorf("failed to delete vehicle notification: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return errors.New("notification not found")
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

func (repo *RepositoryImpl) IsUserMemberOfShop(user *bootstrap.User, shopID string) (bool, error) {
	stmt := SELECT(COUNT(ShopMembers.ID).AS("count")).
		FROM(ShopMembers).
		WHERE(
			ShopMembers.ShopID.EQ(String(shopID)).
				AND(ShopMembers.UserID.EQ(String(user.UserID))),
		)

	var result struct {
		Count int64 `sql:"primary_key"`
	}
	err := stmt.Query(repo.db, &result)
	if err != nil {
		return false, fmt.Errorf("failed to check membership: %w", err)
	}

	return result.Count > 0, nil
}

func (repo *RepositoryImpl) GetNotificationItems(user *bootstrap.User, notificationID string) ([]model.ShopNotificationItems, error) {
	stmt := SELECT(ShopNotificationItems.AllColumns).
		FROM(ShopNotificationItems).
		WHERE(ShopNotificationItems.NotificationID.EQ(String(notificationID))).
		ORDER_BY(ShopNotificationItems.SaveTime.ASC())

	var items []model.ShopNotificationItems
	err := stmt.Query(repo.db, &items)
	if err != nil {
		return nil, fmt.Errorf("failed to get notification items: %w", err)
	}

	return items, nil
}
