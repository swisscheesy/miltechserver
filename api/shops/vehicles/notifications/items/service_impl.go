package items

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/bootstrap"
	"time"

	"github.com/google/uuid"
)

type ServiceImpl struct {
	repo Repository
}

func NewService(repo Repository) *ServiceImpl {
	return &ServiceImpl{repo: repo}
}

func (service *ServiceImpl) AddNotificationItem(user *bootstrap.User, item model.ShopNotificationItems) (*model.ShopNotificationItems, error) {
	if user == nil {
		return nil, errors.New("unauthorized user")
	}

	notification, err := service.repo.GetVehicleNotificationByID(user, item.NotificationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get notification: %w", err)
	}

	vehicle, err := service.repo.GetShopVehicleByID(user, notification.VehicleID)
	if err != nil {
		return nil, fmt.Errorf("failed to get vehicle: %w", err)
	}

	isMember, err := service.repo.IsUserMemberOfShop(user, notification.ShopID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify membership: %w", err)
	}

	if !isMember {
		return nil, errors.New("access denied: user is not a member of this shop")
	}

	item.ID = uuid.New().String()
	item.ShopID = notification.ShopID
	item.SaveTime = time.Now()

	createdItem, err := service.repo.CreateNotificationItem(user, item)
	if err != nil {
		return nil, fmt.Errorf("failed to add notification item: %w", err)
	}

	fieldChanges, err := buildItemAdditionFieldChanges([]model.ShopNotificationItems{*createdItem})
	if err != nil {
		slog.Warn("Failed to build field changes for item addition", "error", err)
		fieldChanges = `{"fields_changed": ["items"], "item_count": 1}`
	}

	service.recordNotificationChange(
		user,
		item.NotificationID,
		notification.ShopID,
		notification.VehicleID,
		"items_added",
		fieldChanges,
		notification.Title,
		notification.Type,
		vehicle.Admin,
	)

	slog.Info("Notification item added", "user_id", user.UserID, "notification_id", item.NotificationID, "item_id", item.ID)
	return createdItem, nil
}

func (service *ServiceImpl) GetNotificationItems(user *bootstrap.User, notificationID string) ([]model.ShopNotificationItems, error) {
	if user == nil {
		return nil, errors.New("unauthorized user")
	}

	notification, err := service.repo.GetVehicleNotificationByID(user, notificationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get notification: %w", err)
	}

	isMember, err := service.repo.IsUserMemberOfShop(user, notification.ShopID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify membership: %w", err)
	}

	if !isMember {
		return nil, errors.New("access denied: user is not a member of this shop")
	}

	items, err := service.repo.GetNotificationItems(user, notificationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get notification items: %w", err)
	}

	if items == nil {
		return []model.ShopNotificationItems{}, nil
	}

	return items, nil
}

func (service *ServiceImpl) GetShopNotificationItems(user *bootstrap.User, shopID string) ([]model.ShopNotificationItems, error) {
	if user == nil {
		return nil, errors.New("unauthorized user")
	}

	isMember, err := service.repo.IsUserMemberOfShop(user, shopID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify membership: %w", err)
	}

	if !isMember {
		return nil, errors.New("access denied: user is not a member of this shop")
	}

	items, err := service.repo.GetShopNotificationItems(user, shopID)
	if err != nil {
		return nil, fmt.Errorf("failed to get shop notification items: %w", err)
	}

	if items == nil {
		return []model.ShopNotificationItems{}, nil
	}

	return items, nil
}

func (service *ServiceImpl) AddNotificationItemList(user *bootstrap.User, items []model.ShopNotificationItems) ([]model.ShopNotificationItems, error) {
	if user == nil {
		return nil, errors.New("unauthorized user")
	}

	if len(items) == 0 {
		return nil, errors.New("no items to add")
	}

	notification, err := service.repo.GetVehicleNotificationByID(user, items[0].NotificationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get notification: %w", err)
	}

	vehicle, err := service.repo.GetShopVehicleByID(user, notification.VehicleID)
	if err != nil {
		return nil, fmt.Errorf("failed to get vehicle: %w", err)
	}

	isMember, err := service.repo.IsUserMemberOfShop(user, notification.ShopID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify membership: %w", err)
	}

	if !isMember {
		return nil, errors.New("access denied: user is not a member of this shop")
	}

	now := time.Now()
	for i := range items {
		items[i].ID = uuid.New().String()
		items[i].ShopID = notification.ShopID
		items[i].SaveTime = now
	}

	createdItems, err := service.repo.CreateNotificationItemList(user, items)
	if err != nil {
		return nil, fmt.Errorf("failed to add notification items: %w", err)
	}

	fieldChanges, err := buildItemAdditionFieldChanges(createdItems)
	if err != nil {
		slog.Warn("Failed to build field changes for item additions", "error", err)
		fieldChanges = fmt.Sprintf(`{"fields_changed": ["items"], "item_count": %d}`, len(createdItems))
	}

	service.recordNotificationChange(
		user,
		items[0].NotificationID,
		notification.ShopID,
		notification.VehicleID,
		"items_added",
		fieldChanges,
		notification.Title,
		notification.Type,
		vehicle.Admin,
	)

	slog.Info("Notification items added", "user_id", user.UserID, "notification_id", items[0].NotificationID, "count", len(createdItems))
	return createdItems, nil
}

func (service *ServiceImpl) RemoveNotificationItem(user *bootstrap.User, itemID string) error {
	if user == nil {
		return errors.New("unauthorized user")
	}

	item, err := service.repo.GetNotificationItemByID(user, itemID)
	if err != nil {
		return fmt.Errorf("failed to get notification item: %w", err)
	}

	notification, err := service.repo.GetVehicleNotificationByID(user, item.NotificationID)
	if err != nil {
		return fmt.Errorf("failed to get notification: %w", err)
	}

	vehicle, err := service.repo.GetShopVehicleByID(user, notification.VehicleID)
	if err != nil {
		return fmt.Errorf("failed to get vehicle: %w", err)
	}

	isMember, err := service.repo.IsUserMemberOfShop(user, notification.ShopID)
	if err != nil {
		return fmt.Errorf("failed to verify membership: %w", err)
	}

	if !isMember {
		return errors.New("access denied: user is not a member of this shop")
	}

	err = service.repo.DeleteNotificationItem(user, itemID)
	if err != nil {
		return fmt.Errorf("failed to remove notification item: %w", err)
	}

	fieldChanges, err := buildItemRemovalFieldChanges([]model.ShopNotificationItems{*item})
	if err != nil {
		slog.Warn("Failed to build field changes for item removal", "error", err)
		fieldChanges = `{"fields_changed": ["items"], "item_count": 1}`
	}

	service.recordNotificationChange(
		user,
		item.NotificationID,
		item.ShopID,
		notification.VehicleID,
		"items_removed",
		fieldChanges,
		notification.Title,
		notification.Type,
		vehicle.Admin,
	)

	slog.Info("Notification item removed", "user_id", user.UserID, "item_id", itemID, "notification_id", item.NotificationID)
	return nil
}

func (service *ServiceImpl) RemoveNotificationItemList(user *bootstrap.User, itemIDs []string) error {
	if user == nil {
		return errors.New("unauthorized user")
	}

	if len(itemIDs) == 0 {
		return errors.New("no items to remove")
	}

	items, err := service.repo.GetNotificationItemsByIDs(user, itemIDs)
	if err != nil {
		return fmt.Errorf("failed to get notification items: %w", err)
	}

	if len(items) == 0 {
		slog.Warn("No notification items found for deletion", "user_id", user.UserID, "requested_count", len(itemIDs))
		return errors.New("no notification items found")
	}

	firstItem := items[0]
	notification, err := service.repo.GetVehicleNotificationByID(user, firstItem.NotificationID)
	if err != nil {
		return fmt.Errorf("failed to get notification: %w", err)
	}

	vehicle, err := service.repo.GetShopVehicleByID(user, notification.VehicleID)
	if err != nil {
		return fmt.Errorf("failed to get vehicle: %w", err)
	}

	isMember, err := service.repo.IsUserMemberOfShop(user, notification.ShopID)
	if err != nil {
		return fmt.Errorf("failed to verify membership: %w", err)
	}

	if !isMember {
		return errors.New("access denied: user is not a member of this shop")
	}

	for _, item := range items {
		if item.NotificationID != firstItem.NotificationID {
			return errors.New("cannot delete items from multiple notifications in a single operation")
		}
	}

	err = service.repo.DeleteNotificationItemList(user, itemIDs)
	if err != nil {
		return fmt.Errorf("failed to remove notification items: %w", err)
	}

	fieldChanges, err := buildItemRemovalFieldChanges(items)
	if err != nil {
		slog.Warn("Failed to build field changes for item removals", "error", err)
		fieldChanges = fmt.Sprintf(`{"fields_changed": ["items"], "item_count": %d}`, len(items))
	}

	service.recordNotificationChange(
		user,
		firstItem.NotificationID,
		firstItem.ShopID,
		notification.VehicleID,
		"items_removed",
		fieldChanges,
		notification.Title,
		notification.Type,
		vehicle.Admin,
	)

	slog.Info("Notification items removed", "user_id", user.UserID, "count", len(items), "notification_id", firstItem.NotificationID)
	return nil
}

// itemAuditInfo represents item details captured in audit trail
type itemAuditInfo struct {
	Niin         string `json:"niin"`
	Nomenclature string `json:"nomenclature"`
	Quantity     int32  `json:"quantity"`
}

func buildItemAdditionFieldChanges(items []model.ShopNotificationItems) (string, error) {
	type FieldChangesData struct {
		FieldsChanged []string        `json:"fields_changed"`
		ItemCount     int             `json:"item_count"`
		ItemsAdded    []itemAuditInfo `json:"items_added"`
	}

	itemsInfo := make([]itemAuditInfo, len(items))
	for i, item := range items {
		itemsInfo[i] = itemAuditInfo{
			Niin:         item.Niin,
			Nomenclature: item.Nomenclature,
			Quantity:     item.Quantity,
		}
	}

	data := FieldChangesData{
		FieldsChanged: []string{"items"},
		ItemCount:     len(items),
		ItemsAdded:    itemsInfo,
	}

	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("failed to marshal field changes: %w", err)
	}

	return string(jsonBytes), nil
}

func buildItemRemovalFieldChanges(items []model.ShopNotificationItems) (string, error) {
	type FieldChangesData struct {
		FieldsChanged []string        `json:"fields_changed"`
		ItemCount     int             `json:"item_count"`
		ItemsRemoved  []itemAuditInfo `json:"items_removed"`
	}

	itemsInfo := make([]itemAuditInfo, len(items))
	for i, item := range items {
		itemsInfo[i] = itemAuditInfo{
			Niin:         item.Niin,
			Nomenclature: item.Nomenclature,
			Quantity:     item.Quantity,
		}
	}

	data := FieldChangesData{
		FieldsChanged: []string{"items"},
		ItemCount:     len(items),
		ItemsRemoved:  itemsInfo,
	}

	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("failed to marshal field changes: %w", err)
	}

	return string(jsonBytes), nil
}

func (service *ServiceImpl) recordNotificationChange(
	user *bootstrap.User,
	notificationID string,
	shopID string,
	vehicleID string,
	changeType string,
	fieldChanges string,
	notificationTitle string,
	notificationType string,
	vehicleAdmin string,
) {
	change := model.ShopVehicleNotificationChanges{
		NotificationID:    &notificationID,
		ShopID:            shopID,
		VehicleID:         &vehicleID,
		ChangedBy:         &user.UserID,
		ChangeType:        changeType,
		FieldChanges:      fieldChanges,
		NotificationTitle: &notificationTitle,
		NotificationType:  &notificationType,
		VehicleAdmin:      &vehicleAdmin,
	}

	err := service.repo.CreateNotificationChange(user, change)
	if err != nil {
		slog.Warn("Failed to record notification change", "error", err, "notification_id", notificationID, "change_type", changeType)
	}
}
