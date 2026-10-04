package aggregates

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/lib/pq"

	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/response"
	"miltechserver/api/shops/shared"
	"miltechserver/bootstrap"
)

func (repo *RepositoryImpl) getVehicleByIDForMember(ctx context.Context, tx *sql.Tx, user *bootstrap.User, vehicleID string) (*model.ShopVehicle, error) {
	const query = `
SELECT
	v.id, v.creator_id, v.niin, v.admin, v.model, v.serial, v.uoc,
	v.mileage, v.hours, v.comment, v.save_time, v.last_updated, v.shop_id,
	v.tracked_mileage, v.tracked_hours
FROM shop_vehicle v
INNER JOIN shop_members sm ON sm.shop_id = v.shop_id AND sm.user_id = $2
WHERE v.id = $1
LIMIT 1`

	var vehicle model.ShopVehicle
	var trackedMileage sql.NullInt64
	var trackedHours sql.NullInt64
	err := tx.QueryRowContext(ctx, query, vehicleID, user.UserID).Scan(
		&vehicle.ID,
		&vehicle.CreatorID,
		&vehicle.Niin,
		&vehicle.Admin,
		&vehicle.Model,
		&vehicle.Serial,
		&vehicle.Uoc,
		&vehicle.Mileage,
		&vehicle.Hours,
		&vehicle.Comment,
		&vehicle.SaveTime,
		&vehicle.LastUpdated,
		&vehicle.ShopID,
		&trackedMileage,
		&trackedHours,
	)
	if err != nil {
		if shared.ErrorsIsNoRows(err) {
			return nil, shared.ErrVehicleAccessDenied
		}
		return nil, fmt.Errorf("failed to query vehicle maintenance snapshot vehicle: %w", err)
	}

	vehicle.TrackedMileage = shared.NullInt32Ptr(trackedMileage)
	vehicle.TrackedHours = shared.NullInt32Ptr(trackedHours)
	return &vehicle, nil
}

func (repo *RepositoryImpl) getVehicleNotificationsWithItems(ctx context.Context, tx *sql.Tx, vehicleID string, limits SnapshotLimits) ([]response.VehicleNotificationWithItems, error) {
	notifications, err := repo.getVehicleNotifications(ctx, tx, vehicleID, limits.NotificationsLimit)
	if err != nil {
		return nil, err
	}
	if len(notifications) == 0 {
		return []response.VehicleNotificationWithItems{}, nil
	}

	notificationIDs := make([]string, len(notifications))
	for i, notification := range notifications {
		notificationIDs[i] = notification.ID
	}

	items, err := repo.getItemsByNotificationIDs(ctx, tx, notificationIDs, limits.NotificationItemsLimit)
	if err != nil {
		return nil, err
	}

	itemsByNotification := make(map[string][]model.ShopNotificationItems, len(notificationIDs))
	for _, item := range items {
		itemsByNotification[item.NotificationID] = append(itemsByNotification[item.NotificationID], item)
	}

	result := make([]response.VehicleNotificationWithItems, len(notifications))
	for i, notification := range notifications {
		notificationItems := itemsByNotification[notification.ID]
		if notificationItems == nil {
			notificationItems = []model.ShopNotificationItems{}
		}
		result[i] = response.VehicleNotificationWithItems{
			Notification: notification,
			Items:        notificationItems,
		}
	}

	return result, nil
}

// buildNotificationsQuery holds the SELECT column list, ORDER BY, and LIMIT shared by every
// scope this query is called with, so the two scopes can never diverge in behavior the way
// GetVehicleServices/getShopSnapshotServices did before the ORDER BY was unified. whereClause
// and membershipJoin carry the only real variance: which column scopes the rows, and whether a
// shop-membership check is required. limitPlaceholder is the positional arg index for LIMIT,
// which shifts depending on how many args the membership join consumes.
func buildNotificationsQuery(whereClause, membershipJoin string, limitPlaceholder int) string {
	return fmt.Sprintf(`
SELECT n.id, n.shop_id, n.vehicle_id, n.title, n.description, n.type, n.completed, n.save_time, n.last_updated, n.attached_shop_list
FROM shop_vehicle_notifications n
%s
WHERE %s
ORDER BY n.save_time DESC, n.id ASC
LIMIT NULLIF($%d, 0)`, membershipJoin, whereClause, limitPlaceholder)
}

func scanVehicleNotification(scanner rowScanner) (model.ShopVehicleNotifications, error) {
	var notification model.ShopVehicleNotifications
	err := scanner.Scan(
		&notification.ID,
		&notification.ShopID,
		&notification.VehicleID,
		&notification.Title,
		&notification.Description,
		&notification.Type,
		&notification.Completed,
		&notification.SaveTime,
		&notification.LastUpdated,
		&notification.AttachedShopList,
	)
	if err != nil {
		return model.ShopVehicleNotifications{}, fmt.Errorf("failed to scan vehicle notification: %w", err)
	}
	return notification, nil
}

func (repo *RepositoryImpl) getVehicleNotifications(ctx context.Context, tx *sql.Tx, vehicleID string, limit int) ([]model.ShopVehicleNotifications, error) {
	query := buildNotificationsQuery("n.vehicle_id = $1", "", 2)

	rows, err := tx.QueryContext(ctx, query, vehicleID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query vehicle notifications: %w", err)
	}
	defer rows.Close()

	notifications := []model.ShopVehicleNotifications{}
	for rows.Next() {
		notification, err := scanVehicleNotification(rows)
		if err != nil {
			return nil, err
		}
		notifications = append(notifications, notification)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate vehicle notifications: %w", err)
	}

	return notifications, nil
}

func (repo *RepositoryImpl) getItemsByNotificationIDs(ctx context.Context, tx *sql.Tx, notificationIDs []string, perNotificationLimit int) ([]model.ShopNotificationItems, error) {
	if len(notificationIDs) == 0 {
		return []model.ShopNotificationItems{}, nil
	}

	query := (`
WITH ranked_items AS (
	SELECT
		id,
		shop_id,
		notification_id,
		niin,
		nomenclature,
		quantity,
		save_time,
		nickname,
		unit_of_measure,
		ROW_NUMBER() OVER (
			PARTITION BY notification_id
			ORDER BY save_time ASC, id ASC
		) AS item_rank
	FROM shop_notification_items
	WHERE notification_id = ANY($1::text[])
)
SELECT id, shop_id, notification_id, niin, nomenclature, quantity, save_time, nickname, unit_of_measure
FROM ranked_items
WHERE ($2 = 0 OR item_rank <= $2)
ORDER BY notification_id ASC, save_time ASC, id ASC`)

	args := []any{pq.Array(notificationIDs), perNotificationLimit}

	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query notification items: %w", err)
	}
	defer rows.Close()

	items := []model.ShopNotificationItems{}
	for rows.Next() {
		var item model.ShopNotificationItems
		err := rows.Scan(
			&item.ID,
			&item.ShopID,
			&item.NotificationID,
			&item.Niin,
			&item.Nomenclature,
			&item.Quantity,
			&item.SaveTime,
			&item.Nickname,
			&item.UnitOfMeasure,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan notification item: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate notification items: %w", err)
	}

	return items, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}
