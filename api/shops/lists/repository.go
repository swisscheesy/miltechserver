package lists

import (
	"context"
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/response"
	"miltechserver/bootstrap"
)

type Repository interface {
	CreateShopList(ctx context.Context, user *bootstrap.User, list model.ShopLists) (*response.ShopListWithUsername, error)
	GetShopLists(ctx context.Context, user *bootstrap.User, shopID string) ([]response.ShopListWithUsername, error)
	GetShopListByID(ctx context.Context, user *bootstrap.User, listID string) (*response.ShopListWithUsername, error)
	UpdateShopList(ctx context.Context, user *bootstrap.User, list model.ShopLists) error
	DeleteShopList(ctx context.Context, user *bootstrap.User, listID string) error
}
