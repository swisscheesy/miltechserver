package aggregates

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/lib/pq"
	"time"

	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/response"
	"miltechserver/api/shops/shared"
	"miltechserver/bootstrap"
)

func (repo *RepositoryImpl) getListsWithItems(ctx context.Context, tx *sql.Tx, user *bootstrap.User, shopID string, limits ListTreeLimits) ([]response.ShopListWithItems, error) {
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

	rows, err := tx.QueryContext(ctx, query, shopID, user.UserID, limits.ListsLimit, limits.ItemsLimitPerList)
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
					CreatedByUsername: shared.NullStringPtr(listCreatorName),
					Description:       listDescription,
					CreatedAt:         shared.TimePtr(listCreatedAt),
					UpdatedAt:         shared.TimePtr(listUpdatedAt),
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
				AddedByUsername: shared.NullStringPtr(itemAddedByName),
				CreatedAt:       shared.NullTimePtr(itemCreatedAt),
				UpdatedAt:       shared.NullTimePtr(itemUpdatedAt),
				Nickname:        shared.NullStringPtr(itemNickname),
				UnitOfMeasure:   shared.NullStringPtr(itemUnitOfMeasure),
			})
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate shop lists with items: %w", err)
	}

	return lists, nil
}

func (repo *RepositoryImpl) getShopSnapshot(ctx context.Context, tx *sql.Tx, user *bootstrap.User, shopID string, options ShopSnapshotOptions) (*response.ShopSnapshotResponse, error) {
	summary, err := repo.getShopSnapshotSummary(ctx, tx, user, shopID)
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
		Messages:      []response.ShopMessageResponse{},
		Services:      []response.EquipmentServiceResponse{},
		RecentChanges: []response.NotificationChangeWithUsername{},
	}

	if includes["vehicles"] {
		vehicles, err := repo.getShopSnapshotVehicles(ctx, tx, user, shopID, options.VehiclesLimit)
		if err != nil {
			return nil, err
		}
		result.Vehicles = vehicles
	}
	if includes["lists"] {
		lists, err := repo.getListsWithItems(ctx, tx, user, shopID, ListTreeLimits{
			ListsLimit:        options.ListsLimit,
			ItemsLimitPerList: options.ItemsLimitPerList,
		})
		if err != nil {
			return nil, err
		}
		result.Lists = lists
	}
	if includes["notifications"] {
		notifications, err := repo.getShopNotificationsWithItems(ctx, tx, user, shopID, options.NotificationsLimit, options.NotificationItemsLimit)
		if err != nil {
			return nil, err
		}
		result.Notifications = notifications
	}
	if includes["messages"] {
		messages, err := repo.getShopSnapshotMessages(ctx, tx, user, shopID, options.MessageLimit)
		if err != nil {
			return nil, err
		}
		result.Messages = messages
	}
	if includes["services"] {
		services, err := repo.getShopSnapshotServices(ctx, tx, user, shopID, options.ServicesLimit)
		if err != nil {
			return nil, err
		}
		result.Services = services
	}
	if includes["changes"] {
		changes, err := repo.getShopSnapshotRecentChanges(ctx, tx, user, shopID, options.ChangesLimit)
		if err != nil {
			return nil, err
		}
		result.RecentChanges = changes
	}

	return result, nil
}

func (repo *RepositoryImpl) getBootstrap(ctx context.Context, tx *sql.Tx, user *bootstrap.User, options BootstrapOptions) ([]response.ShopBootstrapSummary, error) {
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

	rows, err := tx.QueryContext(ctx, query, user.UserID)
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
		shop.Details = shared.NullStringPtr(details)
		shop.Equipment = []response.ShopEquipmentSummary{}
		shopIndexes[shop.ID] = len(shops)
		shops = append(shops, shop)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate shops bootstrap summaries: %w", err)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("failed to close bootstrap summaries: %w", err)
	}
	if len(shops) == 0 {
		return shops, nil
	}

	shopIDs := make([]string, len(shops))
	for i, shop := range shops {
		shopIDs[i] = shop.ID
	}

	equipmentByShop, err := repo.getBootstrapEquipment(ctx, tx, shopIDs, options.EquipmentLimitPerShop)
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

func (repo *RepositoryImpl) getBootstrapEquipment(ctx context.Context, tx *sql.Tx, shopIDs []string, equipmentLimitPerShop int) (map[string][]response.ShopEquipmentSummary, error) {
	if len(shopIDs) == 0 {
		return map[string][]response.ShopEquipmentSummary{}, nil
	}

	query := (`
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
	WHERE shop_id = ANY($1::text[])
)
SELECT shop_id, id, admin, model, serial, niin
FROM ranked_equipment
WHERE ($2 = 0 OR equipment_rank <= $2)
ORDER BY shop_id ASC, equipment_rank ASC`)

	args := []any{pq.Array(shopIDs), equipmentLimitPerShop}

	rows, err := tx.QueryContext(ctx, query, args...)
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

func (repo *RepositoryImpl) getShopSnapshotSummary(ctx context.Context, tx *sql.Tx, user *bootstrap.User, shopID string) (*response.ShopSnapshotSummary, error) {
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
	err := tx.QueryRowContext(ctx, query, shopID, user.UserID).Scan(
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
		if shared.ErrorsIsNoRows(err) {
			return nil, shared.ErrShopAccessDenied
		}
		return nil, fmt.Errorf("failed to query shop snapshot summary: %w", err)
	}
	summary.Details = shared.NullStringPtr(details)
	return &summary, nil
}

func (repo *RepositoryImpl) getShopSnapshotVehicles(ctx context.Context, tx *sql.Tx, user *bootstrap.User, shopID string, limit int) ([]model.ShopVehicle, error) {
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

	rows, err := tx.QueryContext(ctx, query, shopID, user.UserID, limit)
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
		vehicle.TrackedMileage = shared.NullInt32Ptr(trackedMileage)
		vehicle.TrackedHours = shared.NullInt32Ptr(trackedHours)
		vehicles = append(vehicles, vehicle)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate shop snapshot vehicles: %w", err)
	}

	return vehicles, nil
}

func (repo *RepositoryImpl) getShopNotificationsWithItems(ctx context.Context, tx *sql.Tx, user *bootstrap.User, shopID string, notificationLimit int, itemLimitPerNotification int) ([]response.VehicleNotificationWithItems, error) {
	notifications, err := repo.getShopSnapshotNotifications(ctx, tx, user, shopID, notificationLimit)
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

	items, err := repo.getItemsByNotificationIDs(ctx, tx, notificationIDs, itemLimitPerNotification)
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

func (repo *RepositoryImpl) getShopSnapshotNotifications(ctx context.Context, tx *sql.Tx, user *bootstrap.User, shopID string, limit int) ([]model.ShopVehicleNotifications, error) {
	query := buildNotificationsQuery(
		"n.shop_id = $1",
		"INNER JOIN shop_members sm ON sm.shop_id = n.shop_id AND sm.user_id = $2",
		3,
	)

	rows, err := tx.QueryContext(ctx, query, shopID, user.UserID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query shop snapshot notifications: %w", err)
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
		return nil, fmt.Errorf("failed to iterate shop snapshot notifications: %w", err)
	}

	return notifications, nil
}

func (repo *RepositoryImpl) getShopSnapshotMessages(ctx context.Context, tx *sql.Tx, user *bootstrap.User, shopID string, limit int) ([]response.ShopMessageResponse, error) {
	const query = `
SELECT msg.id, msg.shop_id, msg.user_id, msg.message, msg.created_at, msg.updated_at, msg.is_edited, msg.parent_id,
       NULLIF(BTRIM(u.username), '') AS author_username
FROM shop_messages msg
INNER JOIN shop_members sm ON sm.shop_id = msg.shop_id AND sm.user_id = $2
LEFT JOIN users u ON u.uid = msg.user_id
WHERE msg.shop_id = $1
ORDER BY msg.created_at DESC, msg.id ASC
LIMIT NULLIF($3, 0)`

	rows, err := tx.QueryContext(ctx, query, shopID, user.UserID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query shop snapshot messages: %w", err)
	}
	defer rows.Close()

	messages := []response.ShopMessageResponse{}
	for rows.Next() {
		var message response.ShopMessageResponse
		var createdAt sql.NullTime
		var updatedAt sql.NullTime
		var isEdited sql.NullBool
		var parentID sql.NullString
		var authorUsername sql.NullString
		err := rows.Scan(
			&message.ID,
			&message.ShopID,
			&message.UserID,
			&message.Message,
			&createdAt,
			&updatedAt,
			&isEdited,
			&parentID,
			&authorUsername,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan shop snapshot message: %w", err)
		}
		message.CreatedAt = shared.NullTimePtr(createdAt)
		message.UpdatedAt = shared.NullTimePtr(updatedAt)
		message.IsEdited = shared.NullBoolPtr(isEdited)
		message.ParentID = shared.NullStringPtr(parentID)
		message.AuthorUsername = shared.NullStringPtr(authorUsername)
		messages = append(messages, message)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate shop snapshot messages: %w", err)
	}

	return messages, nil
}

func (repo *RepositoryImpl) getShopSnapshotServices(ctx context.Context, tx *sql.Tx, user *bootstrap.User, shopID string, limit int) ([]response.EquipmentServiceResponse, error) {
	query := buildServiceQuery(
		"es.shop_id = $1",
		"INNER JOIN shop_members sm ON sm.shop_id = es.shop_id AND sm.user_id = $2",
		3,
	)

	rows, err := tx.QueryContext(ctx, query, shopID, user.UserID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query shop snapshot services: %w", err)
	}
	defer rows.Close()

	services := []response.EquipmentServiceResponse{}
	for rows.Next() {
		service, err := scanEquipmentService(rows)
		if err != nil {
			return nil, err
		}
		services = append(services, service)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate shop snapshot services: %w", err)
	}

	return services, nil
}

func (repo *RepositoryImpl) getShopSnapshotRecentChanges(ctx context.Context, tx *sql.Tx, user *bootstrap.User, shopID string, limit int) ([]response.NotificationChangeWithUsername, error) {
	query := buildRecentChangesQuery(
		"c.shop_id = $1",
		"INNER JOIN shop_members sm ON sm.shop_id = c.shop_id AND sm.user_id = $2",
		3,
	)

	rows, err := tx.QueryContext(ctx, query, shopID, user.UserID, limit)
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
