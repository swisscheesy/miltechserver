package repository

import (
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/bootstrap"
)

type UserVehicleRepository interface {
	// User Vehicle Operations
	GetUserVehiclesByUserId(user *bootstrap.User) ([]model.UserVehicle, error)
	GetUserVehicleById(user *bootstrap.User, vehicleId string) (*model.UserVehicle, error)
	UpsertUserVehicle(user *bootstrap.User, vehicle model.UserVehicle) error
	DeleteUserVehicle(user *bootstrap.User, vehicleId string) error
	DeleteAllUserVehiclesByUserId(user *bootstrap.User) error
	DeleteAllUserVehicles(user *bootstrap.User) error

	// User Vehicle Notifications Operations
	GetVehicleNotificationsByUserId(user *bootstrap.User) ([]model.UserVehicleNotifications, error)
	GetVehicleNotificationsByVehicleId(user *bootstrap.User, vehicleId string) ([]model.UserVehicleNotifications, error)
	GetVehicleNotificationById(user *bootstrap.User, notificationId string) (*model.UserVehicleNotifications, error)
	UpsertVehicleNotification(user *bootstrap.User, notification model.UserVehicleNotifications) error
	DeleteVehicleNotification(user *bootstrap.User, notificationId string) error
	DeleteAllVehicleNotificationsByVehicle(user *bootstrap.User, vehicleId string) error

	// User Vehicle Comments Operations
	GetVehicleCommentsByUserId(user *bootstrap.User) ([]model.UserVehicleComments, error)
	GetVehicleCommentsByVehicleId(user *bootstrap.User, vehicleId string) ([]model.UserVehicleComments, error)
	GetVehicleCommentsByNotificationId(user *bootstrap.User, notificationId string) ([]model.UserVehicleComments, error)
	GetVehicleCommentById(user *bootstrap.User, commentId string) (*model.UserVehicleComments, error)
	UpsertVehicleComment(user *bootstrap.User, comment model.UserVehicleComments) error
	DeleteVehicleComment(user *bootstrap.User, commentId string) error
	DeleteAllVehicleCommentsByVehicle(user *bootstrap.User, vehicleId string) error

	// User Notification Items Operations
	GetNotificationItemsByUserId(user *bootstrap.User) ([]model.UserNotificationItems, error)
	GetNotificationItemsByNotificationId(user *bootstrap.User, notificationId string) ([]model.UserNotificationItems, error)
	GetNotificationItemById(user *bootstrap.User, itemId string) (*model.UserNotificationItems, error)
	UpsertNotificationItem(user *bootstrap.User, item model.UserNotificationItems) error
	UpsertNotificationItemList(user *bootstrap.User, items []model.UserNotificationItems) error
	DeleteNotificationItem(user *bootstrap.User, itemId string) error
	DeleteAllNotificationItemsByNotification(user *bootstrap.User, notificationId string) error
}
