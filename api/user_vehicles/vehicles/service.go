package vehicles

import (
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/bootstrap"
)

type Service interface {
	GetByUser(user *bootstrap.User) ([]model.UserVehicle, error)
	GetByID(user *bootstrap.User, vehicleID string) (*model.UserVehicle, error)
	Upsert(user *bootstrap.User, vehicle model.UserVehicle) error
	Delete(user *bootstrap.User, vehicleID string) error
	DeleteAll(user *bootstrap.User) error
}
