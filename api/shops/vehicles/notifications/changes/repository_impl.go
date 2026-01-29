package changes

import (
	"database/sql"
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

func (repo *RepositoryImpl) GetNotificationChanges(user *bootstrap.User, notificationID string) ([]response.NotificationChangeWithUsername, error) {
	rawSQL := `
		SELECT
			c.id,
			c.notification_id,
			c.shop_id,
			c.vehicle_id,
			c.changed_by,
			COALESCE(u.username, 'Unknown User') as changed_by_username,
			c.changed_at,
			c.change_type,
			c.field_changes,
			COALESCE(n.title, c.notification_title, 'Deleted Notification') as notification_title,
			c.notification_type,
			COALESCE(v.admin, c.vehicle_admin) as vehicle_admin,
			CASE WHEN c.notification_id IS NULL OR c.vehicle_id IS NULL THEN true ELSE false END as is_deleted
		FROM shop_vehicle_notification_changes c
		LEFT JOIN users u ON c.changed_by = u.uid
		LEFT JOIN shop_vehicle_notifications n ON c.notification_id = n.id
		LEFT JOIN shop_vehicle v ON c.vehicle_id = v.id
		WHERE c.notification_id = $1
		ORDER BY c.changed_at DESC
	`

	rows, err := repo.db.Query(rawSQL, notificationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get notification changes: %w", err)
	}
	defer rows.Close()

	var changes []response.NotificationChangeWithUsername
	for rows.Next() {
		var change response.NotificationChangeWithUsername
		var fieldChangesJSON string

		err := rows.Scan(
			&change.ID,
			&change.NotificationID,
			&change.ShopID,
			&change.VehicleID,
			&change.ChangedBy,
			&change.ChangedByUsername,
			&change.ChangedAt,
			&change.ChangeType,
			&fieldChangesJSON,
			&change.NotificationTitle,
			&change.NotificationType,
			&change.VehicleAdmin,
			&change.IsDeleted,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan change row: %w", err)
		}

		change.FieldChanges = make(map[string]interface{})
		if fieldChangesJSON != "" {
			change.FieldChanges["raw"] = fieldChangesJSON
		}

		changes = append(changes, change)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating change rows: %w", err)
	}

	return changes, nil
}

func (repo *RepositoryImpl) GetNotificationChangesByShop(user *bootstrap.User, shopID string, limit int) ([]response.NotificationChangeWithUsername, error) {
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}

	rawSQL := `
		SELECT
			c.id,
			c.notification_id,
			c.shop_id,
			c.vehicle_id,
			c.changed_by,
			COALESCE(u.username, 'Unknown User') as changed_by_username,
			c.changed_at,
			c.change_type,
			c.field_changes,
			COALESCE(n.title, c.notification_title, 'Deleted Notification') as notification_title,
			c.notification_type,
			COALESCE(v.admin, c.vehicle_admin) as vehicle_admin,
			CASE WHEN c.notification_id IS NULL OR c.vehicle_id IS NULL THEN true ELSE false END as is_deleted
		FROM shop_vehicle_notification_changes c
		LEFT JOIN users u ON c.changed_by = u.uid
		LEFT JOIN shop_vehicle_notifications n ON c.notification_id = n.id
		LEFT JOIN shop_vehicle v ON c.vehicle_id = v.id
		WHERE c.shop_id = $1
		ORDER BY c.changed_at DESC
		LIMIT $2
	`

	rows, err := repo.db.Query(rawSQL, shopID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get shop notification changes: %w", err)
	}
	defer rows.Close()

	var changes []response.NotificationChangeWithUsername
	for rows.Next() {
		var change response.NotificationChangeWithUsername
		var fieldChangesJSON string

		err := rows.Scan(
			&change.ID,
			&change.NotificationID,
			&change.ShopID,
			&change.VehicleID,
			&change.ChangedBy,
			&change.ChangedByUsername,
			&change.ChangedAt,
			&change.ChangeType,
			&fieldChangesJSON,
			&change.NotificationTitle,
			&change.NotificationType,
			&change.VehicleAdmin,
			&change.IsDeleted,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan change row: %w", err)
		}

		change.FieldChanges = make(map[string]interface{})
		if fieldChangesJSON != "" {
			change.FieldChanges["raw"] = fieldChangesJSON
		}

		changes = append(changes, change)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating change rows: %w", err)
	}

	return changes, nil
}

func (repo *RepositoryImpl) GetNotificationChangesByVehicle(user *bootstrap.User, vehicleID string) ([]response.NotificationChangeWithUsername, error) {
	rawSQL := `
		SELECT
			c.id,
			c.notification_id,
			c.shop_id,
			c.vehicle_id,
			c.changed_by,
			COALESCE(u.username, 'Unknown User') as changed_by_username,
			c.changed_at,
			c.change_type,
			c.field_changes,
			COALESCE(n.title, c.notification_title, 'Deleted Notification') as notification_title,
			c.notification_type,
			COALESCE(v.admin, c.vehicle_admin) as vehicle_admin,
			CASE WHEN c.notification_id IS NULL OR c.vehicle_id IS NULL THEN true ELSE false END as is_deleted
		FROM shop_vehicle_notification_changes c
		LEFT JOIN users u ON c.changed_by = u.uid
		LEFT JOIN shop_vehicle_notifications n ON c.notification_id = n.id
		LEFT JOIN shop_vehicle v ON c.vehicle_id = v.id
		WHERE c.vehicle_id = $1
		ORDER BY c.changed_at DESC
	`

	rows, err := repo.db.Query(rawSQL, vehicleID)
	if err != nil {
		return nil, fmt.Errorf("failed to get vehicle notification changes: %w", err)
	}
	defer rows.Close()

	var changes []response.NotificationChangeWithUsername
	for rows.Next() {
		var change response.NotificationChangeWithUsername
		var fieldChangesJSON string

		err := rows.Scan(
			&change.ID,
			&change.NotificationID,
			&change.ShopID,
			&change.VehicleID,
			&change.ChangedBy,
			&change.ChangedByUsername,
			&change.ChangedAt,
			&change.ChangeType,
			&fieldChangesJSON,
			&change.NotificationTitle,
			&change.NotificationType,
			&change.VehicleAdmin,
			&change.IsDeleted,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan change row: %w", err)
		}

		change.FieldChanges = make(map[string]interface{})
		if fieldChangesJSON != "" {
			change.FieldChanges["raw"] = fieldChangesJSON
		}

		changes = append(changes, change)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating change rows: %w", err)
	}

	return changes, nil
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
