package notifications

import (
	"context"
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/bootstrap"
)

type Service interface {
	GetByUser(ctx context.Context, user *bootstrap.User) ([]model.UserVehicleNotifications, error)
	GetByVehicle(ctx context.Context, user *bootstrap.User, vehicleID string) ([]model.UserVehicleNotifications, error)
	GetByID(ctx context.Context, user *bootstrap.User, notificationID string) (*model.UserVehicleNotifications, error)
	Upsert(ctx context.Context, user *bootstrap.User, notification model.UserVehicleNotifications) error
	Delete(ctx context.Context, user *bootstrap.User, notificationID string) error
	DeleteAllByVehicle(ctx context.Context, user *bootstrap.User, vehicleID string) error
}
