package repository

import (
	"github.com/gin-gonic/gin"
	"miltechserver/bootstrap"
	"miltechserver/prisma/db"
)

type UserSavesRepository interface {
	GetQuickSaveItemsByUserId(ctx *gin.Context, user *bootstrap.User) ([]db.UserItemsQuickModel, error)
}
