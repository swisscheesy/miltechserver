package notifications

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/response"
	"miltechserver/api/shops/shared"
	"miltechserver/bootstrap"
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

func (service *ServiceImpl) CreateVehicleNotification(user *bootstrap.User, notification model.ShopVehicleNotifications) (*model.ShopVehicleNotifications, error) {
	if user == nil {
		return nil, errors.New("unauthorized user")
	}

	vehicle, err := service.repo.GetShopVehicleByID(user, notification.VehicleID)
	if err != nil {
		return nil, fmt.Errorf("failed to get vehicle: %w", err)
	}

	isMember, err := service.repo.IsUserMemberOfShop(user, vehicle.ShopID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify membership: %w", err)
	}

	if !isMember {
		return nil, errors.New("access denied: user is not a member of this shop")
	}

	notification.ID = uuid.New().String()
	notification.ShopID = vehicle.ShopID
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

	createdNotification, err := service.repo.CreateVehicleNotification(user, notification)
	if err != nil {
		return nil, fmt.Errorf("failed to create vehicle notification: %w", err)
	}

	service.recordNotificationChange(
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

func (service *ServiceImpl) GetVehicleNotifications(user *bootstrap.User, vehicleID string) ([]model.ShopVehicleNotifications, error) {
	if user == nil {
		return nil, errors.New("unauthorized user")
	}

	vehicle, err := service.repo.GetShopVehicleByID(user, vehicleID)
	if err != nil {
		return nil, fmt.Errorf("failed to get vehicle: %w", err)
	}

	isMember, err := service.repo.IsUserMemberOfShop(user, vehicle.ShopID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify membership: %w", err)
	}

	if !isMember {
		return nil, errors.New("access denied: user is not a member of this shop")
	}

	notifications, err := service.repo.GetVehicleNotifications(user, vehicleID)
	if err != nil {
		return nil, fmt.Errorf("failed to get vehicle notifications: %w", err)
	}

	if notifications == nil {
		return []model.ShopVehicleNotifications{}, nil
	}

	return notifications, nil
}

func (service *ServiceImpl) GetVehicleNotificationsWithItems(user *bootstrap.User, vehicleID string) ([]response.VehicleNotificationWithItems, error) {
	if user == nil {
		return nil, errors.New("unauthorized user")
	}

	vehicle, err := service.repo.GetShopVehicleByID(user, vehicleID)
	if err != nil {
		return nil, fmt.Errorf("failed to get vehicle: %w", err)
	}

	isMember, err := service.repo.IsUserMemberOfShop(user, vehicle.ShopID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify membership: %w", err)
	}

	if !isMember {
		return nil, errors.New("access denied: user is not a member of this shop")
	}

	notificationsWithItems, err := service.repo.GetVehicleNotificationsWithItems(user, vehicleID)
	if err != nil {
		return nil, fmt.Errorf("failed to get vehicle notifications with items: %w", err)
	}

	if notificationsWithItems == nil {
		return []response.VehicleNotificationWithItems{}, nil
	}

	return notificationsWithItems, nil
}

func (service *ServiceImpl) GetShopNotifications(user *bootstrap.User, shopID string) ([]model.ShopVehicleNotifications, error) {
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

	notifications, err := service.repo.GetShopNotifications(user, shopID)
	if err != nil {
		return nil, fmt.Errorf("failed to get shop notifications: %w", err)
	}

	if notifications == nil {
		return []model.ShopVehicleNotifications{}, nil
	}

	return notifications, nil
}

func (service *ServiceImpl) GetVehicleNotificationByID(user *bootstrap.User, notificationID string) (*model.ShopVehicleNotifications, error) {
	if user == nil {
		return nil, errors.New("unauthorized user")
	}

	notification, err := service.repo.GetVehicleNotificationByID(user, notificationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get vehicle notification: %w", err)
	}

	isMember, err := service.repo.IsUserMemberOfShop(user, notification.ShopID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify membership: %w", err)
	}

	if !isMember {
		return nil, errors.New("access denied: user is not a member of this shop")
	}

	return notification, nil
}

func (service *ServiceImpl) UpdateVehicleNotification(user *bootstrap.User, notification model.ShopVehicleNotifications) error {
	if user == nil {
		return errors.New("unauthorized user")
	}

	currentNotification, err := service.repo.GetVehicleNotificationByID(user, notification.ID)
	if err != nil {
		return fmt.Errorf("failed to get current notification: %w", err)
	}

	vehicle, err := service.repo.GetShopVehicleByID(user, currentNotification.VehicleID)
	if err != nil {
		return fmt.Errorf("failed to get vehicle: %w", err)
	}

	isMember, err := service.repo.IsUserMemberOfShop(user, currentNotification.ShopID)
	if err != nil {
		return fmt.Errorf("failed to verify membership: %w", err)
	}

	if !isMember {
		return errors.New("access denied: user is not a member of this shop")
	}

	notification.LastUpdated = time.Now()

	fieldChanges, err := buildFieldChanges(currentNotification, &notification)
	if err != nil {
		slog.Warn("Failed to build field changes", "error", err)
		fieldChanges = `{"fields_changed": []}`
	}

	changeType := determineChangeType(currentNotification, &notification)

	err = service.repo.UpdateVehicleNotification(user, notification)
	if err != nil {
		return fmt.Errorf("failed to update vehicle notification: %w", err)
	}

	service.recordNotificationChange(
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

func (service *ServiceImpl) DeleteVehicleNotification(user *bootstrap.User, notificationID string) error {
	if user == nil {
		return errors.New("unauthorized user")
	}

	notification, err := service.repo.GetVehicleNotificationByID(user, notificationID)
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

	service.recordNotificationChange(
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

	err = service.repo.DeleteVehicleNotification(user, notificationID)
	if err != nil {
		return fmt.Errorf("failed to delete vehicle notification: %w", err)
	}

	slog.Info("Vehicle notification deleted", "user_id", user.UserID, "notification_id", notificationID)
	return nil
}

// recordNotificationChange is a helper to record audit trail changes (best-effort)
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

	changeData["fields_changed"] = changedFields

	jsonBytes, err := json.Marshal(changeData)
	if err != nil {
		return "{}", fmt.Errorf("failed to marshal field changes: %w", err)
	}
	return string(jsonBytes), nil
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
