package repository

import (
	"context"
	"miltechserver/prisma/db"
)

type UserSavesRepositoryImpl struct {
	Db *db.PrismaClient
}

func NewUserSavesRepositoryImpl(db *db.PrismaClient) *UserSavesRepositoryImpl {
	return &UserSavesRepositoryImpl{Db: db}
}

func (repo *UserSavesRepositoryImpl) GetQuickSaveItemsByUserId(ctx context.Context, userId string) ([]db.UserItemsQuickModel, error) {
	items, _ := repo.Db.UserItemsQuick.FindMany(db.UserItemsQuick.UserID.Equals(userId)).Exec(ctx)

	_, userErr := repo.Db.Users.FindFirst(db.Users.UID.Equals(userId)).Exec(ctx)
	if userErr != nil {
		return items, userErr
	}

	return items, nil

}
