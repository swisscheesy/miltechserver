package notifications

import (
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/response"
	"miltechserver/bootstrap"
)

type Repository interface {
	CreateVehicleNotification(user *bootstrap.User, notification model.ShopVehicleNotifications) (*model.ShopVehicleNotifications, error)
	GetVehicleNotifications(user *bootstrap.User, vehicleID string) ([]model.ShopVehicleNotifications, error)
	GetVehicleNotificationsWithItems(user *bootstrap.User, vehicleID string) ([]response.VehicleNotificationWithItems, error)
	GetShopNotifications(user *bootstrap.User, shopID string) ([]model.ShopVehicleNotifications, error)
	GetVehicleNotificationByID(user *bootstrap.User, notificationID string) (*model.ShopVehicleNotifications, error)
	UpdateVehicleNotification(user *bootstrap.User, notification model.ShopVehicleNotifications) error
	DeleteVehicleNotification(user *bootstrap.User, notificationID string) error
	CreateNotificationChange(user *bootstrap.User, change model.ShopVehicleNotificationChanges) error
	GetShopVehicleByID(user *bootstrap.User, vehicleID string) (*model.ShopVehicle, error)
	IsUserMemberOfShop(user *bootstrap.User, shopID string) (bool, error)
}
