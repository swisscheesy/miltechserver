package service

import (
	"miltechserver/api/repository"
	"miltechserver/bootstrap"

	"miltechserver/api/auth"
)

type UserGeneralServiceImpl struct {
	UserGeneralRepository repository.UserGeneralRepository
}

func NewUserGeneralServiceImpl(userGeneralRepository repository.UserGeneralRepository) *UserGeneralServiceImpl {
	return &UserGeneralServiceImpl{UserGeneralRepository: userGeneralRepository}
}

func (service *UserGeneralServiceImpl) UpsertUser(user *bootstrap.User, userDto auth.UserDto) error {
	return service.UserGeneralRepository.UpsertUser(user, userDto)
}

func (service *UserGeneralServiceImpl) DeleteUser(uid string) error {
	return service.UserGeneralRepository.DeleteUser(uid)
}
