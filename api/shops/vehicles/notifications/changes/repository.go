package changes

import (
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/response"
	"miltechserver/bootstrap"
)

type Repository interface {
	GetNotificationChanges(user *bootstrap.User, notificationID string) ([]response.NotificationChangeWithUsername, error)
	GetNotificationChangesByShop(user *bootstrap.User, shopID string, limit int) ([]response.NotificationChangeWithUsername, error)
	GetNotificationChangesByVehicle(user *bootstrap.User, vehicleID string) ([]response.NotificationChangeWithUsername, error)
	GetVehicleNotificationByID(user *bootstrap.User, notificationID string) (*model.ShopVehicleNotifications, error)
	GetShopVehicleByID(user *bootstrap.User, vehicleID string) (*model.ShopVehicle, error)
	IsUserMemberOfShop(user *bootstrap.User, shopID string) (bool, error)
}
