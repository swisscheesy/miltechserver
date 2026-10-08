package settings

import (
	"context"
	"miltechserver/api/request"
	"miltechserver/bootstrap"
)

type Repository interface {
	GetShopAdminOnlyListsSetting(ctx context.Context, shopID string) (bool, error)
	UpdateShopAdminOnlyListsSetting(ctx context.Context, user *bootstrap.User, shopID string, adminOnlyLists bool) error
	GetShopSettings(ctx context.Context, shopID string) (*request.ShopSettings, error)
	UpdateShopSettings(ctx context.Context, user *bootstrap.User, shopID string, updates request.UpdateShopSettingsRequest) error
}
