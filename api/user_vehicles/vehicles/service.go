package vehicles

import (
	"context"
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/bootstrap"
)

type Service interface {
	GetByUser(ctx context.Context, user *bootstrap.User) ([]model.UserVehicle, error)
	GetByID(ctx context.Context, user *bootstrap.User, vehicleID string) (*model.UserVehicle, error)
	Upsert(ctx context.Context, user *bootstrap.User, vehicle model.UserVehicle) error
	Delete(ctx context.Context, user *bootstrap.User, vehicleID string) error
	DeleteAll(ctx context.Context, user *bootstrap.User) error
}
