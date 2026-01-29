package settings

import (
	"miltechserver/api/request"
	"miltechserver/bootstrap"
)

type Service interface {
	GetShopAdminOnlyListsSetting(user *bootstrap.User, shopID string) (bool, error)
	UpdateShopAdminOnlyListsSetting(user *bootstrap.User, shopID string, adminOnlyLists bool) error
	IsUserShopAdmin(user *bootstrap.User, shopID string) (bool, error)
	GetShopSettings(user *bootstrap.User, shopID string) (*request.ShopSettings, error)
	UpdateShopSettings(user *bootstrap.User, shopID string, updates request.UpdateShopSettingsRequest) (*request.ShopSettings, error)
}
