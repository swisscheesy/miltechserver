package settings

import "miltechserver/api/request"

type Repository interface {
	GetShopAdminOnlyListsSetting(shopID string) (bool, error)
	UpdateShopAdminOnlyListsSetting(shopID string, adminOnlyLists bool) error
	GetShopSettings(shopID string) (*request.ShopSettings, error)
	UpdateShopSettings(shopID string, updates request.UpdateShopSettingsRequest) error
}
