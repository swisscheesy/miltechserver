package notifications

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/go-jet/jet/v2/qrm"
	"github.com/lib/pq"
	"miltechserver/.gen/miltech_ng/public/model"
	. "miltechserver/.gen/miltech_ng/public/table"
	"miltechserver/api/response"
	sharedb "miltechserver/api/shared/db"
	"miltechserver/api/shops/shared"
	"miltechserver/bootstrap"

	. "github.com/go-jet/jet/v2/postgres"
)

type RepositoryImpl struct {
	db *sql.DB
}

var errShopListNotFound = errors.New("shop list not found")

func NewRepository(db *sql.DB) *RepositoryImpl {
	return &RepositoryImpl{db: db}
}

func (repo *RepositoryImpl) CreateVehicleNotification(ctx context.Context, user *bootstrap.User, notification model.ShopVehicleNotifications) (*model.ShopVehicleNotifications, error) {
	return repo.createLegacyNotification(ctx, user, notification)
}

func (repo *RepositoryImpl) GetVehicleNotifications(ctx context.Context, user *bootstrap.User, vehicleID string) ([]model.ShopVehicleNotifications, error) {
	return getVehicleNotifications(ctx, repo.db, vehicleID)
}

func getVehicleNotifications(ctx context.Context, db qrm.Queryable, vehicleID string) ([]model.ShopVehicleNotifications, error) {
	stmt := SELECT(ShopVehicleNotifications.AllColumns).
		FROM(ShopVehicleNotifications).
		WHERE(ShopVehicleNotifications.VehicleID.EQ(String(vehicleID))).
		ORDER_BY(ShopVehicleNotifications.SaveTime.DESC(), ShopVehicleNotifications.ID.ASC())

	var notifications []model.ShopVehicleNotifications
	err := stmt.QueryContext(ctx, db, &notifications)
	if err != nil {
		return nil, fmt.Errorf("failed to get vehicle notifications: %w", err)
	}

	return notifications, nil
}

func (repo *RepositoryImpl) GetVehicleNotificationsWithItems(ctx context.Context, user *bootstrap.User, vehicleID string) ([]response.VehicleNotificationWithItems, error) {
	if user == nil {
		return nil, errors.New("unauthorized user")
	}
	var result []response.VehicleNotificationWithItems
	err := sharedb.WithTxOptions(ctx, repo.db, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true}, func(tx *sql.Tx) error {
		var vehicle model.ShopVehicle
		if err := SELECT(ShopVehicle.AllColumns).FROM(ShopVehicle).WHERE(ShopVehicle.ID.EQ(String(vehicleID))).QueryContext(ctx, tx, &vehicle); err != nil {
			return fmt.Errorf("failed to get vehicle: %w", err)
		}
		var member bool
		if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM shop_members WHERE shop_id=$1 AND user_id=$2)`, vehicle.ShopID, user.UserID).Scan(&member); err != nil {
			return err
		}
		if !member {
			return errors.New("access denied: user is not a member of this shop")
		}
		var err error
		result, err = getVehicleNotificationsWithItems(ctx, tx, vehicleID)
		return err
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func getVehicleNotificationsWithItems(ctx context.Context, db qrm.Queryable, vehicleID string) ([]response.VehicleNotificationWithItems, error) {
	notifications, err := getVehicleNotifications(ctx, db, vehicleID)
	if err != nil {
		return nil, fmt.Errorf("failed to get vehicle notifications: %w", err)
	}

	if len(notifications) == 0 {
		return []response.VehicleNotificationWithItems{}, nil
	}

	notificationIDs := make([]string, len(notifications))
	for i, notification := range notifications {
		notificationIDs[i] = notification.ID
	}

	allItems, err := getItemsByNotificationIDs(ctx, db, notificationIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to get notification items: %w", err)
	}

	itemsByNotification := make(map[string][]model.ShopNotificationItems, len(notificationIDs))
	for _, item := range allItems {
		itemsByNotification[item.NotificationID] = append(itemsByNotification[item.NotificationID], item)
	}

	result := make([]response.VehicleNotificationWithItems, len(notifications))
	for i, notification := range notifications {
		items := itemsByNotification[notification.ID]
		if items == nil {
			items = []model.ShopNotificationItems{}
		}

		result[i] = response.VehicleNotificationWithItems{
			Notification: notification,
			Items:        items,
		}
	}

	return result, nil
}

func (repo *RepositoryImpl) GetItemsByNotificationIDs(ctx context.Context, notificationIDs []string) ([]model.ShopNotificationItems, error) {
	return getItemsByNotificationIDs(ctx, repo.db, notificationIDs)
}
func getItemsByNotificationIDs(ctx context.Context, db qrm.Queryable, notificationIDs []string) ([]model.ShopNotificationItems, error) {
	if len(notificationIDs) == 0 {
		return []model.ShopNotificationItems{}, nil
	}

	stmt := SELECT(ShopNotificationItems.AllColumns).
		FROM(ShopNotificationItems).
		WHERE(RawBool("shop_notification_items.notification_id = ANY(:ids::text[])", RawArgs{":ids": pq.Array(notificationIDs)})).
		ORDER_BY(ShopNotificationItems.SaveTime.ASC(), ShopNotificationItems.ID.ASC())

	var items []model.ShopNotificationItems
	err := stmt.QueryContext(ctx, db, &items)
	if err != nil {
		return nil, err
	}

	return items, nil
}

func (repo *RepositoryImpl) GetShopNotifications(ctx context.Context, user *bootstrap.User, shopID string) ([]model.ShopVehicleNotifications, error) {
	stmt := SELECT(ShopVehicleNotifications.AllColumns).
		FROM(ShopVehicleNotifications).
		WHERE(ShopVehicleNotifications.ShopID.EQ(String(shopID))).
		ORDER_BY(ShopVehicleNotifications.SaveTime.DESC())

	var notifications []model.ShopVehicleNotifications
	err := stmt.QueryContext(ctx, repo.db, &notifications)
	if err != nil {
		return nil, fmt.Errorf("failed to get shop notifications: %w", err)
	}

	return notifications, nil
}

func (repo *RepositoryImpl) GetVehicleNotificationByID(ctx context.Context, user *bootstrap.User, notificationID string) (*model.ShopVehicleNotifications, error) {
	stmt := SELECT(ShopVehicleNotifications.AllColumns).
		FROM(ShopVehicleNotifications).
		WHERE(ShopVehicleNotifications.ID.EQ(String(notificationID)))

	var notification model.ShopVehicleNotifications
	err := stmt.QueryContext(ctx, repo.db, &notification)
	if err != nil {
		if shared.ErrorsIsNoRows(err) {
			return nil, shared.ErrNotificationNotFound
		}
		return nil, fmt.Errorf("vehicle notification lookup failed: %w", err)
	}

	return &notification, nil
}

func (repo *RepositoryImpl) UpdateVehicleNotification(ctx context.Context, user *bootstrap.User, update VehicleNotificationUpdate) error {
	return repo.updateLegacyNotification(ctx, user, update)
}

func (repo *RepositoryImpl) DeleteVehicleNotification(ctx context.Context, user *bootstrap.User, notificationID string) error {
	return repo.deleteLegacyNotification(ctx, user, notificationID)
}

func (repo *RepositoryImpl) CreateNotificationChange(ctx context.Context, user *bootstrap.User, change model.ShopVehicleNotificationChanges) error {
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

	_, err := repo.db.ExecContext(
		ctx, rawSQL,
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

func (repo *RepositoryImpl) GetShopVehicleByID(ctx context.Context, user *bootstrap.User, vehicleID string) (*model.ShopVehicle, error) {
	stmt := SELECT(ShopVehicle.AllColumns).
		FROM(ShopVehicle).
		WHERE(ShopVehicle.ID.EQ(String(vehicleID)))

	var vehicle model.ShopVehicle
	err := stmt.QueryContext(ctx, repo.db, &vehicle)
	if err != nil {
		return nil, fmt.Errorf("shop vehicle not found: %w", err)
	}

	return &vehicle, nil
}

func (repo *RepositoryImpl) GetShopListByID(ctx context.Context, user *bootstrap.User, listID string) (*model.ShopLists, error) {
	stmt := SELECT(ShopLists.AllColumns).
		FROM(ShopLists).
		WHERE(ShopLists.ID.EQ(String(listID)))

	var lists []model.ShopLists
	err := stmt.QueryContext(ctx, repo.db, &lists)
	if err != nil {
		return nil, fmt.Errorf("failed to get shop list: %w", err)
	}

	if len(lists) == 0 {
		return nil, errShopListNotFound
	}

	return &lists[0], nil
}

func (repo *RepositoryImpl) IsUserMemberOfShop(ctx context.Context, user *bootstrap.User, shopID string) (bool, error) {
	stmt := SELECT(Int(1).AS("exists")).
		FROM(ShopMembers).
		WHERE(
			ShopMembers.ShopID.EQ(String(shopID)).
				AND(ShopMembers.UserID.EQ(String(user.UserID))),
		).
		LIMIT(1)

	var result []struct {
		Exists int `sql:"exists"`
	}
	err := stmt.QueryContext(ctx, repo.db, &result)
	if err != nil {
		return false, fmt.Errorf("failed to check membership: %w", err)
	}

	return len(result) > 0, nil
}

func (repo *RepositoryImpl) GetNotificationItems(ctx context.Context, user *bootstrap.User, notificationID string) ([]model.ShopNotificationItems, error) {
	stmt := SELECT(ShopNotificationItems.AllColumns).
		FROM(ShopNotificationItems).
		WHERE(ShopNotificationItems.NotificationID.EQ(String(notificationID))).
		ORDER_BY(ShopNotificationItems.SaveTime.ASC())

	var items []model.ShopNotificationItems
	err := stmt.QueryContext(ctx, repo.db, &items)
	if err != nil {
		return nil, fmt.Errorf("failed to get notification items: %w", err)
	}

	return items, nil
}
