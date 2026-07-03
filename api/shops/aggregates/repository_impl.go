package aggregates

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/response"
	"miltechserver/api/shops/shared"
	"miltechserver/bootstrap"
)

type RepositoryImpl struct {
	db *sql.DB
}

const maintenanceSnapshotNotificationLimit = 50

func NewRepository(db *sql.DB) *RepositoryImpl {
	return &RepositoryImpl{db: db}
}

func (repo *RepositoryImpl) GetListsWithItems(ctx context.Context, user *bootstrap.User, shopID string) ([]response.ShopListWithItems, error) {
	const query = `
SELECT
	l.id, l.shop_id, l.created_by, creator.username, l.description, l.created_at, l.updated_at,
	i.id, i.list_id, i.niin, i.nomenclature, i.quantity, i.added_by, added.username,
	i.created_at, i.updated_at, i.nickname, i.unit_of_measure
FROM shop_lists l
INNER JOIN shop_members sm ON sm.shop_id = l.shop_id AND sm.user_id = $2
LEFT JOIN users creator ON creator.uid = l.created_by
LEFT JOIN shop_list_items i ON i.list_id = l.id
LEFT JOIN users added ON added.uid = i.added_by
WHERE l.shop_id = $1
ORDER BY l.created_at DESC, l.id ASC, i.created_at ASC, i.id ASC`

	rows, err := repo.db.QueryContext(ctx, query, shopID, user.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to query shop lists with items: %w", err)
	}
	defer rows.Close()

	lists := []response.ShopListWithItems{}
	listIndexes := make(map[string]int)

	for rows.Next() {
		var (
			listID            string
			listShopID        string
			listCreatedBy     string
			listCreatorName   sql.NullString
			listDescription   string
			listCreatedAt     time.Time
			listUpdatedAt     time.Time
			itemID            sql.NullString
			itemListID        sql.NullString
			itemNiin          sql.NullString
			itemNomenclature  sql.NullString
			itemQuantity      sql.NullInt64
			itemAddedBy       sql.NullString
			itemAddedByName   sql.NullString
			itemCreatedAt     sql.NullTime
			itemUpdatedAt     sql.NullTime
			itemNickname      sql.NullString
			itemUnitOfMeasure sql.NullString
		)

		err := rows.Scan(
			&listID,
			&listShopID,
			&listCreatedBy,
			&listCreatorName,
			&listDescription,
			&listCreatedAt,
			&listUpdatedAt,
			&itemID,
			&itemListID,
			&itemNiin,
			&itemNomenclature,
			&itemQuantity,
			&itemAddedBy,
			&itemAddedByName,
			&itemCreatedAt,
			&itemUpdatedAt,
			&itemNickname,
			&itemUnitOfMeasure,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan shop list item aggregate row: %w", err)
		}

		listIndex, ok := listIndexes[listID]
		if !ok {
			lists = append(lists, response.ShopListWithItems{
				ShopListWithUsername: response.ShopListWithUsername{
					ID:                listID,
					ShopID:            listShopID,
					CreatedBy:         listCreatedBy,
					CreatedByUsername: nullStringPtr(listCreatorName),
					Description:       listDescription,
					CreatedAt:         timePtr(listCreatedAt),
					UpdatedAt:         timePtr(listUpdatedAt),
				},
				Items: []response.ShopListItemWithUsername{},
			})
			listIndex = len(lists) - 1
			listIndexes[listID] = listIndex
		}

		if itemID.Valid {
			lists[listIndex].Items = append(lists[listIndex].Items, response.ShopListItemWithUsername{
				ID:              itemID.String,
				ListID:          itemListID.String,
				Niin:            itemNiin.String,
				Nomenclature:    itemNomenclature.String,
				Quantity:        int32(itemQuantity.Int64),
				AddedBy:         itemAddedBy.String,
				AddedByUsername: nullStringPtr(itemAddedByName),
				CreatedAt:       nullTimePtr(itemCreatedAt),
				UpdatedAt:       nullTimePtr(itemUpdatedAt),
				Nickname:        nullStringPtr(itemNickname),
				UnitOfMeasure:   nullStringPtr(itemUnitOfMeasure),
			})
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate shop lists with items: %w", err)
	}

	return lists, nil
}

func (repo *RepositoryImpl) GetVehicleByIDForMember(ctx context.Context, user *bootstrap.User, vehicleID string) (*model.ShopVehicle, error) {
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
	err := repo.db.QueryRowContext(ctx, query, vehicleID, user.UserID).Scan(
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
		if errorsIsNoRows(err) {
			return nil, shared.ErrVehicleAccessDenied
		}
		return nil, fmt.Errorf("failed to query vehicle maintenance snapshot vehicle: %w", err)
	}

	vehicle.TrackedMileage = nullInt32Ptr(trackedMileage)
	vehicle.TrackedHours = nullInt32Ptr(trackedHours)
	return &vehicle, nil
}

func (repo *RepositoryImpl) GetVehicleNotificationsWithItems(ctx context.Context, vehicleID string) ([]response.VehicleNotificationWithItems, error) {
	notifications, err := repo.getVehicleNotifications(ctx, vehicleID, maintenanceSnapshotNotificationLimit)
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

	items, err := repo.getItemsByNotificationIDs(ctx, notificationIDs)
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

func (repo *RepositoryImpl) getVehicleNotifications(ctx context.Context, vehicleID string, limit int) ([]model.ShopVehicleNotifications, error) {
	const query = `
SELECT id, shop_id, vehicle_id, title, description, type, completed, save_time, last_updated
FROM shop_vehicle_notifications
WHERE vehicle_id = $1
ORDER BY save_time DESC, id ASC
LIMIT $2`

	rows, err := repo.db.QueryContext(ctx, query, vehicleID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query vehicle notifications: %w", err)
	}
	defer rows.Close()

	notifications := []model.ShopVehicleNotifications{}
	for rows.Next() {
		var notification model.ShopVehicleNotifications
		err := rows.Scan(
			&notification.ID,
			&notification.ShopID,
			&notification.VehicleID,
			&notification.Title,
			&notification.Description,
			&notification.Type,
			&notification.Completed,
			&notification.SaveTime,
			&notification.LastUpdated,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan vehicle notification: %w", err)
		}
		notifications = append(notifications, notification)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate vehicle notifications: %w", err)
	}

	return notifications, nil
}

func (repo *RepositoryImpl) getItemsByNotificationIDs(ctx context.Context, notificationIDs []string) ([]model.ShopNotificationItems, error) {
	if len(notificationIDs) == 0 {
		return []model.ShopNotificationItems{}, nil
	}

	query := fmt.Sprintf(`
SELECT id, shop_id, notification_id, niin, nomenclature, quantity, save_time
FROM shop_notification_items
WHERE notification_id IN (%s)
ORDER BY save_time ASC, id ASC`, placeholders(len(notificationIDs)))

	args := make([]any, len(notificationIDs))
	for i, id := range notificationIDs {
		args[i] = id
	}

	rows, err := repo.db.QueryContext(ctx, query, args...)
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

func (repo *RepositoryImpl) GetVehicleRecentChanges(ctx context.Context, vehicleID string, limit int) ([]response.NotificationChangeWithUsername, error) {
	const query = `
SELECT
	c.id,
	c.notification_id,
	c.shop_id,
	c.vehicle_id,
	c.changed_by,
	COALESCE(u.username, 'Unknown User') AS changed_by_username,
	c.changed_at,
	c.change_type,
	c.field_changes,
	COALESCE(n.title, c.notification_title, 'Deleted Notification') AS notification_title,
	c.notification_type,
	COALESCE(v.admin, c.vehicle_admin) AS vehicle_admin,
	CASE WHEN c.notification_id IS NULL OR c.vehicle_id IS NULL THEN true ELSE false END AS is_deleted
FROM shop_vehicle_notification_changes c
LEFT JOIN users u ON c.changed_by = u.uid
LEFT JOIN shop_vehicle_notifications n ON c.notification_id = n.id
LEFT JOIN shop_vehicle v ON c.vehicle_id = v.id
WHERE c.vehicle_id = $1
ORDER BY c.changed_at DESC, c.id ASC
LIMIT $2`

	rows, err := repo.db.QueryContext(ctx, query, vehicleID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query vehicle notification changes: %w", err)
	}
	defer rows.Close()

	changes := []response.NotificationChangeWithUsername{}
	for rows.Next() {
		change, err := scanNotificationChange(rows)
		if err != nil {
			return nil, err
		}
		changes = append(changes, change)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate vehicle notification changes: %w", err)
	}

	return changes, nil
}

func (repo *RepositoryImpl) GetVehicleServices(ctx context.Context, vehicleID string, limit int) ([]response.EquipmentServiceResponse, error) {
	const query = `
SELECT
	es.id, es.shop_id, es.equipment_id, es.list_id, es.description, es.service_type,
	es.created_by, COALESCE(u.username, 'Unknown User') AS created_by_username,
	es.is_completed, es.created_at, es.updated_at, es.service_date, es.service_hours,
	es.completion_date
FROM equipment_services es
LEFT JOIN users u ON u.uid = es.created_by
WHERE es.equipment_id = $1
ORDER BY es.service_date DESC NULLS LAST, es.created_at DESC, es.id ASC
LIMIT $2`

	rows, err := repo.db.QueryContext(ctx, query, vehicleID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query vehicle equipment services: %w", err)
	}
	defer rows.Close()

	services := []response.EquipmentServiceResponse{}
	for rows.Next() {
		var service response.EquipmentServiceResponse
		var serviceDate sql.NullTime
		var serviceHours sql.NullInt64
		var completionDate sql.NullTime
		err := rows.Scan(
			&service.ID,
			&service.ShopID,
			&service.EquipmentID,
			&service.ListID,
			&service.Description,
			&service.ServiceType,
			&service.CreatedBy,
			&service.CreatedByUsername,
			&service.IsCompleted,
			&service.CreatedAt,
			&service.UpdatedAt,
			&serviceDate,
			&serviceHours,
			&completionDate,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan vehicle equipment service: %w", err)
		}
		service.ServiceDate = nullTimePtr(serviceDate)
		service.ServiceHours = nullInt32Ptr(serviceHours)
		service.CompletionDate = nullTimePtr(completionDate)
		services = append(services, service)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate vehicle equipment services: %w", err)
	}

	return services, nil
}

func (repo *RepositoryImpl) GetShopSnapshot(context.Context, *bootstrap.User, string, ShopSnapshotOptions) (*response.ShopSnapshotResponse, error) {
	return nil, ErrAggregateUnavailable
}

func (repo *RepositoryImpl) GetBootstrap(context.Context, *bootstrap.User, BootstrapOptions) ([]response.ShopBootstrapSummary, error) {
	return nil, ErrAggregateUnavailable
}

func timePtr(value time.Time) *time.Time {
	return &value
}

func nullTimePtr(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	return &value.Time
}

func nullStringPtr(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	return &value.String
}

func nullInt32Ptr(value sql.NullInt64) *int32 {
	if !value.Valid {
		return nil
	}
	intValue := int32(value.Int64)
	return &intValue
}

func errorsIsNoRows(err error) bool {
	return err == sql.ErrNoRows
}

func placeholders(count int) string {
	values := make([]string, count)
	for i := range values {
		values[i] = fmt.Sprintf("$%d", i+1)
	}
	return strings.Join(values, ", ")
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanNotificationChange(scanner rowScanner) (response.NotificationChangeWithUsername, error) {
	var change response.NotificationChangeWithUsername
	var notificationID sql.NullString
	var vehicleID sql.NullString
	var changedBy sql.NullString
	var notificationType sql.NullString
	var vehicleAdmin sql.NullString
	var fieldChanges []byte

	err := scanner.Scan(
		&change.ID,
		&notificationID,
		&change.ShopID,
		&vehicleID,
		&changedBy,
		&change.ChangedByUsername,
		&change.ChangedAt,
		&change.ChangeType,
		&fieldChanges,
		&change.NotificationTitle,
		&notificationType,
		&vehicleAdmin,
		&change.IsDeleted,
	)
	if err != nil {
		return response.NotificationChangeWithUsername{}, fmt.Errorf("failed to scan notification change: %w", err)
	}

	change.NotificationID = nullStringPtr(notificationID)
	change.VehicleID = nullStringPtr(vehicleID)
	change.ChangedBy = nullStringPtr(changedBy)
	change.NotificationType = nullStringPtr(notificationType)
	change.VehicleAdmin = nullStringPtr(vehicleAdmin)
	change.FieldChanges = map[string]interface{}{}
	if len(fieldChanges) > 0 {
		change.FieldChanges["raw"] = string(fieldChanges)
	}

	return change, nil
}
