package items

import (
	"context"
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

func (service *ServiceImpl) WithAuthorization(auth shared.ShopAuthorization) shared.AuthorizationAware {
	return &ServiceImpl{
		repo:         service.repo,
		listRepo:     service.listRepo,
		settingsRepo: service.settingsRepo,
		auth:         auth,
	}
}

func (service *ServiceImpl) AddListItem(ctx context.Context, user *bootstrap.User, item model.ShopListItems) (*response.ShopListItemWithUsername, error) {
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

	list, err := service.listRepo.GetShopListByID(ctx, user, item.ListID)
	if err != nil {
		return nil, fmt.Errorf("failed to get list: %w", err)
	}

	isMember, err := service.auth.IsUserMemberOfShop(ctx, user, list.ShopID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify membership: %w", err)
	}

	if !isMember {
		return nil, errors.New("access denied: user is not a member of this shop")
	}

	canModify, err := service.canUserModifyListWithAdminOnlyCheck(ctx, user, list.ShopID)
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

	createdItem, err := service.repo.AddListItem(ctx, user, item)
	if err != nil {
		return nil, fmt.Errorf("failed to add list item: %w", err)
	}

	slog.Info("List item added", "user_id", user.UserID, "list_id", item.ListID, "item_id", item.ID)
	return createdItem, nil
}

func (service *ServiceImpl) GetListItems(ctx context.Context, user *bootstrap.User, listID string) ([]response.ShopListItemWithUsername, error) {
	if user == nil {
		return nil, errors.New("unauthorized user")
	}

	list, err := service.listRepo.GetShopListByID(ctx, user, listID)
	if err != nil {
		return nil, fmt.Errorf("failed to get list: %w", err)
	}

	isMember, err := service.auth.IsUserMemberOfShop(ctx, user, list.ShopID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify membership: %w", err)
	}

	if !isMember {
		return nil, errors.New("access denied: user is not a member of this shop")
	}

	items, err := service.repo.GetListItems(ctx, user, listID)
	if err != nil {
		return nil, fmt.Errorf("failed to get list items with usernames: %w", err)
	}

	if items == nil {
		return []response.ShopListItemWithUsername{}, nil
	}

	return items, nil
}

func (service *ServiceImpl) UpdateListItem(ctx context.Context, user *bootstrap.User, item model.ShopListItems) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := shared.ValidateItemFields(item.Niin, item.Nomenclature, item.Quantity); err != nil {
		return err
	}
	if err := shared.ValidateItemEnrichment(item.Nickname, item.UnitOfMeasure); err != nil {
		return err
	}

	if user == nil {
		return errors.New("unauthorized user")
	}

	currentItem, err := service.repo.GetListItemByID(ctx, user, item.ID)
	if err != nil {
		return fmt.Errorf("failed to get item: %w", err)
	}

	list, err := service.listRepo.GetShopListByID(ctx, user, currentItem.ListID)
	if err != nil {
		return fmt.Errorf("failed to get list: %w", err)
	}

	isMember, err := service.auth.IsUserMemberOfShop(ctx, user, list.ShopID)
	if err != nil {
		return fmt.Errorf("failed to verify membership: %w", err)
	}

	if !isMember {
		return errors.New("access denied: user is not a member of this shop")
	}

	canModify, err := service.canUserModifyListWithAdminOnlyCheck(ctx, user, list.ShopID)
	if err != nil {
		return fmt.Errorf("failed to verify list modification permissions: %w", err)
	}
	if !canModify {
		return errors.New("access denied: insufficient permissions to modify list items")
	}

	item.UpdatedAt = time.Now()

	err = service.repo.UpdateListItem(ctx, user, item)
	if err != nil {
		return fmt.Errorf("failed to update list item: %w", err)
	}

	slog.Info("List item updated", "user_id", user.UserID, "item_id", item.ID)
	return nil
}

func (service *ServiceImpl) RemoveListItem(ctx context.Context, user *bootstrap.User, itemID string) error {
	if user == nil {
		return errors.New("unauthorized user")
	}

	item, err := service.repo.GetListItemByID(ctx, user, itemID)
	if err != nil {
		return fmt.Errorf("failed to get item: %w", err)
	}

	list, err := service.listRepo.GetShopListByID(ctx, user, item.ListID)
	if err != nil {
		return fmt.Errorf("failed to get list: %w", err)
	}

	isMember, err := service.auth.IsUserMemberOfShop(ctx, user, list.ShopID)
	if err != nil {
		return fmt.Errorf("failed to verify membership: %w", err)
	}

	if !isMember {
		return errors.New("access denied: user is not a member of this shop")
	}

	canModify, err := service.canUserModifyListWithAdminOnlyCheck(ctx, user, list.ShopID)
	if err != nil {
		return fmt.Errorf("failed to verify list modification permissions: %w", err)
	}
	if !canModify {
		return errors.New("access denied: insufficient permissions to modify list items")
	}

	err = service.repo.RemoveListItem(ctx, user, itemID)
	if err != nil {
		return fmt.Errorf("failed to remove list item: %w", err)
	}

	slog.Info("List item removed", "user_id", user.UserID, "item_id", itemID)
	return nil
}

func (service *ServiceImpl) AddListItemBatch(ctx context.Context, user *bootstrap.User, items []model.ShopListItems) ([]response.ShopListItemWithUsername, error) {
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

	if user == nil {
		return nil, errors.New("unauthorized user")
	}

	if len(items) == 0 {
		return []response.ShopListItemWithUsername{}, errors.New("no items to add")
	}

	list, err := service.listRepo.GetShopListByID(ctx, user, items[0].ListID)
	if err != nil {
		return nil, fmt.Errorf("failed to get list: %w", err)
	}

	isMember, err := service.auth.IsUserMemberOfShop(ctx, user, list.ShopID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify membership: %w", err)
	}

	if !isMember {
		return nil, errors.New("access denied: user is not a member of this shop")
	}

	canModify, err := service.canUserModifyListWithAdminOnlyCheck(ctx, user, list.ShopID)
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

	createdItems, err := service.repo.AddListItemBatch(ctx, user, items)
	if err != nil {
		return nil, fmt.Errorf("failed to add list items: %w", err)
	}

	slog.Info("List items added", "user_id", user.UserID, "list_id", items[0].ListID, "count", len(createdItems))
	return createdItems, nil
}

func (service *ServiceImpl) RemoveListItemBatch(ctx context.Context, user *bootstrap.User, itemIDs []string) (int64, error) {
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
	count, err := service.repo.RemoveListItemBatch(ctx, user, itemIDs)
	if err != nil {
		return 0, fmt.Errorf("failed to remove list items: %w", err)
	}
	slog.Info("List items removed", "user_id", user.UserID, "count", count)
	return count, nil
}

// canUserModifyListWithAdminOnlyCheck checks if user can modify lists based on shop's admin_only_lists setting
// If admin_only_lists is true, only shop admins can modify lists
// If admin_only_lists is false, all shop members can modify lists
func (service *ServiceImpl) canUserModifyListWithAdminOnlyCheck(ctx context.Context, user *bootstrap.User, shopID string) (bool, error) {
	isMember, err := service.auth.IsUserMemberOfShop(ctx, user, shopID)
	if err != nil || !isMember {
		return false, err
	}
	adminOnlyLists, err := service.settingsRepo.GetShopAdminOnlyListsSetting(ctx, shopID)
	if err != nil {
		return false, fmt.Errorf("failed to get admin_only_lists setting: %w", err)
	}

	if !adminOnlyLists {
		return true, nil
	}

	isAdmin, err := service.auth.IsUserShopAdmin(ctx, user, shopID)
	if err != nil {
		return false, fmt.Errorf("failed to verify admin status: %w", err)
	}

	return isAdmin, nil
}
