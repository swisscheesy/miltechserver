package notifications

import (
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/bootstrap"
)

type Repository interface {
	GetByUserID(user *bootstrap.User) ([]model.UserVehicleNotifications, error)
	GetByVehicleID(user *bootstrap.User, vehicleID string) ([]model.UserVehicleNotifications, error)
	GetByID(user *bootstrap.User, notificationID string) (*model.UserVehicleNotifications, error)
	Upsert(user *bootstrap.User, notification model.UserVehicleNotifications) error
	Delete(user *bootstrap.User, notificationID string) error
	DeleteAllByVehicle(user *bootstrap.User, vehicleID string) error
}
