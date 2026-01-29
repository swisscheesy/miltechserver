package lists

import (
	"errors"
	"fmt"
	"log/slog"
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/response"
	"miltechserver/api/shops/settings"
	"miltechserver/api/shops/shared"
	"miltechserver/bootstrap"
	"time"

	"github.com/google/uuid"
)

type ServiceImpl struct {
	repo         Repository
	settingsRepo settings.Repository
	auth         shared.ShopAuthorization
}

func NewService(repo Repository, settingsRepo settings.Repository, auth shared.ShopAuthorization) *ServiceImpl {
	return &ServiceImpl{
		repo:         repo,
		settingsRepo: settingsRepo,
		auth:         auth,
	}
}

func (service *ServiceImpl) CreateShopList(user *bootstrap.User, list model.ShopLists) (*response.ShopListWithUsername, error) {
	if user == nil {
		return nil, errors.New("unauthorized user")
	}

	isMember, err := service.auth.IsUserMemberOfShop(user, list.ShopID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify membership: %w", err)
	}

	if !isMember {
		return nil, errors.New("access denied: user is not a member of this shop")
	}

	canModify, err := service.canUserModifyListWithAdminOnlyCheck(user, list.ShopID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify list modification permissions: %w", err)
	}
	if !canModify {
		return nil, errors.New("access denied: insufficient permissions to create lists")
	}

	list.ID = uuid.New().String()
	list.CreatedBy = user.UserID
	now := time.Now()
	list.CreatedAt = now
	list.UpdatedAt = now

	createdList, err := service.repo.CreateShopList(user, list)
	if err != nil {
		return nil, fmt.Errorf("failed to create shop list: %w", err)
	}

	slog.Info("Shop list created", "user_id", user.UserID, "shop_id", list.ShopID, "list_id", list.ID)
	return createdList, nil
}

func (service *ServiceImpl) GetShopLists(user *bootstrap.User, shopID string) ([]response.ShopListWithUsername, error) {
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

	lists, err := service.repo.GetShopLists(user, shopID)
	if err != nil {
		return nil, fmt.Errorf("failed to get shop lists with usernames: %w", err)
	}

	if lists == nil {
		return []response.ShopListWithUsername{}, nil
	}

	return lists, nil
}

func (service *ServiceImpl) GetShopListByID(user *bootstrap.User, listID string) (*response.ShopListWithUsername, error) {
	if user == nil {
		return nil, errors.New("unauthorized user")
	}

	list, err := service.repo.GetShopListByID(user, listID)
	if err != nil {
		return nil, fmt.Errorf("failed to get shop list: %w", err)
	}

	isMember, err := service.auth.IsUserMemberOfShop(user, list.ShopID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify membership: %w", err)
	}

	if !isMember {
		return nil, errors.New("access denied: user is not a member of this shop")
	}

	return list, nil
}

func (service *ServiceImpl) UpdateShopList(user *bootstrap.User, list model.ShopLists) error {
	if user == nil {
		return errors.New("unauthorized user")
	}

	currentList, err := service.repo.GetShopListByID(user, list.ID)
	if err != nil {
		return fmt.Errorf("failed to get list: %w", err)
	}

	canModify, err := service.canUserModifyListWithAdminOnlyCheck(user, currentList.ShopID)
	if err != nil {
		return fmt.Errorf("failed to verify list modification permissions: %w", err)
	}
	if !canModify {
		return errors.New("access denied: insufficient permissions to modify lists")
	}

	list.UpdatedAt = time.Now()

	err = service.repo.UpdateShopList(user, list)
	if err != nil {
		return fmt.Errorf("failed to update shop list: %w", err)
	}

	slog.Info("Shop list updated", "user_id", user.UserID, "list_id", list.ID)
	return nil
}

func (service *ServiceImpl) DeleteShopList(user *bootstrap.User, listID string) error {
	if user == nil {
		return errors.New("unauthorized user")
	}

	list, err := service.repo.GetShopListByID(user, listID)
	if err != nil {
		return fmt.Errorf("failed to get list: %w", err)
	}

	canDelete, err := service.canUserModifyListWithAdminOnlyCheck(user, list.ShopID)
	if err != nil {
		return fmt.Errorf("failed to verify list modification permissions: %w", err)
	}
	if !canDelete {
		return errors.New("access denied: insufficient permissions to delete lists")
	}

	err = service.repo.DeleteShopList(user, listID)
	if err != nil {
		return fmt.Errorf("failed to delete shop list: %w", err)
	}

	slog.Info("Shop list deleted", "user_id", user.UserID, "list_id", listID)
	return nil
}

// canUserModifyListWithAdminOnlyCheck checks if user can modify lists based on shop's admin_only_lists setting
// If admin_only_lists is true, only shop admins can modify lists
// If admin_only_lists is false, all shop members can modify lists
func (service *ServiceImpl) canUserModifyListWithAdminOnlyCheck(user *bootstrap.User, shopID string) (bool, error) {
	adminOnlyLists, err := service.settingsRepo.GetShopAdminOnlyListsSetting(shopID)
	if err != nil {
		return false, fmt.Errorf("failed to get admin_only_lists setting: %w", err)
	}

	if !adminOnlyLists {
		return true, nil
	}

	isAdmin, err := service.auth.IsUserShopAdmin(user, shopID)
	if err != nil {
		return false, fmt.Errorf("failed to verify admin status: %w", err)
	}

	return isAdmin, nil
}
