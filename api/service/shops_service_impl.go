package service

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/repository"
	"miltechserver/api/response"
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

func (service *ShopsServiceImpl) UpdateShop(user *bootstrap.User, shop model.Shops) (*model.Shops, error) {
	if user == nil {
		return nil, errors.New("unauthorized user")
	}

	// Check if user is admin or creator of the shop
	isAdmin, err := service.ShopsRepository.IsUserShopAdmin(user, shop.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify admin status: %w", err)
	}

	if !isAdmin {
		return nil, errors.New("access denied: only shop admins can update shops")
	}

	updatedShop, err := service.ShopsRepository.UpdateShop(user, shop)
	if err != nil {
		return nil, fmt.Errorf("failed to update shop: %w", err)
	}

	slog.Info("Shop updated successfully", "user_id", user.UserID, "shop_id", shop.ID, "shop_name", shop.Name)
	return updatedShop, nil
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

func (service *ShopsServiceImpl) GetUserDataWithShops(user *bootstrap.User) (*response.UserShopsResponse, error) {
	if user == nil {
		return nil, errors.New("unauthorized user")
	}

	shopsWithStats, err := service.ShopsRepository.GetShopsWithStatsForUser(user)
	if err != nil {
		return nil, fmt.Errorf("failed to get user shops with stats: %w", err)
	}

	userShopsResponse := &response.UserShopsResponse{
		User:  user,
		Shops: shopsWithStats,
	}

	slog.Info("User data with shops and statistics retrieved", "user_id", user.UserID, "shops_count", len(shopsWithStats))
	return userShopsResponse, nil
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

	// Check the current member count
	memberCount, err := service.ShopsRepository.GetShopMemberCount(user, shopID)
	if err != nil {
		return fmt.Errorf("failed to get member count: %w", err)
	}

	// If this is the only member, delete the entire shop
	if memberCount == 1 {
		err = service.ShopsRepository.DeleteShop(user, shopID)
		if err != nil {
			return fmt.Errorf("failed to delete shop: %w", err)
		}
		slog.Info("Shop deleted as last member left", "user_id", user.UserID, "shop_id", shopID)
	} else {
		// Otherwise, just remove this member
		err = service.ShopsRepository.RemoveMemberFromShop(user, shopID, user.UserID)
		if err != nil {
			return fmt.Errorf("failed to leave shop: %w", err)
		}
		slog.Info("User left shop", "user_id", user.UserID, "shop_id", shopID)
	}

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

func (service *ShopsServiceImpl) GetShopMembers(user *bootstrap.User, shopID string) ([]response.ShopMemberWithUsername, error) {
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
		return []response.ShopMemberWithUsername{}, nil
	}

	return members, nil
}

// Shop Invite Code Operations
func (service *ShopsServiceImpl) GenerateInviteCode(user *bootstrap.User, shopID string) (*model.ShopInviteCodes, error) {
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
		ID:        uuid.New().String(),
		ShopID:    shopID,
		Code:      code,
		CreatedBy: user.UserID,
		IsActive:  func() *bool { b := true; return &b }(),
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

	// Get invite code to check shop ownership
	inviteCode, err := service.ShopsRepository.GetInviteCodeByID(codeID)
	if err != nil {
		return fmt.Errorf("failed to get invite code: %w", err)
	}

	// Check if user is admin of the shop
	isAdmin, err := service.ShopsRepository.IsUserShopAdmin(user, inviteCode.ShopID)
	if err != nil {
		return fmt.Errorf("failed to verify admin status: %w", err)
	}

	if !isAdmin {
		return errors.New("only shop administrators can deactivate invite codes")
	}

	err = service.ShopsRepository.DeactivateInviteCode(user, codeID)
	if err != nil {
		return fmt.Errorf("failed to deactivate invite code: %w", err)
	}

	slog.Info("Invite code deactivated", "user_id", user.UserID, "code_id", codeID)
	return nil
}

func (service *ShopsServiceImpl) DeleteInviteCode(user *bootstrap.User, codeID string) error {
	if user == nil {
		return errors.New("unauthorized user")
	}

	// Get invite code to check shop ownership
	inviteCode, err := service.ShopsRepository.GetInviteCodeByID(codeID)
	if err != nil {
		return fmt.Errorf("failed to get invite code: %w", err)
	}

	// Check if user is admin of the shop
	isAdmin, err := service.ShopsRepository.IsUserShopAdmin(user, inviteCode.ShopID)
	if err != nil {
		return fmt.Errorf("failed to verify admin status: %w", err)
	}

	if !isAdmin {
		return errors.New("only shop administrators can delete invite codes")
	}

	err = service.ShopsRepository.DeleteInviteCode(user, codeID)
	if err != nil {
		return fmt.Errorf("failed to delete invite code: %w", err)
	}

	slog.Info("Invite code deleted", "user_id", user.UserID, "code_id", codeID)
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
	now := time.Now().UTC()
	vehicle.SaveTime = now
	vehicle.LastUpdated = now

	// Handle null/empty values for string fields
	if vehicle.Niin == "" {
		vehicle.Niin = ""
	}
	if vehicle.Model == "" {
		vehicle.Model = ""
	}
	if vehicle.Serial == "" {
		vehicle.Serial = ""
	}
	if vehicle.Comment == "" {
		vehicle.Comment = ""
	}
	if vehicle.Admin == "" {
		vehicle.Admin = ""
	}
	if vehicle.Uoc == "" {
		vehicle.Uoc = "UNK"
	}

	// Handle null values for int fields (already 0 by default in Go, but being explicit)
	if vehicle.Mileage == 0 {
		vehicle.Mileage = 0
	}
	if vehicle.Hours == 0 {
		vehicle.Hours = 0
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

	// Get current vehicle to check permissions and get ShopID
	currentVehicle, err := service.ShopsRepository.GetShopVehicleByID(user, vehicle.ID)
	if err != nil {
		return fmt.Errorf("failed to get current vehicle: %w", err)
	}

	// Populate ShopID from current vehicle to ensure correct vehicle is updated
	vehicle.ShopID = currentVehicle.ShopID

	// Check if user is vehicle creator OR shop admin
	isCreator := currentVehicle.CreatorID == user.UserID
	isAdmin, err := service.ShopsRepository.IsUserShopAdmin(user, currentVehicle.ShopID)
	if err != nil {
		return fmt.Errorf("failed to verify admin status: %w", err)
	}

	if !isCreator && !isAdmin {
		return errors.New("access denied: only vehicle creator or shop admin can update vehicles")
	}

	// Handle null/empty values for string fields
	if vehicle.Niin == "" {
		vehicle.Niin = ""
	}
	if vehicle.Model == "" {
		vehicle.Model = ""
	}
	if vehicle.Serial == "" {
		vehicle.Serial = ""
	}
	if vehicle.Comment == "" {
		vehicle.Comment = ""
	}
	if vehicle.Admin == "" {
		vehicle.Admin = ""
	}
	if vehicle.Uoc == "" {
		vehicle.Uoc = "UNK"
	}

	// Handle null values for int fields (already 0 by default in Go, but being explicit)
	if vehicle.Mileage == 0 {
		vehicle.Mileage = 0
	}
	if vehicle.Hours == 0 {
		vehicle.Hours = 0
	}

	vehicle.LastUpdated = time.Now().UTC()

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

	// Check if user is vehicle creator OR shop admin
	isCreator := vehicle.CreatorID == user.UserID
	isAdmin, err := service.ShopsRepository.IsUserShopAdmin(user, vehicle.ShopID)
	if err != nil {
		return fmt.Errorf("failed to verify admin status: %w", err)
	}

	if !isCreator && !isAdmin {
		return errors.New("access denied: only vehicle creator or shop admin can delete vehicles")
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

func (service *ShopsServiceImpl) GetVehicleNotificationsWithItems(user *bootstrap.User, vehicleID string) ([]response.VehicleNotificationWithItems, error) {
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

	notificationsWithItems, err := service.ShopsRepository.GetVehicleNotificationsWithItems(user, vehicleID)
	if err != nil {
		return nil, fmt.Errorf("failed to get vehicle notifications with items: %w", err)
	}

	if notificationsWithItems == nil {
		return []response.VehicleNotificationWithItems{}, nil
	}

	return notificationsWithItems, nil
}

func (service *ShopsServiceImpl) GetShopNotifications(user *bootstrap.User, shopID string) ([]model.ShopVehicleNotifications, error) {
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

	notifications, err := service.ShopsRepository.GetShopNotifications(user, shopID)
	if err != nil {
		return nil, fmt.Errorf("failed to get shop notifications: %w", err)
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

func (service *ShopsServiceImpl) GetShopNotificationItems(user *bootstrap.User, shopID string) ([]model.ShopNotificationItems, error) {
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

	items, err := service.ShopsRepository.GetShopNotificationItems(user, shopID)
	if err != nil {
		return nil, fmt.Errorf("failed to get shop notification items: %w", err)
	}

	if items == nil {
		return []model.ShopNotificationItems{}, nil
	}

	return items, nil
}

func (service *ShopsServiceImpl) AddNotificationItemList(user *bootstrap.User, items []model.ShopNotificationItems) ([]model.ShopNotificationItems, error) {
	if user == nil {
		return nil, errors.New("unauthorized user")
	}

	if len(items) == 0 {
		return nil, errors.New("no items to add")
	}

	// Get notification to verify access (use first item's notification ID)
	notification, err := service.ShopsRepository.GetVehicleNotificationByID(user, items[0].NotificationID)
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

	// Set IDs and shop ID for all items
	now := time.Now()
	for i := range items {
		items[i].ID = uuid.New().String()
		items[i].ShopID = notification.ShopID
		items[i].SaveTime = now
	}

	createdItems, err := service.ShopsRepository.CreateNotificationItemList(user, items)
	if err != nil {
		return nil, fmt.Errorf("failed to add notification items: %w", err)
	}

	slog.Info("Notification items added", "user_id", user.UserID, "notification_id", items[0].NotificationID, "count", len(createdItems))
	return createdItems, nil
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

// Shop List Operations
func (service *ShopsServiceImpl) CreateShopList(user *bootstrap.User, list model.ShopLists) (*response.ShopListWithUsername, error) {
	if user == nil {
		return nil, errors.New("unauthorized user")
	}

	// Check if user is member of the shop
	isMember, err := service.ShopsRepository.IsUserMemberOfShop(user, list.ShopID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify membership: %w", err)
	}

	if !isMember {
		return nil, errors.New("access denied: user is not a member of this shop")
	}

	list.ID = uuid.New().String()
	list.CreatedBy = user.UserID
	now := time.Now()
	list.CreatedAt = now
	list.UpdatedAt = now

	createdList, err := service.ShopsRepository.CreateShopList(user, list)
	if err != nil {
		return nil, fmt.Errorf("failed to create shop list: %w", err)
	}

	slog.Info("Shop list created", "user_id", user.UserID, "shop_id", list.ShopID, "list_id", list.ID)
	return createdList, nil
}

func (service *ShopsServiceImpl) GetShopLists(user *bootstrap.User, shopID string) ([]response.ShopListWithUsername, error) {
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

	lists, err := service.ShopsRepository.GetShopLists(user, shopID)
	if err != nil {
		return nil, fmt.Errorf("failed to get shop lists with usernames: %w", err)
	}

	if lists == nil {
		return []response.ShopListWithUsername{}, nil
	}

	return lists, nil
}

func (service *ShopsServiceImpl) GetShopListByID(user *bootstrap.User, listID string) (*response.ShopListWithUsername, error) {
	if user == nil {
		return nil, errors.New("unauthorized user")
	}

	list, err := service.ShopsRepository.GetShopListByID(user, listID)
	if err != nil {
		return nil, fmt.Errorf("failed to get shop list: %w", err)
	}

	// Check if user is member of the shop
	isMember, err := service.ShopsRepository.IsUserMemberOfShop(user, list.ShopID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify membership: %w", err)
	}

	if !isMember {
		return nil, errors.New("access denied: user is not a member of this shop")
	}

	return list, nil
}

func (service *ShopsServiceImpl) UpdateShopList(user *bootstrap.User, list model.ShopLists) error {
	if user == nil {
		return errors.New("unauthorized user")
	}

	// Get the current list to verify permissions
	currentList, err := service.ShopsRepository.GetShopListByID(user, list.ID)
	if err != nil {
		return fmt.Errorf("failed to get list: %w", err)
	}

	// Convert to model for helper function
	modelList := &model.ShopLists{
		ID:        currentList.ID,
		ShopID:    currentList.ShopID,
		CreatedBy: currentList.CreatedBy,
	}
	
	// Check if user can modify this list (either they created it or they're an admin)
	canModify, err := service.canUserModifyList(user, modelList)
	if err != nil {
		return fmt.Errorf("failed to verify permissions: %w", err)
	}

	if !canModify {
		return errors.New("access denied: only shop admins or list creator can modify this list")
	}

	list.UpdatedAt = time.Now()

	err = service.ShopsRepository.UpdateShopList(user, list)
	if err != nil {
		return fmt.Errorf("failed to update shop list: %w", err)
	}

	slog.Info("Shop list updated", "user_id", user.UserID, "list_id", list.ID)
	return nil
}

func (service *ShopsServiceImpl) DeleteShopList(user *bootstrap.User, listID string) error {
	if user == nil {
		return errors.New("unauthorized user")
	}

	// Get the list to verify permissions
	list, err := service.ShopsRepository.GetShopListByID(user, listID)
	if err != nil {
		return fmt.Errorf("failed to get list: %w", err)
	}

	// Convert to model for helper function
	modelList := &model.ShopLists{
		ID:        list.ID,
		ShopID:    list.ShopID,
		CreatedBy: list.CreatedBy,
	}
	
	// Check if user can delete this list (either they created it or they're an admin)
	canDelete, err := service.canUserModifyList(user, modelList)
	if err != nil {
		return fmt.Errorf("failed to verify permissions: %w", err)
	}

	if !canDelete {
		return errors.New("access denied: only shop admins or list creator can delete this list")
	}

	err = service.ShopsRepository.DeleteShopList(user, listID)
	if err != nil {
		return fmt.Errorf("failed to delete shop list: %w", err)
	}

	slog.Info("Shop list deleted", "user_id", user.UserID, "list_id", listID)
	return nil
}

// Shop List Item Operations
func (service *ShopsServiceImpl) AddListItem(user *bootstrap.User, item model.ShopListItems) (*response.ShopListItemWithUsername, error) {
	if user == nil {
		return nil, errors.New("unauthorized user")
	}

	// Get the list to verify access
	list, err := service.ShopsRepository.GetShopListByID(user, item.ListID)
	if err != nil {
		return nil, fmt.Errorf("failed to get list: %w", err)
	}

	// Check if user is member of the shop
	isMember, err := service.ShopsRepository.IsUserMemberOfShop(user, list.ShopID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify membership: %w", err)
	}

	if !isMember {
		return nil, errors.New("access denied: user is not a member of this shop")
	}

	item.ID = uuid.New().String()
	item.AddedBy = user.UserID
	now := time.Now()
	item.CreatedAt = now
	item.UpdatedAt = now

	createdItem, err := service.ShopsRepository.AddListItem(user, item)
	if err != nil {
		return nil, fmt.Errorf("failed to add list item: %w", err)
	}

	slog.Info("List item added", "user_id", user.UserID, "list_id", item.ListID, "item_id", item.ID)
	return createdItem, nil
}

func (service *ShopsServiceImpl) GetListItems(user *bootstrap.User, listID string) ([]response.ShopListItemWithUsername, error) {
	if user == nil {
		return nil, errors.New("unauthorized user")
	}

	// Get the list to verify access
	list, err := service.ShopsRepository.GetShopListByID(user, listID)
	if err != nil {
		return nil, fmt.Errorf("failed to get list: %w", err)
	}

	// Check if user is member of the shop
	isMember, err := service.ShopsRepository.IsUserMemberOfShop(user, list.ShopID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify membership: %w", err)
	}

	if !isMember {
		return nil, errors.New("access denied: user is not a member of this shop")
	}

	items, err := service.ShopsRepository.GetListItems(user, listID)
	if err != nil {
		return nil, fmt.Errorf("failed to get list items with usernames: %w", err)
	}

	if items == nil {
		return []response.ShopListItemWithUsername{}, nil
	}

	return items, nil
}

func (service *ShopsServiceImpl) UpdateListItem(user *bootstrap.User, item model.ShopListItems) error {
	if user == nil {
		return errors.New("unauthorized user")
	}

	// Get the current item to verify permissions
	currentItem, err := service.ShopsRepository.GetListItemByID(user, item.ID)
	if err != nil {
		return fmt.Errorf("failed to get item: %w", err)
	}

	// Get the list to verify shop membership
	list, err := service.ShopsRepository.GetShopListByID(user, currentItem.ListID)
	if err != nil {
		return fmt.Errorf("failed to get list: %w", err)
	}

	// Check if user is member of the shop
	isMember, err := service.ShopsRepository.IsUserMemberOfShop(user, list.ShopID)
	if err != nil {
		return fmt.Errorf("failed to verify membership: %w", err)
	}

	if !isMember {
		return errors.New("access denied: user is not a member of this shop")
	}

	item.UpdatedAt = time.Now()

	err = service.ShopsRepository.UpdateListItem(user, item)
	if err != nil {
		return fmt.Errorf("failed to update list item: %w", err)
	}

	slog.Info("List item updated", "user_id", user.UserID, "item_id", item.ID)
	return nil
}

func (service *ShopsServiceImpl) RemoveListItem(user *bootstrap.User, itemID string) error {
	if user == nil {
		return errors.New("unauthorized user")
	}

	// Get the item to verify permissions
	item, err := service.ShopsRepository.GetListItemByID(user, itemID)
	if err != nil {
		return fmt.Errorf("failed to get item: %w", err)
	}

	// Get the list to verify shop membership
	list, err := service.ShopsRepository.GetShopListByID(user, item.ListID)
	if err != nil {
		return fmt.Errorf("failed to get list: %w", err)
	}

	// Check if user is member of the shop
	isMember, err := service.ShopsRepository.IsUserMemberOfShop(user, list.ShopID)
	if err != nil {
		return fmt.Errorf("failed to verify membership: %w", err)
	}

	if !isMember {
		return errors.New("access denied: user is not a member of this shop")
	}

	err = service.ShopsRepository.RemoveListItem(user, itemID)
	if err != nil {
		return fmt.Errorf("failed to remove list item: %w", err)
	}

	slog.Info("List item removed", "user_id", user.UserID, "item_id", itemID)
	return nil
}

func (service *ShopsServiceImpl) AddListItemBatch(user *bootstrap.User, items []model.ShopListItems) ([]response.ShopListItemWithUsername, error) {
	if user == nil {
		return nil, errors.New("unauthorized user")
	}

	if len(items) == 0 {
		return []response.ShopListItemWithUsername{}, errors.New("no items to add")
	}

	// Get the list to verify access (use first item's list ID)
	list, err := service.ShopsRepository.GetShopListByID(user, items[0].ListID)
	if err != nil {
		return nil, fmt.Errorf("failed to get list: %w", err)
	}

	// Check if user is member of the shop
	isMember, err := service.ShopsRepository.IsUserMemberOfShop(user, list.ShopID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify membership: %w", err)
	}

	if !isMember {
		return nil, errors.New("access denied: user is not a member of this shop")
	}

	// Set IDs and metadata for all items
	now := time.Now()
	for i := range items {
		items[i].ID = uuid.New().String()
		items[i].AddedBy = user.UserID
		items[i].CreatedAt = now
		items[i].UpdatedAt = now
	}

	createdItems, err := service.ShopsRepository.AddListItemBatch(user, items)
	if err != nil {
		return nil, fmt.Errorf("failed to add list items: %w", err)
	}

	slog.Info("List items added", "user_id", user.UserID, "list_id", items[0].ListID, "count", len(createdItems))
	return createdItems, nil
}

func (service *ShopsServiceImpl) RemoveListItemBatch(user *bootstrap.User, itemIDs []string) error {
	if user == nil {
		return errors.New("unauthorized user")
	}

	if len(itemIDs) == 0 {
		return errors.New("no items to remove")
	}

	err := service.ShopsRepository.RemoveListItemBatch(user, itemIDs)
	if err != nil {
		return fmt.Errorf("failed to remove list items: %w", err)
	}

	slog.Info("List items removed", "user_id", user.UserID, "count", len(itemIDs))
	return nil
}

// Helper function to check if user can modify a list (they created it or they're a shop admin)
func (service *ShopsServiceImpl) canUserModifyList(user *bootstrap.User, list *model.ShopLists) (bool, error) {
	// If user created the list, they can modify it
	if list.CreatedBy == user.UserID {
		return true, nil
	}

	// Check if user is shop admin
	userRole, err := service.ShopsRepository.GetUserRoleInShop(user, list.ShopID)
	if err != nil {
		return false, fmt.Errorf("failed to get user role: %w", err)
	}

	return userRole == "admin", nil
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
