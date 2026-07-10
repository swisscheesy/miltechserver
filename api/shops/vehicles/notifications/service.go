package notifications

import (
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/response"
	"miltechserver/bootstrap"
)

type VehicleNotificationUpdate struct {
	Notification        model.ShopVehicleNotifications
	AttachedShopListSet bool
	AttachedShopList    *string
}

type Service interface {
	CreateVehicleNotification(user *bootstrap.User, notification model.ShopVehicleNotifications) (*model.ShopVehicleNotifications, error)
	GetVehicleNotifications(user *bootstrap.User, vehicleID string) ([]model.ShopVehicleNotifications, error)
	GetVehicleNotificationsWithItems(user *bootstrap.User, vehicleID string) ([]response.VehicleNotificationWithItems, error)
	GetShopNotifications(user *bootstrap.User, shopID string) ([]model.ShopVehicleNotifications, error)
	GetVehicleNotificationByID(user *bootstrap.User, notificationID string) (*model.ShopVehicleNotifications, error)
	UpdateVehicleNotification(user *bootstrap.User, update VehicleNotificationUpdate) error
	DeleteVehicleNotification(user *bootstrap.User, notificationID string) error
}
