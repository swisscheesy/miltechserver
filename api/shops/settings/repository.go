package settings

import (
	"miltechserver/api/request"
	"miltechserver/bootstrap"
)

type Repository interface {
	GetShopAdminOnlyListsSetting(shopID string) (bool, error)
	UpdateShopAdminOnlyListsSetting(user *bootstrap.User, shopID string, adminOnlyLists bool) error
	GetShopSettings(shopID string) (*request.ShopSettings, error)
	UpdateShopSettings(user *bootstrap.User, shopID string, updates request.UpdateShopSettingsRequest) error
}
