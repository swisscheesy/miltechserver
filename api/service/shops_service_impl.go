package service

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/repository"
	"miltechserver/bootstrap"
	"strings"
	"time"

	"github.com/google/uuid"
)

type ShopsServiceImpl struct {
	ShopsRepository repository.ShopsRepository
}

func NewShopsServiceImpl(shopsRepository repository.ShopsRepository) *ShopsServiceImpl {
	return &ShopsServiceImpl{ShopsRepository: shopsRepository}
}

// Shop Operations
func (service *ShopsServiceImpl) CreateShop(user *bootstrap.User, shop model.Shops) (*model.Shops, error) {
	if user == nil {
		return nil, errors.New("unauthorized user")
	}

	// Generate UUID for shop
	shop.ID = uuid.New().String()
	shop.CreatedBy = user.UserID
	now := time.Now()
	shop.CreatedAt = &now
	shop.UpdatedAt = &now

	createdShop, err := service.ShopsRepository.CreateShop(user, shop)
	if err != nil {
		slog.Error("Failed to create shop", "error", err, "user_id", user.UserID)
		return nil, fmt.Errorf("failed to create shop: %w", err)
	}

	// Add creator as admin to shop_members
	err = service.ShopsRepository.AddMemberToShop(user, shop.ID, "admin")
	if err != nil {
		slog.Error("Failed to add creator as admin to shop", "error", err, "user_id", user.UserID, "shop_id", shop.ID)
		// Don't return error here as shop was created successfully
	}

	slog.Info("Shop created successfully", "user_id", user.UserID, "shop_id", shop.ID, "shop_name", shop.Name)
	return createdShop, nil
}

func (service *ShopsServiceImpl) DeleteShop(user *bootstrap.User, shopID string) error {
	if user == nil {
		return errors.New("unauthorized user")
	}

	// Check if user is admin of the shop
	isAdmin, err := service.ShopsRepository.IsUserShopAdmin(user, shopID)
	if err != nil {
		return fmt.Errorf("failed to verify admin status: %w", err)
	}

	if !isAdmin {
		return errors.New("only shop administrators can delete shops")
	}

	err = service.ShopsRepository.DeleteShop(user, shopID)
	if err != nil {
		slog.Error("Failed to delete shop", "error", err, "user_id", user.UserID, "shop_id", shopID)
		return fmt.Errorf("failed to delete shop: %w", err)
	}

	slog.Info("Shop deleted successfully", "user_id", user.UserID, "shop_id", shopID)
	return nil
}

func (service *ShopsServiceImpl) GetShopsByUser(user *bootstrap.User) ([]model.Shops, error) {
	if user == nil {
		return nil, errors.New("unauthorized user")
	}

	shops, err := service.ShopsRepository.GetShopsByUser(user)
	if err != nil {
		return nil, fmt.Errorf("failed to get shops: %w", err)
	}

	if shops == nil {
		return []model.Shops{}, nil
	}

	return shops, nil
}

func (service *ShopsServiceImpl) GetShopByID(user *bootstrap.User, shopID string) (*model.Shops, error) {
	if user == nil {
		return nil, errors.New("unauthorized user")
	}

	// Check if user is member of the shop
	isMember, err := service.ShopsRepository.IsUserMemberOfShop(user, shopID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify membership: %w", err)
	}

	if !isMember {
		return nil, errors.New("access denied: user is not a member of this shop")
	}

	shop, err := service.ShopsRepository.GetShopByID(user, shopID)
	if err != nil {
		return nil, fmt.Errorf("failed to get shop: %w", err)
	}

	return shop, nil
}

// Shop Member Operations
func (service *ShopsServiceImpl) JoinShopViaInviteCode(user *bootstrap.User, inviteCode string) error {
	if user == nil {
		return errors.New("unauthorized user")
	}

	// Get invite code details
	code, err := service.ShopsRepository.GetInviteCodeByCode(inviteCode)
	if err != nil {
		return fmt.Errorf("invalid invite code: %w", err)
	}

	// Check if code is active
	if code.IsActive != nil && !*code.IsActive {
		return errors.New("invite code is inactive")
	}

	// Check if code has expired
	if code.ExpiresAt != nil && time.Now().After(*code.ExpiresAt) {
		return errors.New("invite code has expired")
	}

	// Check if code has reached max uses
	if code.MaxUses != nil && code.CurrentUses != nil && *code.CurrentUses >= *code.MaxUses {
		return errors.New("invite code has reached maximum uses")
	}

	// Check if user is already a member
	isMember, err := service.ShopsRepository.IsUserMemberOfShop(user, code.ShopID)
	if err != nil {
		return fmt.Errorf("failed to check membership: %w", err)
	}

	if isMember {
		return errors.New("user is already a member of this shop")
	}

	// Add user to shop as regular member
	err = service.ShopsRepository.AddMemberToShop(user, code.ShopID, "member")
	if err != nil {
		return fmt.Errorf("failed to add member to shop: %w", err)
	}

	// Increment invite code usage
	err = service.ShopsRepository.IncrementInviteCodeUsage(code.ID)
	if err != nil {
		slog.Error("Failed to increment invite code usage", "error", err, "code_id", code.ID)
		// Don't return error as user was successfully added
	}

	slog.Info("User joined shop via invite code", "user_id", user.UserID, "shop_id", code.ShopID, "invite_code", inviteCode)
	return nil
}

func (service *ShopsServiceImpl) LeaveShop(user *bootstrap.User, shopID string) error {
	if user == nil {
		return errors.New("unauthorized user")
	}

	// Check if user is member of the shop
	isMember, err := service.ShopsRepository.IsUserMemberOfShop(user, shopID)
	if err != nil {
		return fmt.Errorf("failed to verify membership: %w", err)
	}

	if !isMember {
		return errors.New("user is not a member of this shop")
	}

	err = service.ShopsRepository.RemoveMemberFromShop(user, shopID, user.UserID)
	if err != nil {
		return fmt.Errorf("failed to leave shop: %w", err)
	}

	slog.Info("User left shop", "user_id", user.UserID, "shop_id", shopID)
	return nil
}

func (service *ShopsServiceImpl) RemoveMemberFromShop(user *bootstrap.User, shopID string, targetUserID string) error {
	if user == nil {
		return errors.New("unauthorized user")
	}

	// Check if user is admin of the shop
	isAdmin, err := service.ShopsRepository.IsUserShopAdmin(user, shopID)
	if err != nil {
		return fmt.Errorf("failed to verify admin status: %w", err)
	}

	if !isAdmin {
		return errors.New("only shop administrators can remove members")
	}

	// Prevent self-removal via this endpoint (use leave shop instead)
	if user.UserID == targetUserID {
		return errors.New("use leave shop endpoint to remove yourself")
	}

	err = service.ShopsRepository.RemoveMemberFromShop(user, shopID, targetUserID)
	if err != nil {
		return fmt.Errorf("failed to remove member: %w", err)
	}

	slog.Info("Member removed from shop", "admin_user_id", user.UserID, "removed_user_id", targetUserID, "shop_id", shopID)
	return nil
}

func (service *ShopsServiceImpl) GetShopMembers(user *bootstrap.User, shopID string) ([]model.ShopMembers, error) {
	if user == nil {
		return nil, errors.New("unauthorized user")
	}

	// Check if user is member of the shop
	isMember, err := service.ShopsRepository.IsUserMemberOfShop(user, shopID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify membership: %w", err)
	}

	if !isMember {
		return nil, errors.New("access denied: user is not a member of this shop")
	}

	members, err := service.ShopsRepository.GetShopMembers(user, shopID)
	if err != nil {
		return nil, fmt.Errorf("failed to get shop members: %w", err)
	}

	if members == nil {
		return []model.ShopMembers{}, nil
	}

	return members, nil
}

// Shop Invite Code Operations
func (service *ShopsServiceImpl) GenerateInviteCode(user *bootstrap.User, shopID string, maxUses *int32, expiresAt *string) (*model.ShopInviteCodes, error) {
	if user == nil {
		return nil, errors.New("unauthorized user")
	}

	// Check if user is member of the shop
	isMember, err := service.ShopsRepository.IsUserMemberOfShop(user, shopID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify membership: %w", err)
	}

	if !isMember {
		return nil, errors.New("access denied: user is not a member of this shop")
	}

	// Generate a random invite code (max 9 characters)
	code, err := service.generateShortCode()
	if err != nil {
		return nil, fmt.Errorf("failed to generate invite code: %w", err)
	}

	inviteCode := model.ShopInviteCodes{
		ID:          uuid.New().String(),
		ShopID:      shopID,
		Code:        code,
		CreatedBy:   user.UserID,
		MaxUses:     maxUses,
		CurrentUses: func() *int32 { i := int32(0); return &i }(),
		IsActive:    func() *bool { b := true; return &b }(),
	}

	// Parse expiration date if provided
	if expiresAt != nil && *expiresAt != "" {
		expTime, err := time.Parse(time.RFC3339, *expiresAt)
		if err != nil {
			return nil, fmt.Errorf("invalid expiration date format: %w", err)
		}
		inviteCode.ExpiresAt = &expTime
	}

	now := time.Now()
	inviteCode.CreatedAt = &now

	createdCode, err := service.ShopsRepository.CreateInviteCode(user, inviteCode)
	if err != nil {
		return nil, fmt.Errorf("failed to create invite code: %w", err)
	}

	slog.Info("Invite code generated", "user_id", user.UserID, "shop_id", shopID, "code", code)
	return createdCode, nil
}

func (service *ShopsServiceImpl) GetInviteCodesByShop(user *bootstrap.User, shopID string) ([]model.ShopInviteCodes, error) {
	if user == nil {
		return nil, errors.New("unauthorized user")
	}

	// Check if user is member of the shop
	isMember, err := service.ShopsRepository.IsUserMemberOfShop(user, shopID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify membership: %w", err)
	}

	if !isMember {
		return nil, errors.New("access denied: user is not a member of this shop")
	}

	codes, err := service.ShopsRepository.GetInviteCodesByShop(user, shopID)
	if err != nil {
		return nil, fmt.Errorf("failed to get invite codes: %w", err)
	}

	if codes == nil {
		return []model.ShopInviteCodes{}, nil
	}

	return codes, nil
}

func (service *ShopsServiceImpl) DeactivateInviteCode(user *bootstrap.User, codeID string) error {
	if user == nil {
		return errors.New("unauthorized user")
	}

	err := service.ShopsRepository.DeactivateInviteCode(user, codeID)
	if err != nil {
		return fmt.Errorf("failed to deactivate invite code: %w", err)
	}

	slog.Info("Invite code deactivated", "user_id", user.UserID, "code_id", codeID)
	return nil
}

// Shop Message Operations
func (service *ShopsServiceImpl) CreateShopMessage(user *bootstrap.User, message model.ShopMessages) (*model.ShopMessages, error) {
	if user == nil {
		return nil, errors.New("unauthorized user")
	}

	// Check if user is member of the shop
	isMember, err := service.ShopsRepository.IsUserMemberOfShop(user, message.ShopID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify membership: %w", err)
	}

	if !isMember {
		return nil, errors.New("access denied: user is not a member of this shop")
	}

	message.ID = uuid.New().String()
	message.UserID = user.UserID
	now := time.Now()
	message.CreatedAt = &now
	message.UpdatedAt = &now
	message.IsEdited = func() *bool { b := false; return &b }()

	createdMessage, err := service.ShopsRepository.CreateShopMessage(user, message)
	if err != nil {
		return nil, fmt.Errorf("failed to create shop message: %w", err)
	}

	slog.Info("Shop message created", "user_id", user.UserID, "shop_id", message.ShopID, "message_id", message.ID)
	return createdMessage, nil
}

func (service *ShopsServiceImpl) GetShopMessages(user *bootstrap.User, shopID string) ([]model.ShopMessages, error) {
	if user == nil {
		return nil, errors.New("unauthorized user")
	}

	// Check if user is member of the shop
	isMember, err := service.ShopsRepository.IsUserMemberOfShop(user, shopID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify membership: %w", err)
	}

	if !isMember {
		return nil, errors.New("access denied: user is not a member of this shop")
	}

	messages, err := service.ShopsRepository.GetShopMessages(user, shopID)
	if err != nil {
		return nil, fmt.Errorf("failed to get shop messages: %w", err)
	}

	if messages == nil {
		return []model.ShopMessages{}, nil
	}

	return messages, nil
}

func (service *ShopsServiceImpl) UpdateShopMessage(user *bootstrap.User, message model.ShopMessages) error {
	if user == nil {
		return errors.New("unauthorized user")
	}

	// Only the message creator can update their message
	message.UserID = user.UserID
	now := time.Now()
	message.UpdatedAt = &now
	message.IsEdited = func() *bool { b := true; return &b }()

	err := service.ShopsRepository.UpdateShopMessage(user, message)
	if err != nil {
		return fmt.Errorf("failed to update shop message: %w", err)
	}

	slog.Info("Shop message updated", "user_id", user.UserID, "message_id", message.ID)
	return nil
}

func (service *ShopsServiceImpl) DeleteShopMessage(user *bootstrap.User, messageID string) error {
	if user == nil {
		return errors.New("unauthorized user")
	}

	err := service.ShopsRepository.DeleteShopMessage(user, messageID)
	if err != nil {
		return fmt.Errorf("failed to delete shop message: %w", err)
	}

	slog.Info("Shop message deleted", "user_id", user.UserID, "message_id", messageID)
	return nil
}

// Shop Vehicle Operations
func (service *ShopsServiceImpl) CreateShopVehicle(user *bootstrap.User, vehicle model.ShopVehicle) (*model.ShopVehicle, error) {
	if user == nil {
		return nil, errors.New("unauthorized user")
	}

	// Check if user is member of the shop
	isMember, err := service.ShopsRepository.IsUserMemberOfShop(user, vehicle.ShopID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify membership: %w", err)
	}

	if !isMember {
		return nil, errors.New("access denied: user is not a member of this shop")
	}

	vehicle.ID = uuid.New().String()
	vehicle.CreatorID = user.UserID
	vehicle.Admin = user.UserID // Creator becomes the admin
	now := time.Now()
	vehicle.SaveTime = now
	vehicle.LastUpdated = now

	if vehicle.Uoc == "" {
		vehicle.Uoc = "UNK"
	}

	createdVehicle, err := service.ShopsRepository.CreateShopVehicle(user, vehicle)
	if err != nil {
		return nil, fmt.Errorf("failed to create shop vehicle: %w", err)
	}

	slog.Info("Shop vehicle created", "user_id", user.UserID, "shop_id", vehicle.ShopID, "vehicle_id", vehicle.ID)
	return createdVehicle, nil
}

func (service *ShopsServiceImpl) GetShopVehicles(user *bootstrap.User, shopID string) ([]model.ShopVehicle, error) {
	if user == nil {
		return nil, errors.New("unauthorized user")
	}

	// Check if user is member of the shop
	isMember, err := service.ShopsRepository.IsUserMemberOfShop(user, shopID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify membership: %w", err)
	}

	if !isMember {
		return nil, errors.New("access denied: user is not a member of this shop")
	}

	vehicles, err := service.ShopsRepository.GetShopVehicles(user, shopID)
	if err != nil {
		return nil, fmt.Errorf("failed to get shop vehicles: %w", err)
	}

	if vehicles == nil {
		return []model.ShopVehicle{}, nil
	}

	return vehicles, nil
}

func (service *ShopsServiceImpl) GetShopVehicleByID(user *bootstrap.User, vehicleID string) (*model.ShopVehicle, error) {
	if user == nil {
		return nil, errors.New("unauthorized user")
	}

	vehicle, err := service.ShopsRepository.GetShopVehicleByID(user, vehicleID)
	if err != nil {
		return nil, fmt.Errorf("failed to get shop vehicle: %w", err)
	}

	// Check if user is member of the shop
	isMember, err := service.ShopsRepository.IsUserMemberOfShop(user, vehicle.ShopID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify membership: %w", err)
	}

	if !isMember {
		return nil, errors.New("access denied: user is not a member of this shop")
	}

	return vehicle, nil
}

func (service *ShopsServiceImpl) UpdateShopVehicle(user *bootstrap.User, vehicle model.ShopVehicle) error {
	if user == nil {
		return errors.New("unauthorized user")
	}

	// Get current vehicle to check permissions
	currentVehicle, err := service.ShopsRepository.GetShopVehicleByID(user, vehicle.ID)
	if err != nil {
		return fmt.Errorf("failed to get current vehicle: %w", err)
	}

	// Check if user is member of the shop
	isMember, err := service.ShopsRepository.IsUserMemberOfShop(user, currentVehicle.ShopID)
	if err != nil {
		return fmt.Errorf("failed to verify membership: %w", err)
	}

	if !isMember {
		return errors.New("access denied: user is not a member of this shop")
	}

	vehicle.LastUpdated = time.Now()

	err = service.ShopsRepository.UpdateShopVehicle(user, vehicle)
	if err != nil {
		return fmt.Errorf("failed to update shop vehicle: %w", err)
	}

	slog.Info("Shop vehicle updated", "user_id", user.UserID, "vehicle_id", vehicle.ID)
	return nil
}

func (service *ShopsServiceImpl) DeleteShopVehicle(user *bootstrap.User, vehicleID string) error {
	if user == nil {
		return errors.New("unauthorized user")
	}

	// Get vehicle to check permissions
	vehicle, err := service.ShopsRepository.GetShopVehicleByID(user, vehicleID)
	if err != nil {
		return fmt.Errorf("failed to get vehicle: %w", err)
	}

	// Check if user is member of the shop
	isMember, err := service.ShopsRepository.IsUserMemberOfShop(user, vehicle.ShopID)
	if err != nil {
		return fmt.Errorf("failed to verify membership: %w", err)
	}

	if !isMember {
		return errors.New("access denied: user is not a member of this shop")
	}

	err = service.ShopsRepository.DeleteShopVehicle(user, vehicleID)
	if err != nil {
		return fmt.Errorf("failed to delete shop vehicle: %w", err)
	}

	slog.Info("Shop vehicle deleted", "user_id", user.UserID, "vehicle_id", vehicleID)
	return nil
}

// Shop Vehicle Notification Operations
func (service *ShopsServiceImpl) CreateVehicleNotification(user *bootstrap.User, notification model.ShopVehicleNotifications) (*model.ShopVehicleNotifications, error) {
	if user == nil {
		return nil, errors.New("unauthorized user")
	}

	// Get vehicle to verify access and get shop ID
	vehicle, err := service.ShopsRepository.GetShopVehicleByID(user, notification.VehicleID)
	if err != nil {
		return nil, fmt.Errorf("failed to get vehicle: %w", err)
	}

	// Check if user is member of the shop
	isMember, err := service.ShopsRepository.IsUserMemberOfShop(user, vehicle.ShopID)
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

	// Validate notification type
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

	createdNotification, err := service.ShopsRepository.CreateVehicleNotification(user, notification)
	if err != nil {
		return nil, fmt.Errorf("failed to create vehicle notification: %w", err)
	}

	slog.Info("Vehicle notification created", "user_id", user.UserID, "vehicle_id", notification.VehicleID, "notification_id", notification.ID)
	return createdNotification, nil
}

func (service *ShopsServiceImpl) GetVehicleNotifications(user *bootstrap.User, vehicleID string) ([]model.ShopVehicleNotifications, error) {
	if user == nil {
		return nil, errors.New("unauthorized user")
	}

	// Get vehicle to verify access
	vehicle, err := service.ShopsRepository.GetShopVehicleByID(user, vehicleID)
	if err != nil {
		return nil, fmt.Errorf("failed to get vehicle: %w", err)
	}

	// Check if user is member of the shop
	isMember, err := service.ShopsRepository.IsUserMemberOfShop(user, vehicle.ShopID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify membership: %w", err)
	}

	if !isMember {
		return nil, errors.New("access denied: user is not a member of this shop")
	}

	notifications, err := service.ShopsRepository.GetVehicleNotifications(user, vehicleID)
	if err != nil {
		return nil, fmt.Errorf("failed to get vehicle notifications: %w", err)
	}

	if notifications == nil {
		return []model.ShopVehicleNotifications{}, nil
	}

	return notifications, nil
}

func (service *ShopsServiceImpl) GetVehicleNotificationByID(user *bootstrap.User, notificationID string) (*model.ShopVehicleNotifications, error) {
	if user == nil {
		return nil, errors.New("unauthorized user")
	}

	notification, err := service.ShopsRepository.GetVehicleNotificationByID(user, notificationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get vehicle notification: %w", err)
	}

	// Check if user is member of the shop
	isMember, err := service.ShopsRepository.IsUserMemberOfShop(user, notification.ShopID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify membership: %w", err)
	}

	if !isMember {
		return nil, errors.New("access denied: user is not a member of this shop")
	}

	return notification, nil
}

func (service *ShopsServiceImpl) UpdateVehicleNotification(user *bootstrap.User, notification model.ShopVehicleNotifications) error {
	if user == nil {
		return errors.New("unauthorized user")
	}

	// Get current notification to check permissions
	currentNotification, err := service.ShopsRepository.GetVehicleNotificationByID(user, notification.ID)
	if err != nil {
		return fmt.Errorf("failed to get current notification: %w", err)
	}

	// Check if user is member of the shop
	isMember, err := service.ShopsRepository.IsUserMemberOfShop(user, currentNotification.ShopID)
	if err != nil {
		return fmt.Errorf("failed to verify membership: %w", err)
	}

	if !isMember {
		return errors.New("access denied: user is not a member of this shop")
	}

	notification.LastUpdated = time.Now()

	err = service.ShopsRepository.UpdateVehicleNotification(user, notification)
	if err != nil {
		return fmt.Errorf("failed to update vehicle notification: %w", err)
	}

	slog.Info("Vehicle notification updated", "user_id", user.UserID, "notification_id", notification.ID)
	return nil
}

func (service *ShopsServiceImpl) DeleteVehicleNotification(user *bootstrap.User, notificationID string) error {
	if user == nil {
		return errors.New("unauthorized user")
	}

	// Get notification to check permissions
	notification, err := service.ShopsRepository.GetVehicleNotificationByID(user, notificationID)
	if err != nil {
		return fmt.Errorf("failed to get notification: %w", err)
	}

	// Check if user is member of the shop
	isMember, err := service.ShopsRepository.IsUserMemberOfShop(user, notification.ShopID)
	if err != nil {
		return fmt.Errorf("failed to verify membership: %w", err)
	}

	if !isMember {
		return errors.New("access denied: user is not a member of this shop")
	}

	err = service.ShopsRepository.DeleteVehicleNotification(user, notificationID)
	if err != nil {
		return fmt.Errorf("failed to delete vehicle notification: %w", err)
	}

	slog.Info("Vehicle notification deleted", "user_id", user.UserID, "notification_id", notificationID)
	return nil
}

// Shop Notification Item Operations
func (service *ShopsServiceImpl) AddNotificationItem(user *bootstrap.User, item model.ShopNotificationItems) (*model.ShopNotificationItems, error) {
	if user == nil {
		return nil, errors.New("unauthorized user")
	}

	// Get notification to verify access
	notification, err := service.ShopsRepository.GetVehicleNotificationByID(user, item.NotificationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get notification: %w", err)
	}

	// Check if user is member of the shop
	isMember, err := service.ShopsRepository.IsUserMemberOfShop(user, notification.ShopID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify membership: %w", err)
	}

	if !isMember {
		return nil, errors.New("access denied: user is not a member of this shop")
	}

	item.ID = uuid.New().String()
	item.ShopID = notification.ShopID
	item.SaveTime = time.Now()

	createdItem, err := service.ShopsRepository.CreateNotificationItem(user, item)
	if err != nil {
		return nil, fmt.Errorf("failed to add notification item: %w", err)
	}

	slog.Info("Notification item added", "user_id", user.UserID, "notification_id", item.NotificationID, "item_id", item.ID)
	return createdItem, nil
}

func (service *ShopsServiceImpl) GetNotificationItems(user *bootstrap.User, notificationID string) ([]model.ShopNotificationItems, error) {
	if user == nil {
		return nil, errors.New("unauthorized user")
	}

	// Get notification to verify access
	notification, err := service.ShopsRepository.GetVehicleNotificationByID(user, notificationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get notification: %w", err)
	}

	// Check if user is member of the shop
	isMember, err := service.ShopsRepository.IsUserMemberOfShop(user, notification.ShopID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify membership: %w", err)
	}

	if !isMember {
		return nil, errors.New("access denied: user is not a member of this shop")
	}

	items, err := service.ShopsRepository.GetNotificationItems(user, notificationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get notification items: %w", err)
	}

	if items == nil {
		return []model.ShopNotificationItems{}, nil
	}

	return items, nil
}

func (service *ShopsServiceImpl) AddNotificationItemList(user *bootstrap.User, items []model.ShopNotificationItems) error {
	if user == nil {
		return errors.New("unauthorized user")
	}

	if len(items) == 0 {
		return errors.New("no items to add")
	}

	// Get notification to verify access (use first item's notification ID)
	notification, err := service.ShopsRepository.GetVehicleNotificationByID(user, items[0].NotificationID)
	if err != nil {
		return fmt.Errorf("failed to get notification: %w", err)
	}

	// Check if user is member of the shop
	isMember, err := service.ShopsRepository.IsUserMemberOfShop(user, notification.ShopID)
	if err != nil {
		return fmt.Errorf("failed to verify membership: %w", err)
	}

	if !isMember {
		return errors.New("access denied: user is not a member of this shop")
	}

	// Set IDs and shop ID for all items
	now := time.Now()
	for i := range items {
		items[i].ID = uuid.New().String()
		items[i].ShopID = notification.ShopID
		items[i].SaveTime = now
	}

	err = service.ShopsRepository.CreateNotificationItemList(user, items)
	if err != nil {
		return fmt.Errorf("failed to add notification items: %w", err)
	}

	slog.Info("Notification items added", "user_id", user.UserID, "notification_id", items[0].NotificationID, "count", len(items))
	return nil
}

func (service *ShopsServiceImpl) RemoveNotificationItem(user *bootstrap.User, itemID string) error {
	if user == nil {
		return errors.New("unauthorized user")
	}

	err := service.ShopsRepository.DeleteNotificationItem(user, itemID)
	if err != nil {
		return fmt.Errorf("failed to remove notification item: %w", err)
	}

	slog.Info("Notification item removed", "user_id", user.UserID, "item_id", itemID)
	return nil
}

func (service *ShopsServiceImpl) RemoveNotificationItemList(user *bootstrap.User, itemIDs []string) error {
	if user == nil {
		return errors.New("unauthorized user")
	}

	if len(itemIDs) == 0 {
		return errors.New("no items to remove")
	}

	err := service.ShopsRepository.DeleteNotificationItemList(user, itemIDs)
	if err != nil {
		return fmt.Errorf("failed to remove notification items: %w", err)
	}

	slog.Info("Notification items removed", "user_id", user.UserID, "count", len(itemIDs))
	return nil
}

// Helper function to generate a short invite code
func (service *ShopsServiceImpl) generateShortCode() (string, error) {
	// Generate 4 random bytes to create an 8-character hex string
	bytes := make([]byte, 4)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}

	// Convert to hex and take first 8 characters, then uppercase
	code := strings.ToUpper(hex.EncodeToString(bytes))

	// Ensure it's exactly 8 characters (within the 9 character limit)
	if len(code) > 8 {
		code = code[:8]
	}

	return code, nil
}
