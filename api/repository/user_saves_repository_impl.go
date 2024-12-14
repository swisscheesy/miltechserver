package repository

import (
	"errors"
	"github.com/gin-gonic/gin"
	"log/slog"
	"miltechserver/bootstrap"
	"miltechserver/prisma/db"
)

type UserSavesRepositoryImpl struct {
	Db *db.PrismaClient
}

func NewUserSavesRepositoryImpl(db *db.PrismaClient) *UserSavesRepositoryImpl {
	return &UserSavesRepositoryImpl{Db: db}
}

func (repo *UserSavesRepositoryImpl) GetQuickSaveItemsByUserId(ctx *gin.Context, user *bootstrap.User) ([]db.UserItemsQuickModel, error) {
	if user != nil {
		items, _ := repo.Db.UserItemsQuick.FindMany(db.UserItemsQuick.UserID.Equals(user.UserID)).Exec(ctx)
		slog.Info("User saves retrieved", "user_id", user.UserID)
		return items, nil
	} else {
		return nil, errors.New("user not found")
	}

}
