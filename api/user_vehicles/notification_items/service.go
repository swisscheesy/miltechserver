package notification_items

import (
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/bootstrap"
)

type Service interface {
	GetByUser(user *bootstrap.User) ([]model.UserNotificationItems, error)
	GetByNotification(user *bootstrap.User, notificationID string) ([]model.UserNotificationItems, error)
	GetByID(user *bootstrap.User, itemID string) (*model.UserNotificationItems, error)
	Upsert(user *bootstrap.User, item model.UserNotificationItems) error
	UpsertBatch(user *bootstrap.User, items []model.UserNotificationItems) error
	Delete(user *bootstrap.User, itemID string) error
	DeleteAllByNotification(user *bootstrap.User, notificationID string) error
}
