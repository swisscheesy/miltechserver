package items

import (
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/bootstrap"
)

type Service interface {
	AddNotificationItem(user *bootstrap.User, item model.ShopNotificationItems) (*model.ShopNotificationItems, error)
	GetNotificationItems(user *bootstrap.User, notificationID string) ([]model.ShopNotificationItems, error)
	GetShopNotificationItems(user *bootstrap.User, shopID string) ([]model.ShopNotificationItems, error)
	AddNotificationItemList(user *bootstrap.User, items []model.ShopNotificationItems) ([]model.ShopNotificationItems, error)
	RemoveNotificationItem(user *bootstrap.User, itemID string) error
	RemoveNotificationItemList(user *bootstrap.User, itemIDs []string) error
}
