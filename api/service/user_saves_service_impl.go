package service

import (
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/repository"
	"miltechserver/bootstrap"
)

type UserSavesServiceImpl struct {
	UserSavesRepository repository.UserSavesRepository
}

func NewUserSavesServiceImpl(userSavesRepository repository.UserSavesRepository) *UserSavesServiceImpl {
	return &UserSavesServiceImpl{UserSavesRepository: userSavesRepository}
}

// GetQuickSaveItemsByUser is a function that returns the quick save items of a user
func (service *UserSavesServiceImpl) GetQuickSaveItemsByUser(user *bootstrap.User) ([]model.UserItemsQuick, error) {
	return service.UserSavesRepository.GetQuickSaveItemsByUserId(user)
}
