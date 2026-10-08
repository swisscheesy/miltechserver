package settings

import (
	"context"
	"miltechserver/api/request"
	"miltechserver/bootstrap"
)

type Service interface {
	GetShopAdminOnlyListsSetting(ctx context.Context, user *bootstrap.User, shopID string) (bool, error)
	UpdateShopAdminOnlyListsSetting(ctx context.Context, user *bootstrap.User, shopID string, adminOnlyLists bool) error
	IsUserShopAdmin(ctx context.Context, user *bootstrap.User, shopID string) (bool, error)
	GetShopSettings(ctx context.Context, user *bootstrap.User, shopID string) (*request.ShopSettings, error)
	UpdateShopSettings(ctx context.Context, user *bootstrap.User, shopID string, updates request.UpdateShopSettingsRequest) (*request.ShopSettings, error)
}
