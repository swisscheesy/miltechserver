package images

import (
	"miltechserver/api/user_saves/shared"
	"miltechserver/bootstrap"
)

type ServiceImpl struct {
	repo Repository
}

func NewService(repo Repository) *ServiceImpl {
	return &ServiceImpl{repo: repo}
}

func (service *ServiceImpl) Upload(user *bootstrap.User, itemID string, tableType string, imageData []byte) (string, error) {
	if user == nil {
		return "", shared.ErrUserNotFound
	}

	return service.repo.Upload(user, itemID, tableType, imageData)
}

func (service *ServiceImpl) Delete(user *bootstrap.User, itemID string, tableType string) error {
	if user == nil {
		return shared.ErrUserNotFound
	}

	return service.repo.Delete(user, itemID, tableType)
}

func (service *ServiceImpl) Get(user *bootstrap.User, itemID string, tableType string) ([]byte, string, error) {
	if user == nil {
		return nil, "", shared.ErrUserNotFound
	}

	return service.repo.Get(user, itemID, tableType)
}
