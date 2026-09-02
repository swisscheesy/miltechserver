package notifications

import (
	"context"
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/bootstrap"
)

type ServiceImpl struct {
	repository Repository
}

func NewService(repository Repository) *ServiceImpl {
	return &ServiceImpl{repository: repository}
}

func (service *ServiceImpl) GetByUser(ctx context.Context, user *bootstrap.User) ([]model.UserVehicleNotifications, error) {
	notifications, err := service.repository.GetByUserID(user)
	if notifications == nil {
		return []model.UserVehicleNotifications{}, nil
	}
	return notifications, err
}

func (service *ServiceImpl) GetByVehicle(ctx context.Context, user *bootstrap.User, vehicleID string) ([]model.UserVehicleNotifications, error) {
	notifications, err := service.repository.GetByVehicleID(user, vehicleID)
	if notifications == nil {
		return []model.UserVehicleNotifications{}, nil
	}
	return notifications, err
}

func (service *ServiceImpl) GetByID(ctx context.Context, user *bootstrap.User, notificationID string) (*model.UserVehicleNotifications, error) {
	return service.repository.GetByID(user, notificationID)
}

func (service *ServiceImpl) Upsert(ctx context.Context, user *bootstrap.User, notification model.UserVehicleNotifications) error {
	return service.repository.Upsert(user, notification)
}

func (service *ServiceImpl) Delete(ctx context.Context, user *bootstrap.User, notificationID string) error {
	return service.repository.Delete(user, notificationID)
}

func (service *ServiceImpl) DeleteAllByVehicle(ctx context.Context, user *bootstrap.User, vehicleID string) error {
	return service.repository.DeleteAllByVehicle(user, vehicleID)
}
