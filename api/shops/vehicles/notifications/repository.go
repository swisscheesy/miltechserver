package notifications

import (
	"context"
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/response"
	"miltechserver/bootstrap"
)

type Repository interface {
	CreateVehicleNotification(ctx context.Context, user *bootstrap.User, notification model.ShopVehicleNotifications) (*model.ShopVehicleNotifications, error)
	GetVehicleNotifications(ctx context.Context, user *bootstrap.User, vehicleID string) ([]model.ShopVehicleNotifications, error)
	GetVehicleNotificationsWithItems(ctx context.Context, user *bootstrap.User, vehicleID string) ([]response.VehicleNotificationWithItems, error)
	GetShopNotifications(ctx context.Context, user *bootstrap.User, shopID string) ([]model.ShopVehicleNotifications, error)
	GetVehicleNotificationByID(ctx context.Context, user *bootstrap.User, notificationID string) (*model.ShopVehicleNotifications, error)
	UpdateVehicleNotification(ctx context.Context, user *bootstrap.User, update VehicleNotificationUpdate) error
	DeleteVehicleNotification(ctx context.Context, user *bootstrap.User, notificationID string) error
	CreateNotificationChange(ctx context.Context, user *bootstrap.User, change model.ShopVehicleNotificationChanges) error
	GetShopVehicleByID(ctx context.Context, user *bootstrap.User, vehicleID string) (*model.ShopVehicle, error)
	GetShopListByID(ctx context.Context, user *bootstrap.User, listID string) (*model.ShopLists, error)
	IsUserMemberOfShop(ctx context.Context, user *bootstrap.User, shopID string) (bool, error)
}
