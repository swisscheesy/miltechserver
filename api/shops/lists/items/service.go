package items

import (
	"context"
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/response"
	"miltechserver/bootstrap"
)

type Service interface {
	AddListItem(ctx context.Context, user *bootstrap.User, item model.ShopListItems) (*response.ShopListItemWithUsername, error)
	GetListItems(ctx context.Context, user *bootstrap.User, listID string) ([]response.ShopListItemWithUsername, error)
	UpdateListItem(ctx context.Context, user *bootstrap.User, item model.ShopListItems) error
	RemoveListItem(ctx context.Context, user *bootstrap.User, itemID string) error
	AddListItemBatch(ctx context.Context, user *bootstrap.User, items []model.ShopListItems) ([]response.ShopListItemWithUsername, error)
	RemoveListItemBatch(ctx context.Context, user *bootstrap.User, itemIDs []string) error
}
