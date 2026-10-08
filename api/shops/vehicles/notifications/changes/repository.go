package changes

import (
	"context"
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/response"
	"miltechserver/bootstrap"
)

type Repository interface {
	GetNotificationChanges(ctx context.Context, user *bootstrap.User, notificationID string) ([]response.NotificationChangeWithUsername, error)
	GetNotificationChangesByShop(ctx context.Context, user *bootstrap.User, shopID string, limit int) ([]response.NotificationChangeWithUsername, error)
	GetNotificationChangesByVehicle(ctx context.Context, user *bootstrap.User, vehicleID string) ([]response.NotificationChangeWithUsername, error)
	GetVehicleNotificationByID(ctx context.Context, user *bootstrap.User, notificationID string) (*model.ShopVehicleNotifications, error)
	GetShopVehicleByID(ctx context.Context, user *bootstrap.User, vehicleID string) (*model.ShopVehicle, error)
	IsUserMemberOfShop(ctx context.Context, user *bootstrap.User, shopID string) (bool, error)
}
