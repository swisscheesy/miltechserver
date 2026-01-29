package core

import (
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

func (service *ServiceImpl) CreateShop(user *bootstrap.User, shop model.Shops) (*model.Shops, error) {
	if user == nil {
		return nil, errors.New("unauthorized user")
	}

	shop.ID = uuid.New().String()
	shop.CreatedBy = user.UserID
	now := time.Now()
	shop.CreatedAt = &now
	shop.UpdatedAt = &now

	createdShop, err := service.repo.CreateShop(user, shop)
	if err != nil {
		slog.Error("Failed to create shop", "error", err, "user_id", user.UserID)
		return nil, fmt.Errorf("failed to create shop: %w", err)
	}

	err = service.repo.AddMemberToShop(user, shop.ID, "admin")
	if err != nil {
		slog.Error("Failed to add creator as admin to shop", "error", err, "user_id", user.UserID, "shop_id", shop.ID)
	}

	slog.Info("Shop created successfully", "user_id", user.UserID, "shop_id", shop.ID, "shop_name", shop.Name)
	return createdShop, nil
}

func (service *ServiceImpl) UpdateShop(user *bootstrap.User, shop model.Shops) (*model.Shops, error) {
	if user == nil {
		return nil, errors.New("unauthorized user")
	}

	isAdmin, err := service.auth.IsUserShopAdmin(user, shop.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify admin status: %w", err)
	}

	if !isAdmin {
		return nil, errors.New("access denied: only shop admins can update shops")
	}

	updatedShop, err := service.repo.UpdateShop(user, shop)
	if err != nil {
		return nil, fmt.Errorf("failed to update shop: %w", err)
	}

	slog.Info("Shop updated successfully", "user_id", user.UserID, "shop_id", shop.ID, "shop_name", shop.Name)
	return updatedShop, nil
}

func (service *ServiceImpl) DeleteShop(user *bootstrap.User, shopID string) error {
	if user == nil {
		return errors.New("unauthorized user")
	}

	isAdmin, err := service.auth.IsUserShopAdmin(user, shopID)
	if err != nil {
		return fmt.Errorf("failed to verify admin status: %w", err)
	}

	if !isAdmin {
		return errors.New("only shop administrators can delete shops")
	}

	return service.deleteShopWithBlobCleanup(user, shopID)
}

func (service *ServiceImpl) GetShopsByUser(user *bootstrap.User) ([]model.Shops, error) {
	if user == nil {
		return nil, errors.New("unauthorized user")
	}

	shops, err := service.repo.GetShopsByUser(user)
	if err != nil {
		return nil, fmt.Errorf("failed to get shops: %w", err)
	}

	if shops == nil {
		return []model.Shops{}, nil
	}

	return shops, nil
}

func (service *ServiceImpl) GetShopByID(user *bootstrap.User, shopID string) (*response.ShopDetailResponse, error) {
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

	shop, err := service.repo.GetShopByID(user, shopID)
	if err != nil {
		return nil, fmt.Errorf("failed to get shop: %w", err)
	}

	return shop, nil
}

func (service *ServiceImpl) GetUserDataWithShops(user *bootstrap.User) (*response.UserShopsResponse, error) {
	if user == nil {
		return nil, errors.New("unauthorized user")
	}

	shopsWithStats, err := service.repo.GetShopsWithStatsForUser(user)
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

// deleteShopWithBlobCleanup is a private helper that deletes a shop and cleans up associated blobs
// This is used by both DeleteShop (admin deletion) and LeaveShop (last member deletion)
func (service *ServiceImpl) deleteShopWithBlobCleanup(user *bootstrap.User, shopID string) error {
	err := service.repo.DeleteShop(user, shopID)
	if err != nil {
		slog.Error("Failed to delete shop", "error", err, "user_id", user.UserID, "shop_id", shopID)
		return fmt.Errorf("failed to delete shop: %w", err)
	}

	err = service.repo.DeleteShopMessageBlobs(shopID)
	if err != nil {
		slog.Warn("Failed to delete shop message blobs during shop deletion",
			"shop_id", shopID,
			"user_id", user.UserID,
			"error", err)
	}

	slog.Info("Shop deleted successfully", "user_id", user.UserID, "shop_id", shopID)
	return nil
}
