package vehicles

import (
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/bootstrap"
)

type ServiceImpl struct {
	repository Repository
}

func NewService(repository Repository) *ServiceImpl {
	return &ServiceImpl{repository: repository}
}

func (service *ServiceImpl) GetByUser(user *bootstrap.User) ([]model.UserVehicle, error) {
	vehicles, err := service.repository.GetByUserID(user)
	if vehicles == nil {
		return []model.UserVehicle{}, nil
	}
	return vehicles, err
}

func (service *ServiceImpl) GetByID(user *bootstrap.User, vehicleID string) (*model.UserVehicle, error) {
	return service.repository.GetByID(user, vehicleID)
}

func (service *ServiceImpl) Upsert(user *bootstrap.User, vehicle model.UserVehicle) error {
	return service.repository.Upsert(user, vehicle)
}

func (service *ServiceImpl) Delete(user *bootstrap.User, vehicleID string) error {
	return service.repository.Delete(user, vehicleID)
}

func (service *ServiceImpl) DeleteAll(user *bootstrap.User) error {
	return service.repository.DeleteAll(user)
}
