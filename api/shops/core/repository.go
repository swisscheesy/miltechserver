package core

import (
	"context"
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/response"
	"miltechserver/bootstrap"
)

type Repository interface {
	CreateShop(ctx context.Context, user *bootstrap.User, shop model.Shops) (*model.Shops, error)
	UpdateShop(ctx context.Context, user *bootstrap.User, shop model.Shops) (*model.Shops, error)
	DeleteShop(ctx context.Context, user *bootstrap.User, shopID string) error
	GetShopsByUser(ctx context.Context, user *bootstrap.User) ([]model.Shops, error)
	GetShopByID(ctx context.Context, user *bootstrap.User, shopID string) (*response.ShopDetailResponse, error)
	GetShopsWithStatsForUser(ctx context.Context, user *bootstrap.User) ([]response.ShopWithStats, error)
	GetShopEquipmentOverview(ctx context.Context, user *bootstrap.User) ([]response.ShopEquipmentOverview, error)
}
