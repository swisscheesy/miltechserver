package settings

import (
	"errors"
	"fmt"
	"log/slog"
	"miltechserver/api/request"
	"miltechserver/api/shops/shared"
	"miltechserver/bootstrap"
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

// GetShopAdminOnlyListsSetting returns the admin_only_lists setting for a shop
// Any shop member can read this setting
func (service *ServiceImpl) GetShopAdminOnlyListsSetting(user *bootstrap.User, shopID string) (bool, error) {
	if user == nil {
		return false, errors.New("unauthorized user")
	}

	isMember, err := service.auth.IsUserMemberOfShop(user, shopID)
	if err != nil {
		return false, fmt.Errorf("failed to verify membership: %w", err)
	}

	if !isMember {
		return false, errors.New("access denied: user is not a member of this shop")
	}

	adminOnlyLists, err := service.repo.GetShopAdminOnlyListsSetting(shopID)
	if err != nil {
		return false, fmt.Errorf("failed to get admin_only_lists setting: %w", err)
	}

	slog.Info("Shop admin_only_lists setting retrieved", "user_id", user.UserID, "shop_id", shopID, "admin_only_lists", adminOnlyLists)
	return adminOnlyLists, nil
}

// UpdateShopAdminOnlyListsSetting updates the admin_only_lists setting for a shop
// Only shop admins can modify this setting
func (service *ServiceImpl) UpdateShopAdminOnlyListsSetting(user *bootstrap.User, shopID string, adminOnlyLists bool) error {
	if user == nil {
		return errors.New("unauthorized user")
	}

	isAdmin, err := service.auth.IsUserShopAdmin(user, shopID)
	if err != nil {
		return fmt.Errorf("failed to verify admin status: %w", err)
	}

	if !isAdmin {
		return errors.New("access denied: only shop administrators can modify this setting")
	}

	err = service.repo.UpdateShopAdminOnlyListsSetting(shopID, adminOnlyLists)
	if err != nil {
		return fmt.Errorf("failed to update admin_only_lists setting: %w", err)
	}

	slog.Info("Shop admin_only_lists setting updated by admin", "user_id", user.UserID, "shop_id", shopID, "admin_only_lists", adminOnlyLists)
	return nil
}

// IsUserShopAdmin checks if the authenticated user is an admin for the specified shop
func (service *ServiceImpl) IsUserShopAdmin(user *bootstrap.User, shopID string) (bool, error) {
	if user == nil {
		return false, errors.New("unauthorized user")
	}

	isMember, err := service.auth.IsUserMemberOfShop(user, shopID)
	if err != nil {
		return false, fmt.Errorf("failed to verify shop membership: %w", err)
	}

	if !isMember {
		return false, nil
	}

	isAdmin, err := service.auth.IsUserShopAdmin(user, shopID)
	if err != nil {
		return false, fmt.Errorf("failed to verify admin status: %w", err)
	}

	return isAdmin, nil
}

// GetShopSettings returns all settings for a shop
// Any shop member can read settings
func (service *ServiceImpl) GetShopSettings(user *bootstrap.User, shopID string) (*request.ShopSettings, error) {
	if user == nil {
		return nil, errors.New("unauthorized user")
	}

	isMember, err := service.auth.IsUserMemberOfShop(user, shopID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify membership: %w", err)
	}

	if !isMember {
		return nil, errors.New("access denied: user is not a member of this shop")
	}

	settings, err := service.repo.GetShopSettings(shopID)
	if err != nil {
		return nil, fmt.Errorf("failed to get shop settings: %w", err)
	}

	slog.Info("Shop settings retrieved", "user_id", user.UserID, "shop_id", shopID)
	return settings, nil
}

// UpdateShopSettings updates one or more shop settings (admin only)
// Supports partial updates - only provided fields are modified
func (service *ServiceImpl) UpdateShopSettings(user *bootstrap.User, shopID string, updates request.UpdateShopSettingsRequest) (*request.ShopSettings, error) {
	if user == nil {
		return nil, errors.New("unauthorized user")
	}

	isAdmin, err := service.auth.IsUserShopAdmin(user, shopID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify admin status: %w", err)
	}

	if !isAdmin {
		return nil, errors.New("access denied: only shop administrators can modify settings")
	}

	err = service.repo.UpdateShopSettings(shopID, updates)
	if err != nil {
		return nil, fmt.Errorf("failed to update shop settings: %w", err)
	}

	updatedSettings, err := service.repo.GetShopSettings(shopID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch updated settings: %w", err)
	}

	slog.Info("Shop settings updated by admin", "user_id", user.UserID, "shop_id", shopID)
	return updatedSettings, nil
}
