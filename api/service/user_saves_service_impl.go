package service

import (
	"github.com/gin-gonic/gin"
	"miltechserver/api/repository"
	"miltechserver/bootstrap"
	"miltechserver/prisma/db"
)

type UserSavesServiceImpl struct {
	UserSavesRepository repository.UserSavesRepository
}

func NewUserSavesServiceImpl(userSavesRepository repository.UserSavesRepository) *UserSavesServiceImpl {
	return &UserSavesServiceImpl{UserSavesRepository: userSavesRepository}
}

// GetQuickSaveItemsByUser is a function that returns the quick save items of a user
func (service *UserSavesServiceImpl) GetQuickSaveItemsByUser(c *gin.Context, user *bootstrap.User) ([]db.UserItemsQuickModel, error) {
	return service.UserSavesRepository.GetQuickSaveItemsByUserId(c, user)
}
