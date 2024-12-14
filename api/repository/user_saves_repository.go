package repository

import (
	"github.com/gin-gonic/gin"
	"miltechserver/prisma/db"
)

type UserSavesRepository interface {
	GetQuickSaveItemsByUserId(ctx *gin.Context, userId string) ([]db.UserItemsQuickModel, error)
}
