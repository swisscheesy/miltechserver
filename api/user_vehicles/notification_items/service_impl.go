package notification_items

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

func (service *ServiceImpl) GetByUser(user *bootstrap.User) ([]model.UserNotificationItems, error) {
	items, err := service.repository.GetByUserID(user)
	if items == nil {
		return []model.UserNotificationItems{}, nil
	}
	return items, err
}

func (service *ServiceImpl) GetByNotification(user *bootstrap.User, notificationID string) ([]model.UserNotificationItems, error) {
	items, err := service.repository.GetByNotificationID(user, notificationID)
	if items == nil {
		return []model.UserNotificationItems{}, nil
	}
	return items, err
}

func (service *ServiceImpl) GetByID(user *bootstrap.User, itemID string) (*model.UserNotificationItems, error) {
	return service.repository.GetByID(user, itemID)
}

func (service *ServiceImpl) Upsert(user *bootstrap.User, item model.UserNotificationItems) error {
	return service.repository.Upsert(user, item)
}

func (service *ServiceImpl) UpsertBatch(user *bootstrap.User, items []model.UserNotificationItems) error {
	return service.repository.UpsertBatch(user, items)
}

func (service *ServiceImpl) Delete(user *bootstrap.User, itemID string) error {
	return service.repository.Delete(user, itemID)
}

func (service *ServiceImpl) DeleteAllByNotification(user *bootstrap.User, notificationID string) error {
	return service.repository.DeleteAllByNotification(user, notificationID)
}
