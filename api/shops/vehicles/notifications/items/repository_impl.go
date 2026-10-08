package items

import (
	"context"
	"database/sql"
	"fmt"
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

func (repo *RepositoryImpl) CreateNotificationItem(ctx context.Context, user *bootstrap.User, item model.ShopNotificationItems) (*model.ShopNotificationItems, error) {
	created, err := repo.createLegacyItems(ctx, user, []model.ShopNotificationItems{item})
	if err != nil {
		return nil, err
	}
	return &created[0], nil
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

func (repo *RepositoryImpl) GetShopNotificationItems(ctx context.Context, user *bootstrap.User, shopID string) ([]model.ShopNotificationItems, error) {
	stmt := SELECT(ShopNotificationItems.AllColumns).
		FROM(ShopNotificationItems).
		WHERE(ShopNotificationItems.ShopID.EQ(String(shopID))).
		ORDER_BY(ShopNotificationItems.SaveTime.DESC())

	var items []model.ShopNotificationItems
	err := stmt.QueryContext(ctx, repo.db, &items)
	if err != nil {
		return nil, fmt.Errorf("failed to get shop notification items: %w", err)
	}

	return items, nil
}

func (repo *RepositoryImpl) GetNotificationItemByID(ctx context.Context, user *bootstrap.User, itemID string) (*model.ShopNotificationItems, error) {
	stmt := SELECT(ShopNotificationItems.AllColumns).
		FROM(ShopNotificationItems).
		WHERE(ShopNotificationItems.ID.EQ(String(itemID)))

	var item model.ShopNotificationItems
	err := stmt.QueryContext(ctx, repo.db, &item)
	if err != nil {
		return nil, fmt.Errorf("failed to get notification item: %w", err)
	}

	return &item, nil
}

func (repo *RepositoryImpl) GetNotificationItemsByIDs(ctx context.Context, user *bootstrap.User, itemIDs []string) ([]model.ShopNotificationItems, error) {
	if len(itemIDs) == 0 {
		return []model.ShopNotificationItems{}, nil
	}

	var expressions []Expression
	for _, id := range itemIDs {
		expressions = append(expressions, String(id))
	}

	stmt := SELECT(ShopNotificationItems.AllColumns).
		FROM(ShopNotificationItems).
		WHERE(ShopNotificationItems.ID.IN(expressions...))

	var items []model.ShopNotificationItems
	err := stmt.QueryContext(ctx, repo.db, &items)
	if err != nil {
		return nil, fmt.Errorf("failed to get notification items: %w", err)
	}

	return items, nil
}

func (repo *RepositoryImpl) CreateNotificationItemList(ctx context.Context, user *bootstrap.User, items []model.ShopNotificationItems) ([]model.ShopNotificationItems, error) {
	return repo.createLegacyItems(ctx, user, items)
}

func (repo *RepositoryImpl) DeleteNotificationItem(ctx context.Context, user *bootstrap.User, itemID string) error {
	_, err := repo.deleteLegacyItems(ctx, user, []string{itemID}, true)
	return err
}

func (repo *RepositoryImpl) DeleteNotificationItemList(ctx context.Context, user *bootstrap.User, itemIDs []string) (int64, error) {
	return repo.deleteLegacyItems(ctx, user, itemIDs, false)
}

func (repo *RepositoryImpl) GetVehicleNotificationByID(ctx context.Context, user *bootstrap.User, notificationID string) (*model.ShopVehicleNotifications, error) {
	stmt := SELECT(ShopVehicleNotifications.AllColumns).
		FROM(ShopVehicleNotifications).
		WHERE(ShopVehicleNotifications.ID.EQ(String(notificationID)))

	var notification model.ShopVehicleNotifications
	err := stmt.QueryContext(ctx, repo.db, &notification)
	if err != nil {
		return nil, fmt.Errorf("vehicle notification not found: %w", err)
	}

	return &notification, nil
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

func (repo *RepositoryImpl) IsUserMemberOfShop(ctx context.Context, user *bootstrap.User, shopID string) (bool, error) {
	stmt := SELECT(COUNT(ShopMembers.ID).AS("count")).
		FROM(ShopMembers).
		WHERE(
			ShopMembers.ShopID.EQ(String(shopID)).
				AND(ShopMembers.UserID.EQ(String(user.UserID))),
		)

	var result struct {
		Count int64 `sql:"primary_key"`
	}
	err := stmt.QueryContext(ctx, repo.db, &result)
	if err != nil {
		return false, fmt.Errorf("failed to check membership: %w", err)
	}

	return result.Count > 0, nil
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

	_, err := repo.db.ExecContext(ctx,
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
