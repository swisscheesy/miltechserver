package user_general

import (
	"miltechserver/api/auth"
	"miltechserver/bootstrap"
)

type ServiceImpl struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &ServiceImpl{repo: repo}
}

func (service *ServiceImpl) UpsertUser(user *bootstrap.User, userDto auth.UserDto) error {
	return service.repo.UpsertUser(user, userDto)
}

func (service *ServiceImpl) DeleteUser(uid string) error {
	return service.repo.DeleteUser(uid)
}

func (service *ServiceImpl) UpdateUserDisplayName(uid string, displayName string) error {
	return service.repo.UpdateUserDisplayName(uid, displayName)
}
