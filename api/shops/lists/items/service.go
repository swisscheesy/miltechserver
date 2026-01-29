package items

import (
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/response"
	"miltechserver/bootstrap"
)

type Service interface {
	AddListItem(user *bootstrap.User, item model.ShopListItems) (*response.ShopListItemWithUsername, error)
	GetListItems(user *bootstrap.User, listID string) ([]response.ShopListItemWithUsername, error)
	UpdateListItem(user *bootstrap.User, item model.ShopListItems) error
	RemoveListItem(user *bootstrap.User, itemID string) error
	AddListItemBatch(user *bootstrap.User, items []model.ShopListItems) ([]response.ShopListItemWithUsername, error)
	RemoveListItemBatch(user *bootstrap.User, itemIDs []string) error
}
