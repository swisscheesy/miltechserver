package lists

import (
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/response"
	"miltechserver/bootstrap"
)

type Repository interface {
	CreateShopList(user *bootstrap.User, list model.ShopLists) (*response.ShopListWithUsername, error)
	GetShopLists(user *bootstrap.User, shopID string) ([]response.ShopListWithUsername, error)
	GetShopListByID(user *bootstrap.User, listID string) (*response.ShopListWithUsername, error)
	UpdateShopList(user *bootstrap.User, list model.ShopLists) error
	DeleteShopList(user *bootstrap.User, listID string) error
}
