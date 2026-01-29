package items

import (
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/bootstrap"
)

type Repository interface {
	CreateNotificationItem(user *bootstrap.User, item model.ShopNotificationItems) (*model.ShopNotificationItems, error)
	GetNotificationItems(user *bootstrap.User, notificationID string) ([]model.ShopNotificationItems, error)
	GetShopNotificationItems(user *bootstrap.User, shopID string) ([]model.ShopNotificationItems, error)
	GetNotificationItemByID(user *bootstrap.User, itemID string) (*model.ShopNotificationItems, error)
	GetNotificationItemsByIDs(user *bootstrap.User, itemIDs []string) ([]model.ShopNotificationItems, error)
	CreateNotificationItemList(user *bootstrap.User, items []model.ShopNotificationItems) ([]model.ShopNotificationItems, error)
	DeleteNotificationItem(user *bootstrap.User, itemID string) error
	DeleteNotificationItemList(user *bootstrap.User, itemIDs []string) error
	GetVehicleNotificationByID(user *bootstrap.User, notificationID string) (*model.ShopVehicleNotifications, error)
	GetShopVehicleByID(user *bootstrap.User, vehicleID string) (*model.ShopVehicle, error)
	IsUserMemberOfShop(user *bootstrap.User, shopID string) (bool, error)
	CreateNotificationChange(user *bootstrap.User, change model.ShopVehicleNotificationChanges) error
}
