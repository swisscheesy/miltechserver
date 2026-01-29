package items

import (
	"errors"
	"fmt"
	"log/slog"
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/response"
	"miltechserver/api/shops/lists"
	"miltechserver/api/shops/settings"
	"miltechserver/api/shops/shared"
	"miltechserver/bootstrap"
	"time"

	"github.com/google/uuid"
)

type ServiceImpl struct {
	repo         Repository
	listRepo     lists.Repository
	settingsRepo settings.Repository
	auth         shared.ShopAuthorization
}

func NewService(repo Repository, listRepo lists.Repository, settingsRepo settings.Repository, auth shared.ShopAuthorization) *ServiceImpl {
	return &ServiceImpl{
		repo:         repo,
		listRepo:     listRepo,
		settingsRepo: settingsRepo,
		auth:         auth,
	}
}

func (service *ServiceImpl) AddListItem(user *bootstrap.User, item model.ShopListItems) (*response.ShopListItemWithUsername, error) {
	if user == nil {
		return nil, errors.New("unauthorized user")
	}

	list, err := service.listRepo.GetShopListByID(user, item.ListID)
	if err != nil {
		return nil, fmt.Errorf("failed to get list: %w", err)
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
		return nil, errors.New("access denied: insufficient permissions to modify list items")
	}

	item.ID = uuid.New().String()
	item.AddedBy = user.UserID
	now := time.Now()
	item.CreatedAt = now
	item.UpdatedAt = now

	createdItem, err := service.repo.AddListItem(user, item)
	if err != nil {
		return nil, fmt.Errorf("failed to add list item: %w", err)
	}

	slog.Info("List item added", "user_id", user.UserID, "list_id", item.ListID, "item_id", item.ID)
	return createdItem, nil
}

func (service *ServiceImpl) GetListItems(user *bootstrap.User, listID string) ([]response.ShopListItemWithUsername, error) {
	if user == nil {
		return nil, errors.New("unauthorized user")
	}

	list, err := service.listRepo.GetShopListByID(user, listID)
	if err != nil {
		return nil, fmt.Errorf("failed to get list: %w", err)
	}

	isMember, err := service.auth.IsUserMemberOfShop(user, list.ShopID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify membership: %w", err)
	}

	if !isMember {
		return nil, errors.New("access denied: user is not a member of this shop")
	}

	items, err := service.repo.GetListItems(user, listID)
	if err != nil {
		return nil, fmt.Errorf("failed to get list items with usernames: %w", err)
	}

	if items == nil {
		return []response.ShopListItemWithUsername{}, nil
	}

	return items, nil
}

func (service *ServiceImpl) UpdateListItem(user *bootstrap.User, item model.ShopListItems) error {
	if user == nil {
		return errors.New("unauthorized user")
	}

	currentItem, err := service.repo.GetListItemByID(user, item.ID)
	if err != nil {
		return fmt.Errorf("failed to get item: %w", err)
	}

	list, err := service.listRepo.GetShopListByID(user, currentItem.ListID)
	if err != nil {
		return fmt.Errorf("failed to get list: %w", err)
	}

	isMember, err := service.auth.IsUserMemberOfShop(user, list.ShopID)
	if err != nil {
		return fmt.Errorf("failed to verify membership: %w", err)
	}

	if !isMember {
		return errors.New("access denied: user is not a member of this shop")
	}

	canModify, err := service.canUserModifyListWithAdminOnlyCheck(user, list.ShopID)
	if err != nil {
		return fmt.Errorf("failed to verify list modification permissions: %w", err)
	}
	if !canModify {
		return errors.New("access denied: insufficient permissions to modify list items")
	}

	item.UpdatedAt = time.Now()

	err = service.repo.UpdateListItem(user, item)
	if err != nil {
		return fmt.Errorf("failed to update list item: %w", err)
	}

	slog.Info("List item updated", "user_id", user.UserID, "item_id", item.ID)
	return nil
}

func (service *ServiceImpl) RemoveListItem(user *bootstrap.User, itemID string) error {
	if user == nil {
		return errors.New("unauthorized user")
	}

	item, err := service.repo.GetListItemByID(user, itemID)
	if err != nil {
		return fmt.Errorf("failed to get item: %w", err)
	}

	list, err := service.listRepo.GetShopListByID(user, item.ListID)
	if err != nil {
		return fmt.Errorf("failed to get list: %w", err)
	}

	isMember, err := service.auth.IsUserMemberOfShop(user, list.ShopID)
	if err != nil {
		return fmt.Errorf("failed to verify membership: %w", err)
	}

	if !isMember {
		return errors.New("access denied: user is not a member of this shop")
	}

	canModify, err := service.canUserModifyListWithAdminOnlyCheck(user, list.ShopID)
	if err != nil {
		return fmt.Errorf("failed to verify list modification permissions: %w", err)
	}
	if !canModify {
		return errors.New("access denied: insufficient permissions to modify list items")
	}

	err = service.repo.RemoveListItem(user, itemID)
	if err != nil {
		return fmt.Errorf("failed to remove list item: %w", err)
	}

	slog.Info("List item removed", "user_id", user.UserID, "item_id", itemID)
	return nil
}

func (service *ServiceImpl) AddListItemBatch(user *bootstrap.User, items []model.ShopListItems) ([]response.ShopListItemWithUsername, error) {
	if user == nil {
		return nil, errors.New("unauthorized user")
	}

	if len(items) == 0 {
		return []response.ShopListItemWithUsername{}, errors.New("no items to add")
	}

	list, err := service.listRepo.GetShopListByID(user, items[0].ListID)
	if err != nil {
		return nil, fmt.Errorf("failed to get list: %w", err)
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
		return nil, errors.New("access denied: insufficient permissions to modify list items")
	}

	now := time.Now()
	for i := range items {
		items[i].ID = uuid.New().String()
		items[i].AddedBy = user.UserID
		items[i].CreatedAt = now
		items[i].UpdatedAt = now
	}

	createdItems, err := service.repo.AddListItemBatch(user, items)
	if err != nil {
		return nil, fmt.Errorf("failed to add list items: %w", err)
	}

	slog.Info("List items added", "user_id", user.UserID, "list_id", items[0].ListID, "count", len(createdItems))
	return createdItems, nil
}

func (service *ServiceImpl) RemoveListItemBatch(user *bootstrap.User, itemIDs []string) error {
	if user == nil {
		return errors.New("unauthorized user")
	}

	if len(itemIDs) == 0 {
		return errors.New("no items to remove")
	}

	firstItem, err := service.repo.GetListItemByID(user, itemIDs[0])
	if err != nil {
		return fmt.Errorf("failed to get item: %w", err)
	}

	list, err := service.listRepo.GetShopListByID(user, firstItem.ListID)
	if err != nil {
		return fmt.Errorf("failed to get list: %w", err)
	}

	isMember, err := service.auth.IsUserMemberOfShop(user, list.ShopID)
	if err != nil {
		return fmt.Errorf("failed to verify membership: %w", err)
	}

	if !isMember {
		return errors.New("access denied: user is not a member of this shop")
	}

	canModify, err := service.canUserModifyListWithAdminOnlyCheck(user, list.ShopID)
	if err != nil {
		return fmt.Errorf("failed to verify list modification permissions: %w", err)
	}
	if !canModify {
		return errors.New("access denied: insufficient permissions to modify list items")
	}

	err = service.repo.RemoveListItemBatch(user, itemIDs)
	if err != nil {
		return fmt.Errorf("failed to remove list items: %w", err)
	}

	slog.Info("List items removed", "user_id", user.UserID, "count", len(itemIDs))
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
