package service

import (
	"github.com/gin-gonic/gin"
	"miltechserver/bootstrap"
	"miltechserver/prisma/db"
)

type UserSavesService interface {
	GetQuickSaveItemsByUser(c *gin.Context, user *bootstrap.User) ([]db.UserItemsQuickModel, error)
}
