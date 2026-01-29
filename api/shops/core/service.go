package core

import (
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/response"
	"miltechserver/bootstrap"
)

type ShopService interface {
	CreateShop(user *bootstrap.User, shop model.Shops) (*model.Shops, error)
	UpdateShop(user *bootstrap.User, shop model.Shops) (*model.Shops, error)
	DeleteShop(user *bootstrap.User, shopID string) error
	GetShopsByUser(user *bootstrap.User) ([]model.Shops, error)
	GetShopByID(user *bootstrap.User, shopID string) (*response.ShopDetailResponse, error)
	GetUserDataWithShops(user *bootstrap.User) (*response.UserShopsResponse, error)
}
