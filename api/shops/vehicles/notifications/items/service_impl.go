package items

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/shops/shared"
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

func (service *ServiceImpl) AddNotificationItem(ctx context.Context, user *bootstrap.User, item model.ShopNotificationItems) (*model.ShopNotificationItems, error) {
	ctx = WithAuditCorrelation(ctx)
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := shared.ValidateItemFields(item.Niin, item.Nomenclature, item.Quantity); err != nil {
		return nil, err
	}
	if err := shared.ValidateItemEnrichment(item.Nickname, item.UnitOfMeasure); err != nil {
		return nil, err
	}

	if user == nil {
		return nil, errors.New("unauthorized user")
	}

	notification, err := service.repo.GetVehicleNotificationByID(ctx, user, item.NotificationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get notification: %w", err)
	}

	vehicle, err := service.repo.GetShopVehicleByID(ctx, user, notification.VehicleID)
	if err != nil {
		return nil, fmt.Errorf("failed to get vehicle: %w", err)
	}

	isMember, err := service.repo.IsUserMemberOfShop(ctx, user, notification.ShopID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify membership: %w", err)
	}

	if !isMember {
		return nil, errors.New("access denied: user is not a member of this shop")
	}

	item.ID = uuid.New().String()
	item.ShopID = notification.ShopID
	item.SaveTime = time.Now()

	createdItem, err := service.repo.CreateNotificationItem(ctx, user, item)
	if err != nil {
		return nil, fmt.Errorf("failed to add notification item: %w", err)
	}

	fieldChanges, err := buildItemAdditionFieldChanges([]model.ShopNotificationItems{*createdItem})
	if err != nil {
		slog.Warn("Failed to build field changes for item addition", "error", err)
		fieldChanges = `{"fields_changed": ["items"], "item_count": 1}`
	}

	service.recordNotificationChange(
		ctx,
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

func (service *ServiceImpl) GetNotificationItems(ctx context.Context, user *bootstrap.User, notificationID string) ([]model.ShopNotificationItems, error) {
	if user == nil {
		return nil, errors.New("unauthorized user")
	}

	notification, err := service.repo.GetVehicleNotificationByID(ctx, user, notificationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get notification: %w", err)
	}

	isMember, err := service.repo.IsUserMemberOfShop(ctx, user, notification.ShopID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify membership: %w", err)
	}

	if !isMember {
		return nil, errors.New("access denied: user is not a member of this shop")
	}

	items, err := service.repo.GetNotificationItems(ctx, user, notificationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get notification items: %w", err)
	}

	if items == nil {
		return []model.ShopNotificationItems{}, nil
	}

	return items, nil
}

func (service *ServiceImpl) GetShopNotificationItems(ctx context.Context, user *bootstrap.User, shopID string) ([]model.ShopNotificationItems, error) {
	if user == nil {
		return nil, errors.New("unauthorized user")
	}

	isMember, err := service.repo.IsUserMemberOfShop(ctx, user, shopID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify membership: %w", err)
	}

	if !isMember {
		return nil, errors.New("access denied: user is not a member of this shop")
	}

	items, err := service.repo.GetShopNotificationItems(ctx, user, shopID)
	if err != nil {
		return nil, fmt.Errorf("failed to get shop notification items: %w", err)
	}

	if items == nil {
		return []model.ShopNotificationItems{}, nil
	}

	return items, nil
}

func (service *ServiceImpl) AddNotificationItemList(ctx context.Context, user *bootstrap.User, items []model.ShopNotificationItems) ([]model.ShopNotificationItems, error) {
	ctx = WithAuditCorrelation(ctx)
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	for _, item := range items {
		if err := shared.ValidateItemFields(item.Niin, item.Nomenclature, item.Quantity); err != nil {
			return nil, err
		}
		if err := shared.ValidateItemEnrichment(item.Nickname, item.UnitOfMeasure); err != nil {
			return nil, err
		}
	}

	for _, item := range items {
		if item.NotificationID != items[0].NotificationID {
			return nil, &shared.ValidationError{}
		}
	}

	if user == nil {
		return nil, errors.New("unauthorized user")
	}

	if len(items) == 0 {
		return nil, errors.New("no items to add")
	}

	notification, err := service.repo.GetVehicleNotificationByID(ctx, user, items[0].NotificationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get notification: %w", err)
	}

	vehicle, err := service.repo.GetShopVehicleByID(ctx, user, notification.VehicleID)
	if err != nil {
		return nil, fmt.Errorf("failed to get vehicle: %w", err)
	}

	isMember, err := service.repo.IsUserMemberOfShop(ctx, user, notification.ShopID)
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

	createdItems, err := service.repo.CreateNotificationItemList(ctx, user, items)
	if err != nil {
		return nil, fmt.Errorf("failed to add notification items: %w", err)
	}

	fieldChanges, err := buildItemAdditionFieldChanges(createdItems)
	if err != nil {
		slog.Warn("Failed to build field changes for item additions", "error", err)
		fieldChanges = fmt.Sprintf(`{"fields_changed": ["items"], "item_count": %d}`, len(createdItems))
	}

	service.recordNotificationChange(
		ctx,
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

func (service *ServiceImpl) RemoveNotificationItem(ctx context.Context, user *bootstrap.User, itemID string) error {
	ctx = WithAuditCorrelation(ctx)
	if user == nil {
		return errors.New("unauthorized user")
	}

	item, err := service.repo.GetNotificationItemByID(ctx, user, itemID)
	if err != nil {
		return fmt.Errorf("failed to get notification item: %w", err)
	}

	notification, err := service.repo.GetVehicleNotificationByID(ctx, user, item.NotificationID)
	if err != nil {
		return fmt.Errorf("failed to get notification: %w", err)
	}

	vehicle, err := service.repo.GetShopVehicleByID(ctx, user, notification.VehicleID)
	if err != nil {
		return fmt.Errorf("failed to get vehicle: %w", err)
	}

	isMember, err := service.repo.IsUserMemberOfShop(ctx, user, notification.ShopID)
	if err != nil {
		return fmt.Errorf("failed to verify membership: %w", err)
	}

	if !isMember {
		return errors.New("access denied: user is not a member of this shop")
	}

	err = service.repo.DeleteNotificationItem(ctx, user, itemID)
	if err != nil {
		return fmt.Errorf("failed to remove notification item: %w", err)
	}

	fieldChanges, err := buildItemRemovalFieldChanges([]model.ShopNotificationItems{*item})
	if err != nil {
		slog.Warn("Failed to build field changes for item removal", "error", err)
		fieldChanges = `{"fields_changed": ["items"], "item_count": 1}`
	}

	service.recordNotificationChange(
		ctx,
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

func (service *ServiceImpl) RemoveNotificationItemList(ctx context.Context, user *bootstrap.User, itemIDs []string) (int64, error) {
	ctx = WithAuditCorrelation(ctx)
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	if user == nil || user.UserID == "" {
		return 0, errors.New("unauthorized user")
	}
	if len(itemIDs) == 0 {
		return 0, errors.New("no items to remove")
	}
	// The repository authorizes all persisted survivors under its transaction locks.
	count, err := service.repo.DeleteNotificationItemList(ctx, user, itemIDs)
	if err != nil {
		return 0, fmt.Errorf("failed to remove notification items: %w", err)
	}
	slog.Info("Notification items removed", "user_id", user.UserID, "count", count)
	return count, nil
}

// ItemAuditSnapshot preserves the physical row at the time of the event.
type ItemAuditSnapshot struct {
	ItemID        string  `json:"item_id"`
	Nickname      *string `json:"nickname"`
	UnitOfMeasure *string `json:"unit_of_measure"`
	Niin          string  `json:"niin"`
	Nomenclature  string  `json:"nomenclature"`
	Quantity      int32   `json:"quantity"`
}

func buildItemAdditionFieldChanges(items []model.ShopNotificationItems) (string, error) {
	type FieldChangesData struct {
		FieldsChanged []string            `json:"fields_changed"`
		ItemCount     int                 `json:"item_count"`
		ItemsAdded    []ItemAuditSnapshot `json:"items_added"`
	}

	itemsInfo := make([]ItemAuditSnapshot, len(items))
	for i, item := range items {
		itemsInfo[i] = ItemAuditSnapshot{
			ItemID:        item.ID,
			Nickname:      item.Nickname,
			UnitOfMeasure: item.UnitOfMeasure,
			Niin:          item.Niin,
			Nomenclature:  item.Nomenclature,
			Quantity:      item.Quantity,
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
		FieldsChanged []string            `json:"fields_changed"`
		ItemCount     int                 `json:"item_count"`
		ItemsRemoved  []ItemAuditSnapshot `json:"items_removed"`
	}

	itemsInfo := make([]ItemAuditSnapshot, len(items))
	for i, item := range items {
		itemsInfo[i] = ItemAuditSnapshot{
			ItemID:        item.ID,
			Nickname:      item.Nickname,
			UnitOfMeasure: item.UnitOfMeasure,
			Niin:          item.Niin,
			Nomenclature:  item.Nomenclature,
			Quantity:      item.Quantity,
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
	ctx context.Context,
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
	if _, ok := service.repo.(interface{ OwnsLegacyNotificationAudits() }); ok {
		return
	}
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

	err := service.repo.CreateNotificationChange(ctx, user, change)
	if err != nil {
		WarnLegacyAuditFailure(ctx, changeType, shopID, vehicleID, notificationID, user.UserID, err)
	}
}

// Compatibility wrappers keep existing notification callers on the shared
// server correlation contract used by vehicle deletion too.
func WithAuditCorrelation(ctx context.Context) context.Context {
	return shared.WithAuditCorrelation(ctx)
}
func WarnLegacyAuditFailure(ctx context.Context, operation, shopID, vehicleID, notificationID, actorID string, err error) {
	shared.WarnLegacyAuditFailure(ctx, operation, shopID, vehicleID, notificationID, actorID, err)
}
