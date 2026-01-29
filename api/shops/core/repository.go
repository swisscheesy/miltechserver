package core

import (
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/response"
	"miltechserver/bootstrap"
)

type Repository interface {
	CreateShop(user *bootstrap.User, shop model.Shops) (*model.Shops, error)
	UpdateShop(user *bootstrap.User, shop model.Shops) (*model.Shops, error)
	DeleteShop(user *bootstrap.User, shopID string) error
	GetShopsByUser(user *bootstrap.User) ([]model.Shops, error)
	GetShopByID(user *bootstrap.User, shopID string) (*response.ShopDetailResponse, error)
	GetShopsWithStatsForUser(user *bootstrap.User) ([]response.ShopWithStats, error)
	AddMemberToShop(user *bootstrap.User, shopID string, role string) error
	DeleteShopMessageBlobs(shopID string) error
}
