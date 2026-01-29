package changes

import (
	"miltechserver/api/response"
	"miltechserver/bootstrap"
)

type Service interface {
	GetNotificationChangeHistory(user *bootstrap.User, notificationID string) ([]response.NotificationChangeWithUsername, error)
	GetShopNotificationChanges(user *bootstrap.User, shopID string, limit int) ([]response.NotificationChangeWithUsername, error)
	GetVehicleNotificationChanges(user *bootstrap.User, vehicleID string) ([]response.NotificationChangeWithUsername, error)
}
