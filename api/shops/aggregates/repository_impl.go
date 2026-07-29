package aggregates

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"miltechserver/.gen/miltech_ng/public/model"
	. "miltechserver/.gen/miltech_ng/public/table"
	"miltechserver/api/response"
	"miltechserver/api/shops/shared"
	"miltechserver/bootstrap"

	. "github.com/go-jet/jet/v2/postgres"
	"github.com/google/uuid"
)

type RepositoryImpl struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *RepositoryImpl {
	return &RepositoryImpl{db: db}
}

func (repo *RepositoryImpl) GetListsWithItems(ctx context.Context, user *bootstrap.User, shopID string, limits ListTreeLimits) ([]response.ShopListWithItems, error) {
	const query = `
WITH ranked_lists AS (
	SELECT
		l.id, l.shop_id, l.created_by, creator.username AS created_by_username, l.description, l.created_at, l.updated_at,
		ROW_NUMBER() OVER (ORDER BY l.created_at DESC, l.id ASC) AS list_rank
	FROM shop_lists l
	INNER JOIN shop_members sm ON sm.shop_id = l.shop_id AND sm.user_id = $2
	LEFT JOIN users creator ON creator.uid = l.created_by
	WHERE l.shop_id = $1
),
ranked_items AS (
	SELECT
		i.id, i.list_id, i.niin, i.nomenclature, i.quantity, i.added_by, added.username AS added_by_username,
		i.created_at, i.updated_at, i.nickname, i.unit_of_measure,
		ROW_NUMBER() OVER (
			PARTITION BY i.list_id
			ORDER BY i.created_at ASC, i.id ASC
		) AS item_rank
	FROM shop_list_items i
	INNER JOIN ranked_lists l ON l.id = i.list_id AND ($3 = 0 OR l.list_rank <= $3)
	LEFT JOIN users added ON added.uid = i.added_by
)
SELECT
	l.id, l.shop_id, l.created_by, l.created_by_username, l.description, l.created_at, l.updated_at,
	i.id, i.list_id, i.niin, i.nomenclature, i.quantity, i.added_by, i.added_by_username,
	i.created_at, i.updated_at, i.nickname, i.unit_of_measure
FROM ranked_lists l
LEFT JOIN ranked_items i ON i.list_id = l.id AND ($4 = 0 OR i.item_rank <= $4)
WHERE $3 = 0 OR l.list_rank <= $3
ORDER BY l.list_rank ASC, i.item_rank ASC NULLS LAST, i.id ASC`

	rows, err := repo.db.QueryContext(ctx, query, shopID, user.UserID, limits.ListsLimit, limits.ItemsLimitPerList)
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

func (repo *RepositoryImpl) GetVehicleNotificationsWithItems(ctx context.Context, vehicleID string, limits SnapshotLimits) ([]response.VehicleNotificationWithItems, error) {
	notifications, err := repo.getVehicleNotifications(ctx, vehicleID, limits.NotificationsLimit)
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

	items, err := repo.getItemsByNotificationIDs(ctx, notificationIDs, limits.NotificationItemsLimit)
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
SELECT id, shop_id, vehicle_id, title, description, type, completed, save_time, last_updated, attached_shop_list
FROM shop_vehicle_notifications
WHERE vehicle_id = $1
ORDER BY save_time DESC, id ASC
LIMIT NULLIF($2, 0)`

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
			&notification.AttachedShopList,
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

func (repo *RepositoryImpl) getItemsByNotificationIDs(ctx context.Context, notificationIDs []string, perNotificationLimit int) ([]model.ShopNotificationItems, error) {
	if len(notificationIDs) == 0 {
		return []model.ShopNotificationItems{}, nil
	}

	itemLimitPlaceholder := len(notificationIDs) + 1
	query := fmt.Sprintf(`
WITH ranked_items AS (
	SELECT
		id,
		shop_id,
		notification_id,
		niin,
		nomenclature,
		quantity,
		save_time,
		ROW_NUMBER() OVER (
			PARTITION BY notification_id
			ORDER BY save_time ASC, id ASC
		) AS item_rank
	FROM shop_notification_items
	WHERE notification_id IN (%s)
)
SELECT id, shop_id, notification_id, niin, nomenclature, quantity, save_time
FROM ranked_items
WHERE ($%d = 0 OR item_rank <= $%d)
ORDER BY notification_id ASC, save_time ASC, id ASC`, placeholders(len(notificationIDs)), itemLimitPlaceholder, itemLimitPlaceholder)

	args := make([]any, 0, len(notificationIDs)+1)
	for _, id := range notificationIDs {
		args = append(args, id)
	}
	args = append(args, perNotificationLimit)

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
LIMIT NULLIF($2, 0)`

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
LIMIT NULLIF($2, 0)`

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

func (repo *RepositoryImpl) GetShopSnapshot(ctx context.Context, user *bootstrap.User, shopID string, options ShopSnapshotOptions) (*response.ShopSnapshotResponse, error) {
	summary, err := repo.getShopSnapshotSummary(ctx, user, shopID)
	if err != nil {
		return nil, err
	}

	includes := options.Includes
	if includes == nil {
		includes = map[string]bool{
			"vehicles":      true,
			"lists":         true,
			"notifications": true,
			"services":      true,
		}
	}

	result := &response.ShopSnapshotResponse{
		Shop:          *summary,
		Vehicles:      []model.ShopVehicle{},
		Lists:         []response.ShopListWithItems{},
		Notifications: []response.VehicleNotificationWithItems{},
		Messages:      []model.ShopMessages{},
		Services:      []response.EquipmentServiceResponse{},
		RecentChanges: []response.NotificationChangeWithUsername{},
	}

	if includes["vehicles"] {
		vehicles, err := repo.getShopSnapshotVehicles(ctx, user, shopID, options.VehiclesLimit)
		if err != nil {
			return nil, err
		}
		result.Vehicles = vehicles
	}
	if includes["lists"] {
		lists, err := repo.GetListsWithItems(ctx, user, shopID, ListTreeLimits{
			ListsLimit:        options.ListsLimit,
			ItemsLimitPerList: options.ItemsLimitPerList,
		})
		if err != nil {
			return nil, err
		}
		result.Lists = lists
	}
	if includes["notifications"] {
		notifications, err := repo.getShopNotificationsWithItems(ctx, user, shopID, options.NotificationsLimit, options.NotificationItemsLimit)
		if err != nil {
			return nil, err
		}
		result.Notifications = notifications
	}
	if includes["messages"] {
		messages, err := repo.getShopSnapshotMessages(ctx, user, shopID, options.MessageLimit)
		if err != nil {
			return nil, err
		}
		result.Messages = messages
	}
	if includes["services"] {
		services, err := repo.getShopSnapshotServices(ctx, user, shopID, options.ServicesLimit)
		if err != nil {
			return nil, err
		}
		result.Services = services
	}
	if includes["changes"] {
		changes, err := repo.getShopSnapshotRecentChanges(ctx, user, shopID, options.ChangesLimit)
		if err != nil {
			return nil, err
		}
		result.RecentChanges = changes
	}

	return result, nil
}

func (repo *RepositoryImpl) GetBootstrap(ctx context.Context, user *bootstrap.User, options BootstrapOptions) ([]response.ShopBootstrapSummary, error) {
	const query = `
SELECT
	s.id,
	s.name,
	s.details,
	sm.role,
	(sm.role = 'admin') AS is_admin,
	s.admin_only_lists,
	(SELECT COUNT(*) FROM shop_members m WHERE m.shop_id = s.id) AS member_count,
	(SELECT COUNT(*) FROM shop_vehicle v WHERE v.shop_id = s.id) AS vehicle_count,
	(SELECT COUNT(*) FROM shop_lists l WHERE l.shop_id = s.id) AS list_count,
	(SELECT COUNT(*) FROM shop_messages msg WHERE msg.shop_id = s.id) AS message_count,
	(SELECT COUNT(*) FROM shop_vehicle_notifications n WHERE n.shop_id = s.id) AS notification_count,
	(SELECT COUNT(*) FROM shop_notification_items ni WHERE ni.shop_id = s.id) AS notification_item_count,
	(SELECT COUNT(*) FROM equipment_services es WHERE es.shop_id = s.id AND es.is_completed = false) AS open_service_count,
	(SELECT COUNT(*) FROM equipment_services es WHERE es.shop_id = s.id) AS service_count,
	(SELECT COUNT(*) FROM shop_vehicle_notification_changes c WHERE c.shop_id = s.id) AS recent_change_count
FROM shop_members sm
INNER JOIN shops s ON s.id = sm.shop_id
WHERE sm.user_id = $1
ORDER BY s.created_at DESC NULLS LAST, s.id DESC`

	rows, err := repo.db.QueryContext(ctx, query, user.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to query shops bootstrap summaries: %w", err)
	}
	defer rows.Close()

	shops := []response.ShopBootstrapSummary{}
	shopIndexes := make(map[string]int)

	for rows.Next() {
		var shop response.ShopBootstrapSummary
		var details sql.NullString
		err := rows.Scan(
			&shop.ID,
			&shop.Name,
			&details,
			&shop.Role,
			&shop.IsAdmin,
			&shop.Settings.AdminOnlyLists,
			&shop.Counts.Members,
			&shop.Counts.Vehicles,
			&shop.Counts.Lists,
			&shop.Counts.Messages,
			&shop.Counts.Notifications,
			&shop.Counts.NotificationItems,
			&shop.Counts.OpenServices,
			&shop.Counts.Services,
			&shop.Counts.RecentChanges,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan shops bootstrap summary: %w", err)
		}
		shop.Details = nullStringPtr(details)
		shop.Equipment = []response.ShopEquipmentSummary{}
		shopIndexes[shop.ID] = len(shops)
		shops = append(shops, shop)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate shops bootstrap summaries: %w", err)
	}
	if len(shops) == 0 {
		return shops, nil
	}

	shopIDs := make([]string, len(shops))
	for i, shop := range shops {
		shopIDs[i] = shop.ID
	}

	equipmentByShop, err := repo.getBootstrapEquipment(ctx, shopIDs, options.EquipmentLimitPerShop)
	if err != nil {
		return nil, err
	}
	for shopID, equipment := range equipmentByShop {
		shopIndex, ok := shopIndexes[shopID]
		if ok {
			shops[shopIndex].Equipment = equipment
		}
	}

	return shops, nil
}

func (repo *RepositoryImpl) getBootstrapEquipment(ctx context.Context, shopIDs []string, equipmentLimitPerShop int) (map[string][]response.ShopEquipmentSummary, error) {
	if len(shopIDs) == 0 {
		return map[string][]response.ShopEquipmentSummary{}, nil
	}

	limitPlaceholder := len(shopIDs) + 1
	query := fmt.Sprintf(`
WITH ranked_equipment AS (
	SELECT
		shop_id,
		id,
		admin,
		model,
		serial,
		niin,
		ROW_NUMBER() OVER (
			PARTITION BY shop_id
			ORDER BY save_time DESC, id DESC
		) AS equipment_rank
	FROM shop_vehicle
	WHERE shop_id IN (%s)
)
SELECT shop_id, id, admin, model, serial, niin
FROM ranked_equipment
WHERE ($%d = 0 OR equipment_rank <= $%d)
ORDER BY shop_id ASC, equipment_rank ASC`, placeholders(len(shopIDs)), limitPlaceholder, limitPlaceholder)

	args := make([]any, 0, len(shopIDs)+1)
	for _, shopID := range shopIDs {
		args = append(args, shopID)
	}
	args = append(args, equipmentLimitPerShop)

	rows, err := repo.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query shops bootstrap equipment: %w", err)
	}
	defer rows.Close()

	equipmentByShop := make(map[string][]response.ShopEquipmentSummary, len(shopIDs))
	for rows.Next() {
		var shopID string
		var equipment response.ShopEquipmentSummary
		err := rows.Scan(
			&shopID,
			&equipment.ID,
			&equipment.Admin,
			&equipment.Model,
			&equipment.Serial,
			&equipment.Niin,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan shops bootstrap equipment: %w", err)
		}
		equipmentByShop[shopID] = append(equipmentByShop[shopID], equipment)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate shops bootstrap equipment: %w", err)
	}

	return equipmentByShop, nil
}

func (repo *RepositoryImpl) getShopSnapshotSummary(ctx context.Context, user *bootstrap.User, shopID string) (*response.ShopSnapshotSummary, error) {
	const query = `
SELECT
	s.id,
	s.name,
	s.details,
	sm.role,
	(sm.role = 'admin') AS is_admin,
	s.admin_only_lists,
	(SELECT COUNT(*) FROM shop_members m WHERE m.shop_id = s.id) AS member_count,
	(SELECT COUNT(*) FROM shop_vehicle v WHERE v.shop_id = s.id) AS vehicle_count,
	(SELECT COUNT(*) FROM shop_lists l WHERE l.shop_id = s.id) AS list_count,
	(SELECT COUNT(*) FROM shop_messages msg WHERE msg.shop_id = s.id) AS message_count,
	(SELECT COUNT(*) FROM shop_vehicle_notifications n WHERE n.shop_id = s.id) AS notification_count,
	(SELECT COUNT(*) FROM equipment_services es WHERE es.shop_id = s.id AND es.is_completed = false) AS open_service_count
FROM shop_members sm
INNER JOIN shops s ON s.id = sm.shop_id
WHERE sm.shop_id = $1 AND sm.user_id = $2
LIMIT 1`

	var summary response.ShopSnapshotSummary
	var details sql.NullString
	err := repo.db.QueryRowContext(ctx, query, shopID, user.UserID).Scan(
		&summary.ID,
		&summary.Name,
		&details,
		&summary.Role,
		&summary.IsAdmin,
		&summary.Settings.AdminOnlyLists,
		&summary.Counts.Members,
		&summary.Counts.Vehicles,
		&summary.Counts.Lists,
		&summary.Counts.Messages,
		&summary.Counts.Notifications,
		&summary.Counts.OpenServices,
	)
	if err != nil {
		if errorsIsNoRows(err) {
			return nil, shared.ErrShopAccessDenied
		}
		return nil, fmt.Errorf("failed to query shop snapshot summary: %w", err)
	}
	summary.Details = nullStringPtr(details)
	return &summary, nil
}

func (repo *RepositoryImpl) getShopSnapshotVehicles(ctx context.Context, user *bootstrap.User, shopID string, limit int) ([]model.ShopVehicle, error) {
	const query = `
SELECT
	v.id, v.creator_id, v.niin, v.admin, v.model, v.serial, v.uoc,
	v.mileage, v.hours, v.comment, v.save_time, v.last_updated, v.shop_id,
	v.tracked_mileage, v.tracked_hours
FROM shop_vehicle v
INNER JOIN shop_members sm ON sm.shop_id = v.shop_id AND sm.user_id = $2
WHERE v.shop_id = $1
ORDER BY v.save_time DESC, v.id ASC
LIMIT NULLIF($3, 0)`

	rows, err := repo.db.QueryContext(ctx, query, shopID, user.UserID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query shop snapshot vehicles: %w", err)
	}
	defer rows.Close()

	vehicles := []model.ShopVehicle{}
	for rows.Next() {
		var vehicle model.ShopVehicle
		var trackedMileage sql.NullInt64
		var trackedHours sql.NullInt64
		err := rows.Scan(
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
			return nil, fmt.Errorf("failed to scan shop snapshot vehicle: %w", err)
		}
		vehicle.TrackedMileage = nullInt32Ptr(trackedMileage)
		vehicle.TrackedHours = nullInt32Ptr(trackedHours)
		vehicles = append(vehicles, vehicle)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate shop snapshot vehicles: %w", err)
	}

	return vehicles, nil
}

func (repo *RepositoryImpl) getShopNotificationsWithItems(ctx context.Context, user *bootstrap.User, shopID string, notificationLimit int, itemLimitPerNotification int) ([]response.VehicleNotificationWithItems, error) {
	notifications, err := repo.getShopSnapshotNotifications(ctx, user, shopID, notificationLimit)
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

	items, err := repo.getSnapshotItemsByNotificationIDs(ctx, notificationIDs, itemLimitPerNotification)
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

func (repo *RepositoryImpl) getSnapshotItemsByNotificationIDs(ctx context.Context, notificationIDs []string, perNotificationLimit int) ([]model.ShopNotificationItems, error) {
	if len(notificationIDs) == 0 {
		return []model.ShopNotificationItems{}, nil
	}

	itemLimitPlaceholder := len(notificationIDs) + 1
	query := fmt.Sprintf(`
WITH ranked_items AS (
	SELECT
		id,
		shop_id,
		notification_id,
		niin,
		nomenclature,
		quantity,
		save_time,
		ROW_NUMBER() OVER (
			PARTITION BY notification_id
			ORDER BY save_time ASC, id ASC
		) AS item_rank
	FROM shop_notification_items
	WHERE notification_id IN (%s)
)
SELECT id, shop_id, notification_id, niin, nomenclature, quantity, save_time
FROM ranked_items
WHERE ($%d = 0 OR item_rank <= $%d)
ORDER BY notification_id ASC, save_time ASC, id ASC`, placeholders(len(notificationIDs)), itemLimitPlaceholder, itemLimitPlaceholder)

	args := make([]any, 0, len(notificationIDs)+1)
	for _, id := range notificationIDs {
		args = append(args, id)
	}
	args = append(args, perNotificationLimit)

	rows, err := repo.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query bounded snapshot notification items: %w", err)
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
			return nil, fmt.Errorf("failed to scan bounded snapshot notification item: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate bounded snapshot notification items: %w", err)
	}

	return items, nil
}

func (repo *RepositoryImpl) getShopSnapshotNotifications(ctx context.Context, user *bootstrap.User, shopID string, limit int) ([]model.ShopVehicleNotifications, error) {
	const query = `
SELECT n.id, n.shop_id, n.vehicle_id, n.title, n.description, n.type, n.completed, n.save_time, n.last_updated, n.attached_shop_list
FROM shop_vehicle_notifications n
INNER JOIN shop_members sm ON sm.shop_id = n.shop_id AND sm.user_id = $2
WHERE n.shop_id = $1
ORDER BY n.save_time DESC, n.id ASC
LIMIT NULLIF($3, 0)`

	rows, err := repo.db.QueryContext(ctx, query, shopID, user.UserID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query shop snapshot notifications: %w", err)
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
			&notification.AttachedShopList,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan shop snapshot notification: %w", err)
		}
		notifications = append(notifications, notification)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate shop snapshot notifications: %w", err)
	}

	return notifications, nil
}

func (repo *RepositoryImpl) getShopSnapshotMessages(ctx context.Context, user *bootstrap.User, shopID string, limit int) ([]model.ShopMessages, error) {
	const query = `
SELECT msg.id, msg.shop_id, msg.user_id, msg.message, msg.created_at, msg.updated_at, msg.is_edited, msg.parent_id
FROM shop_messages msg
INNER JOIN shop_members sm ON sm.shop_id = msg.shop_id AND sm.user_id = $2
WHERE msg.shop_id = $1
ORDER BY msg.created_at DESC, msg.id ASC
LIMIT NULLIF($3, 0)`

	rows, err := repo.db.QueryContext(ctx, query, shopID, user.UserID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query shop snapshot messages: %w", err)
	}
	defer rows.Close()

	messages := []model.ShopMessages{}
	for rows.Next() {
		var message model.ShopMessages
		var createdAt sql.NullTime
		var updatedAt sql.NullTime
		var isEdited sql.NullBool
		var parentID sql.NullString
		err := rows.Scan(
			&message.ID,
			&message.ShopID,
			&message.UserID,
			&message.Message,
			&createdAt,
			&updatedAt,
			&isEdited,
			&parentID,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan shop snapshot message: %w", err)
		}
		message.CreatedAt = nullTimePtr(createdAt)
		message.UpdatedAt = nullTimePtr(updatedAt)
		message.IsEdited = nullBoolPtr(isEdited)
		message.ParentID = nullStringPtr(parentID)
		messages = append(messages, message)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate shop snapshot messages: %w", err)
	}

	return messages, nil
}

func (repo *RepositoryImpl) getShopSnapshotServices(ctx context.Context, user *bootstrap.User, shopID string, limit int) ([]response.EquipmentServiceResponse, error) {
	const query = `
SELECT
	es.id, es.shop_id, es.equipment_id, es.list_id, es.description, es.service_type,
	es.created_by, COALESCE(u.username, 'Unknown User') AS created_by_username,
	es.is_completed, es.created_at, es.updated_at, es.service_date, es.service_hours,
	es.completion_date
FROM equipment_services es
INNER JOIN shop_members sm ON sm.shop_id = es.shop_id AND sm.user_id = $2
LEFT JOIN users u ON u.uid = es.created_by
WHERE es.shop_id = $1
ORDER BY es.service_date ASC NULLS LAST, es.created_at DESC, es.id ASC
LIMIT NULLIF($3, 0)`

	rows, err := repo.db.QueryContext(ctx, query, shopID, user.UserID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query shop snapshot services: %w", err)
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
			return nil, fmt.Errorf("failed to scan shop snapshot service: %w", err)
		}
		service.ServiceDate = nullTimePtr(serviceDate)
		service.ServiceHours = nullInt32Ptr(serviceHours)
		service.CompletionDate = nullTimePtr(completionDate)
		services = append(services, service)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate shop snapshot services: %w", err)
	}

	return services, nil
}

func (repo *RepositoryImpl) getShopSnapshotRecentChanges(ctx context.Context, user *bootstrap.User, shopID string, limit int) ([]response.NotificationChangeWithUsername, error) {
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
INNER JOIN shop_members sm ON sm.shop_id = c.shop_id AND sm.user_id = $2
LEFT JOIN users u ON c.changed_by = u.uid
LEFT JOIN shop_vehicle_notifications n ON c.notification_id = n.id
LEFT JOIN shop_vehicle v ON c.vehicle_id = v.id
WHERE c.shop_id = $1
ORDER BY c.changed_at DESC, c.id ASC
LIMIT NULLIF($3, 0)`

	rows, err := repo.db.QueryContext(ctx, query, shopID, user.UserID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query shop snapshot changes: %w", err)
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
		return nil, fmt.Errorf("failed to iterate shop snapshot changes: %w", err)
	}

	return changes, nil
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

func nullBoolPtr(value sql.NullBool) *bool {
	if !value.Valid {
		return nil
	}
	return &value.Bool
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

func (repo *RepositoryImpl) GetEquipmentPmcsHistory(ctx context.Context, user *bootstrap.User) ([]response.EquipmentWithPmcsHistory, error) {
	equipmentStmt := SELECT(
		ShopVehicle.ID.AS("id"),
		ShopVehicle.ShopID.AS("shop_id"),
		ShopVehicle.Admin.AS("admin"),
		ShopVehicle.Model.AS("model"),
		ShopVehicle.Serial.AS("serial"),
		ShopVehicle.Niin.AS("niin"),
	).FROM(
		ShopVehicle.INNER_JOIN(ShopMembers, ShopMembers.ShopID.EQ(ShopVehicle.ShopID)),
	).WHERE(
		ShopMembers.UserID.EQ(String(user.UserID)),
	).ORDER_BY(
		ShopVehicle.SaveTime.DESC(),
		ShopVehicle.ID.DESC(),
	)

	// NOTE: the destination must be an anonymous struct, not a named local type.
	// go-jet's query result mapper derives its column-matching key from the destination
	// struct's reflect.Type.Name(): a named type causes jet to require dot-prefixed column
	// aliases (e.g. "equipmentrow.id", the convention used for embedded/nested table structs),
	// while an anonymous struct type (Name() == "") matches plain bare-field-name aliases like
	// "id" directly. Since this query aliases columns as plain names via .AS("id") etc., a named
	// struct here silently matches zero columns and returns zero rows with no error. Verified via
	// isolated repro against the live DB: identical query + fields, only named vs. anonymous type,
	// produced 0 vs 2 rows. This matches the working pattern already used a few lines below
	// (the `counts` query) and in pmcs_sbs_progress/repository_impl.go's requireVehicleAccess.
	var equipmentRows []struct {
		ID     string `sql:"id"`
		ShopID string `sql:"shop_id"`
		Admin  string `sql:"admin"`
		Model  string `sql:"model"`
		Serial string `sql:"serial"`
		Niin   string `sql:"niin"`
	}
	if err := equipmentStmt.QueryContext(ctx, repo.db, &equipmentRows); err != nil {
		return nil, fmt.Errorf("failed to query equipment for pmcs history: %w", err)
	}
	if len(equipmentRows) == 0 {
		return []response.EquipmentWithPmcsHistory{}, nil
	}

	equipmentIDs := make([]Expression, 0, len(equipmentRows))
	for _, row := range equipmentRows {
		equipmentIDs = append(equipmentIDs, String(row.ID))
	}

	var inspections []struct {
		model.PmcsSbsInspections
		PerformedByUsername *string `sql:"performed_by_username"`
	}
	inspectionsStmt := SELECT(
		PmcsSbsInspections.AllColumns,
		Users.Username.AS("performed_by_username"),
	).
		FROM(PmcsSbsInspections.LEFT_JOIN(Users, Users.UID.EQ(PmcsSbsInspections.PerformedBy))).
		WHERE(PmcsSbsInspections.EquipmentID.IN(equipmentIDs...)).
		ORDER_BY(PmcsSbsInspections.EquipmentID.ASC(), PmcsSbsInspections.PerformedDate.DESC())

	if err := inspectionsStmt.QueryContext(ctx, repo.db, &inspections); err != nil {
		return nil, fmt.Errorf("failed to query pmcs inspections for equipment history: %w", err)
	}

	faultCountByInspectionID := make(map[uuid.UUID]int)
	if len(inspections) > 0 {
		inspectionIDs := make([]Expression, 0, len(inspections))
		for _, inspection := range inspections {
			inspectionIDs = append(inspectionIDs, UUID(inspection.ID))
		}

		var counts []struct {
			PmcsID uuid.UUID `sql:"pmcs_id"`
			Total  int32     `sql:"total"`
		}
		countStmt := SELECT(
			PmcsSbsFaults.PmcsID.AS("pmcs_id"),
			COUNT(PmcsSbsFaults.PmcsID).AS("total"),
		).FROM(PmcsSbsFaults).
			WHERE(PmcsSbsFaults.PmcsID.IN(inspectionIDs...)).
			GROUP_BY(PmcsSbsFaults.PmcsID)

		if err := countStmt.QueryContext(ctx, repo.db, &counts); err != nil {
			return nil, fmt.Errorf("failed to count pmcs faults for equipment history: %w", err)
		}
		for _, count := range counts {
			faultCountByInspectionID[count.PmcsID] = int(count.Total)
		}
	}

	commentCountByInspectionID := make(map[uuid.UUID]int)
	if len(inspections) > 0 {
		inspectionIDs := make([]Expression, 0, len(inspections))
		for _, inspection := range inspections {
			inspectionIDs = append(inspectionIDs, UUID(inspection.ID))
		}

		var commentCounts []struct {
			PmcsID uuid.UUID `sql:"pmcs_id"`
			Total  int32     `sql:"total"`
		}
		commentCountStmt := SELECT(
			PmcsSbsInspectionComments.PmcsID.AS("pmcs_id"),
			COUNT(PmcsSbsInspectionComments.PmcsID).AS("total"),
		).FROM(PmcsSbsInspectionComments).
			WHERE(PmcsSbsInspectionComments.PmcsID.IN(inspectionIDs...)).
			GROUP_BY(PmcsSbsInspectionComments.PmcsID)

		if err := commentCountStmt.QueryContext(ctx, repo.db, &commentCounts); err != nil {
			return nil, fmt.Errorf("failed to count pmcs inspection comments for equipment history: %w", err)
		}
		for _, count := range commentCounts {
			commentCountByInspectionID[count.PmcsID] = int(count.Total)
		}
	}

	historyByEquipmentID := make(map[string][]response.PmcsHistorySummary, len(equipmentRows))
	for _, inspection := range inspections {
		historyByEquipmentID[inspection.EquipmentID] = append(historyByEquipmentID[inspection.EquipmentID], response.PmcsHistorySummary{
			ID:                  inspection.ID,
			GuideManual:         inspection.GuideManual,
			PerformedDate:       inspection.PerformedDate,
			FaultCount:          faultCountByInspectionID[inspection.ID],
			CommentCount:        commentCountByInspectionID[inspection.ID],
			CreatedAt:           inspection.CreatedAt,
			PerformedBy:         inspection.PerformedBy,
			PerformedByUsername: inspection.PerformedByUsername,
		})
	}

	equipment := make([]response.EquipmentWithPmcsHistory, 0, len(equipmentRows))
	for _, row := range equipmentRows {
		history := historyByEquipmentID[row.ID]
		if history == nil {
			history = []response.PmcsHistorySummary{}
		}
		equipment = append(equipment, response.EquipmentWithPmcsHistory{
			ShopEquipmentSummary: response.ShopEquipmentSummary{
				ID:     row.ID,
				Admin:  row.Admin,
				Model:  row.Model,
				Serial: row.Serial,
				Niin:   row.Niin,
			},
			ShopID:         row.ShopID,
			HistoricalPmcs: history,
		})
	}
	return equipment, nil
}
