package changes

import (
	"context"
	"miltechserver/api/response"
	"miltechserver/bootstrap"
)

type Service interface {
	GetNotificationChangeHistory(ctx context.Context, user *bootstrap.User, notificationID string) ([]response.NotificationChangeWithUsername, error)
	GetShopNotificationChanges(ctx context.Context, user *bootstrap.User, shopID string, limit int) ([]response.NotificationChangeWithUsername, error)
	GetVehicleNotificationChanges(ctx context.Context, user *bootstrap.User, vehicleID string) ([]response.NotificationChangeWithUsername, error)
}
