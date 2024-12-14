package service

import "miltechserver/api/repository"

type UserSavesServiceImpl struct {
	UserSavesRepository repository.UserSavesRepository
}

func NewUserSavesServiceImpl(userSavesRepository repository.UserSavesRepository) *UserSavesServiceImpl {
	return &UserSavesServiceImpl{UserSavesRepository: userSavesRepository}
}
