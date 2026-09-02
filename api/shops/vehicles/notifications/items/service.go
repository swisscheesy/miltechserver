package items

import (
	"context"
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/bootstrap"
)

type Service interface {
	AddNotificationItem(ctx context.Context, user *bootstrap.User, item model.ShopNotificationItems) (*model.ShopNotificationItems, error)
	GetNotificationItems(ctx context.Context, user *bootstrap.User, notificationID string) ([]model.ShopNotificationItems, error)
	GetShopNotificationItems(ctx context.Context, user *bootstrap.User, shopID string) ([]model.ShopNotificationItems, error)
	AddNotificationItemList(ctx context.Context, user *bootstrap.User, items []model.ShopNotificationItems) ([]model.ShopNotificationItems, error)
	RemoveNotificationItem(ctx context.Context, user *bootstrap.User, itemID string) error
	RemoveNotificationItemList(ctx context.Context, user *bootstrap.User, itemIDs []string) error
}
