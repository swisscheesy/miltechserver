package notification_items

import (
	"context"
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/bootstrap"
)

type Service interface {
	GetByUser(ctx context.Context, user *bootstrap.User) ([]model.UserNotificationItems, error)
	GetByNotification(ctx context.Context, user *bootstrap.User, notificationID string) ([]model.UserNotificationItems, error)
	GetByID(ctx context.Context, user *bootstrap.User, itemID string) (*model.UserNotificationItems, error)
	Upsert(ctx context.Context, user *bootstrap.User, item model.UserNotificationItems) error
	UpsertBatch(ctx context.Context, user *bootstrap.User, items []model.UserNotificationItems) error
	Delete(ctx context.Context, user *bootstrap.User, itemID string) error
	DeleteAllByNotification(ctx context.Context, user *bootstrap.User, notificationID string) error
}
