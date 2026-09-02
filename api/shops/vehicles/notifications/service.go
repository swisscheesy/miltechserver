package notifications

import (
	"context"
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
	CreateVehicleNotification(ctx context.Context, user *bootstrap.User, notification model.ShopVehicleNotifications) (*model.ShopVehicleNotifications, error)
	GetVehicleNotifications(ctx context.Context, user *bootstrap.User, vehicleID string) ([]model.ShopVehicleNotifications, error)
	GetVehicleNotificationsWithItems(ctx context.Context, user *bootstrap.User, vehicleID string) ([]response.VehicleNotificationWithItems, error)
	GetShopNotifications(ctx context.Context, user *bootstrap.User, shopID string) ([]model.ShopVehicleNotifications, error)
	GetVehicleNotificationByID(ctx context.Context, user *bootstrap.User, notificationID string) (*model.ShopVehicleNotifications, error)
	UpdateVehicleNotification(ctx context.Context, user *bootstrap.User, update VehicleNotificationUpdate) error
	DeleteVehicleNotification(ctx context.Context, user *bootstrap.User, notificationID string) error
}
