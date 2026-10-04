package items

import (
	"context"
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/bootstrap"
)

type Repository interface {
	CreateNotificationItem(ctx context.Context, user *bootstrap.User, item model.ShopNotificationItems) (*model.ShopNotificationItems, error)
	GetNotificationItems(ctx context.Context, user *bootstrap.User, notificationID string) ([]model.ShopNotificationItems, error)
	GetShopNotificationItems(ctx context.Context, user *bootstrap.User, shopID string) ([]model.ShopNotificationItems, error)
	GetNotificationItemByID(ctx context.Context, user *bootstrap.User, itemID string) (*model.ShopNotificationItems, error)
	GetNotificationItemsByIDs(ctx context.Context, user *bootstrap.User, itemIDs []string) ([]model.ShopNotificationItems, error)
	CreateNotificationItemList(ctx context.Context, user *bootstrap.User, items []model.ShopNotificationItems) ([]model.ShopNotificationItems, error)
	DeleteNotificationItem(ctx context.Context, user *bootstrap.User, itemID string) error
	DeleteNotificationItemList(ctx context.Context, user *bootstrap.User, itemIDs []string) (int64, error)
	GetVehicleNotificationByID(ctx context.Context, user *bootstrap.User, notificationID string) (*model.ShopVehicleNotifications, error)
	GetShopVehicleByID(ctx context.Context, user *bootstrap.User, vehicleID string) (*model.ShopVehicle, error)
	IsUserMemberOfShop(ctx context.Context, user *bootstrap.User, shopID string) (bool, error)
	CreateNotificationChange(ctx context.Context, user *bootstrap.User, change model.ShopVehicleNotificationChanges) error
}
