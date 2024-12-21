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

// UpsertQuickSaveItemByUser is a function that upserts a quick save item for a user
func (service *UserSavesServiceImpl) UpsertQuickSaveItemByUser(user *bootstrap.User, quick model.UserItemsQuick) error {
	return service.UserSavesRepository.UpsertQuickSaveItemByUser(user, quick)
}

// DeleteQuickSaveItemByUser is a function that deletes a quick save item for a user
func (service *UserSavesServiceImpl) DeleteQuickSaveItemByUser(user *bootstrap.User, quick model.UserItemsQuick) error {
	return service.UserSavesRepository.DeleteQuickSaveItemByUser(user, quick)
}

// DeleteAllQuickSaveItemsByUser is a function that deletes all quick save items for a user
func (service *UserSavesServiceImpl) DeleteAllQuickSaveItemsByUser(user *bootstrap.User) error {
	return service.UserSavesRepository.DeleteAllQuickSaveItemsByUser(user)
}

// UpsertQuickSaveItemListByUser is a function that upserts a list of quick save items for a user
func (service *UserSavesServiceImpl) UpsertQuickSaveItemListByUser(user *bootstrap.User, quickItems []model.UserItemsQuick) error {
	return service.UserSavesRepository.UpsertQuickSaveItemListByUser(user, quickItems)
}
