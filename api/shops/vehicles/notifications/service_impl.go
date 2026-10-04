package notifications

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/response"
	"miltechserver/api/shops/shared"
	notificationitems "miltechserver/api/shops/vehicles/notifications/items"
	"miltechserver/bootstrap"
	"strings"
	"time"

	"github.com/google/uuid"
)

type ServiceImpl struct {
	repo Repository
	auth shared.ShopAuthorization
}

func NewService(repo Repository, auth shared.ShopAuthorization) *ServiceImpl {
	return &ServiceImpl{
		repo: repo,
		auth: auth,
	}
}

func (service *ServiceImpl) WithAuthorization(auth shared.ShopAuthorization) shared.AuthorizationAware {
	return &ServiceImpl{
		repo: service.repo,
		auth: auth,
	}
}

func (service *ServiceImpl) validateAttachedShopList(ctx context.Context, user *bootstrap.User, shopID string, attachedShopList *string) error {
	if attachedShopList == nil {
		return nil
	}
	if strings.TrimSpace(*attachedShopList) == "" {
		return errors.New("invalid attached_shop_list")
	}

	list, err := service.repo.GetShopListByID(ctx, user, *attachedShopList)
	if err != nil {
		if errors.Is(err, errShopListNotFound) {
			return errors.New("invalid attached_shop_list")
		}
		return fmt.Errorf("failed to get attached shop list: %w", err)
	}
	if list.ShopID != shopID {
		return errors.New("invalid attached_shop_list")
	}
	return nil
}

func (service *ServiceImpl) CreateVehicleNotification(ctx context.Context, user *bootstrap.User, notification model.ShopVehicleNotifications) (*model.ShopVehicleNotifications, error) {
	ctx = notificationitems.WithAuditCorrelation(ctx)
	if err := shared.ValidateNotificationFields(notification.Title, notification.Type); err != nil {
		return nil, err
	}

	if user == nil {
		return nil, errors.New("unauthorized user")
	}

	vehicle, err := service.repo.GetShopVehicleByID(ctx, user, notification.VehicleID)
	if err != nil {
		return nil, fmt.Errorf("failed to get vehicle: %w", err)
	}

	isMember, err := service.repo.IsUserMemberOfShop(ctx, user, vehicle.ShopID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify membership: %w", err)
	}

	if !isMember {
		return nil, errors.New("access denied: user is not a member of this shop")
	}

	notification.ID = uuid.New().String()
	notification.ShopID = vehicle.ShopID
	if err := service.validateAttachedShopList(ctx, user, notification.ShopID, notification.AttachedShopList); err != nil {
		return nil, err
	}

	now := time.Now()
	notification.SaveTime = now
	notification.LastUpdated = now

	validTypes := []string{"M1", "PM", "MW"}
	isValidType := false
	for _, validType := range validTypes {
		if notification.Type == validType {
			isValidType = true
			break
		}
	}
	if !isValidType {
		return nil, errors.New("invalid notification type: must be M1, PM, or MW")
	}

	createdNotification, err := service.repo.CreateVehicleNotification(ctx, user, notification)
	if err != nil {
		return nil, fmt.Errorf("failed to create vehicle notification: %w", err)
	}

	service.recordNotificationChange(
		ctx,
		user,
		notification.ID,
		notification.ShopID,
		notification.VehicleID,
		"create",
		`{"fields_changed": ["created"]}`,
		notification.Title,
		notification.Type,
		vehicle.Admin,
	)

	slog.Info("Vehicle notification created", "user_id", user.UserID, "vehicle_id", notification.VehicleID, "notification_id", notification.ID)
	return createdNotification, nil
}

func (service *ServiceImpl) GetVehicleNotifications(ctx context.Context, user *bootstrap.User, vehicleID string) ([]model.ShopVehicleNotifications, error) {
	if user == nil {
		return nil, errors.New("unauthorized user")
	}

	vehicle, err := service.repo.GetShopVehicleByID(ctx, user, vehicleID)
	if err != nil {
		return nil, fmt.Errorf("failed to get vehicle: %w", err)
	}

	isMember, err := service.repo.IsUserMemberOfShop(ctx, user, vehicle.ShopID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify membership: %w", err)
	}

	if !isMember {
		return nil, errors.New("access denied: user is not a member of this shop")
	}

	notifications, err := service.repo.GetVehicleNotifications(ctx, user, vehicleID)
	if err != nil {
		return nil, fmt.Errorf("failed to get vehicle notifications: %w", err)
	}

	if notifications == nil {
		return []model.ShopVehicleNotifications{}, nil
	}

	return notifications, nil
}

func (service *ServiceImpl) GetVehicleNotificationsWithItems(ctx context.Context, user *bootstrap.User, vehicleID string) ([]response.VehicleNotificationWithItems, error) {
	if user == nil {
		return nil, errors.New("unauthorized user")
	}

	// The repository authorizes and assembles this combined read in one snapshot.
	notificationsWithItems, err := service.repo.GetVehicleNotificationsWithItems(ctx, user, vehicleID)
	if err != nil {
		return nil, fmt.Errorf("failed to get vehicle notifications with items: %w", err)
	}

	if notificationsWithItems == nil {
		return []response.VehicleNotificationWithItems{}, nil
	}

	return notificationsWithItems, nil
}

func (service *ServiceImpl) GetShopNotifications(ctx context.Context, user *bootstrap.User, shopID string) ([]model.ShopVehicleNotifications, error) {
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

	notifications, err := service.repo.GetShopNotifications(ctx, user, shopID)
	if err != nil {
		return nil, fmt.Errorf("failed to get shop notifications: %w", err)
	}

	if notifications == nil {
		return []model.ShopVehicleNotifications{}, nil
	}

	return notifications, nil
}

func (service *ServiceImpl) GetVehicleNotificationByID(ctx context.Context, user *bootstrap.User, notificationID string) (*model.ShopVehicleNotifications, error) {
	if user == nil {
		return nil, errors.New("unauthorized user")
	}

	notification, err := service.repo.GetVehicleNotificationByID(ctx, user, notificationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get vehicle notification: %w", err)
	}

	isMember, err := service.repo.IsUserMemberOfShop(ctx, user, notification.ShopID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify membership: %w", err)
	}

	if !isMember {
		return nil, errors.New("access denied: user is not a member of this shop")
	}

	return notification, nil
}

func (service *ServiceImpl) UpdateVehicleNotification(ctx context.Context, user *bootstrap.User, update VehicleNotificationUpdate) error {
	ctx = notificationitems.WithAuditCorrelation(ctx)
	if err := shared.ValidateNotificationFields(update.Notification.Title, update.Notification.Type); err != nil {
		return err
	}

	if user == nil {
		return errors.New("unauthorized user")
	}

	notification := update.Notification

	currentNotification, err := service.repo.GetVehicleNotificationByID(ctx, user, notification.ID)
	if err != nil {
		return fmt.Errorf("failed to get current notification: %w", err)
	}

	vehicle, err := service.repo.GetShopVehicleByID(ctx, user, currentNotification.VehicleID)
	if err != nil {
		return fmt.Errorf("failed to get vehicle: %w", err)
	}

	isMember, err := service.repo.IsUserMemberOfShop(ctx, user, currentNotification.ShopID)
	if err != nil {
		return fmt.Errorf("failed to verify membership: %w", err)
	}

	if !isMember {
		return errors.New("access denied: user is not a member of this shop")
	}

	if update.AttachedShopListSet {
		if err := service.validateAttachedShopList(ctx, user, currentNotification.ShopID, update.AttachedShopList); err != nil {
			return err
		}
		update.Notification.AttachedShopList = update.AttachedShopList
	} else {
		update.Notification.AttachedShopList = currentNotification.AttachedShopList
	}

	notification = update.Notification
	notification.LastUpdated = time.Now()

	fieldChanges, err := buildFieldChanges(currentNotification, &notification)
	if err != nil {
		slog.Warn("Failed to build field changes", "error", err)
		fieldChanges = `{"fields_changed": []}`
	}

	changeType := determineChangeType(currentNotification, &notification)

	update.Notification = notification
	err = service.repo.UpdateVehicleNotification(ctx, user, update)
	if err != nil {
		return fmt.Errorf("failed to update vehicle notification: %w", err)
	}

	service.recordNotificationChange(
		ctx,
		user,
		notification.ID,
		currentNotification.ShopID,
		currentNotification.VehicleID,
		changeType,
		fieldChanges,
		notification.Title,
		notification.Type,
		vehicle.Admin,
	)

	slog.Info("Vehicle notification updated", "user_id", user.UserID, "notification_id", notification.ID)
	return nil
}

func (service *ServiceImpl) DeleteVehicleNotification(ctx context.Context, user *bootstrap.User, notificationID string) error {
	ctx = notificationitems.WithAuditCorrelation(ctx)
	if user == nil {
		return errors.New("unauthorized user")
	}

	notification, err := service.repo.GetVehicleNotificationByID(ctx, user, notificationID)
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

	service.recordNotificationChange(
		ctx,
		user,
		notificationID,
		notification.ShopID,
		notification.VehicleID,
		"delete",
		`{"fields_changed": ["deleted"]}`,
		notification.Title,
		notification.Type,
		vehicle.Admin,
	)

	err = service.repo.DeleteVehicleNotification(ctx, user, notificationID)
	if err != nil {
		return fmt.Errorf("failed to delete vehicle notification: %w", err)
	}

	slog.Info("Vehicle notification deleted", "user_id", user.UserID, "notification_id", notificationID)
	return nil
}

// recordNotificationChange is a helper to record audit trail changes (best-effort)
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
		notificationitems.WarnLegacyAuditFailure(ctx, changeType, shopID, vehicleID, notificationID, user.UserID, err)
	}
}

func buildFieldChanges(old, new *model.ShopVehicleNotifications) (string, error) {
	changedFields := []string{}
	changeData := make(map[string]interface{})

	if old.Title != new.Title {
		changedFields = append(changedFields, "title")
	}

	if old.Description != new.Description {
		changedFields = append(changedFields, "description")
	}

	if old.Type != new.Type {
		changedFields = append(changedFields, "type")
	}

	if old.Completed != new.Completed {
		changedFields = append(changedFields, "completed")
	}

	if !sameStringPtr(old.AttachedShopList, new.AttachedShopList) {
		changedFields = append(changedFields, "attached_shop_list")
	}

	changeData["fields_changed"] = changedFields

	jsonBytes, err := json.Marshal(changeData)
	if err != nil {
		return "{}", fmt.Errorf("failed to marshal field changes: %w", err)
	}
	return string(jsonBytes), nil
}

func sameStringPtr(left, right *string) bool {
	if left == nil || right == nil {
		return left == right
	}
	return *left == *right
}

func determineChangeType(old, new *model.ShopVehicleNotifications) string {
	if !old.Completed && new.Completed {
		return "complete"
	}
	if old.Completed && !new.Completed {
		return "reopen"
	}
	return "update"
}
