package repository

import (
	"context"
	"miltechserver/prisma/db"
)

type UserSavesRepository interface {
	GetQuickSaveItemsByUserId(ctx context.Context, userId string) ([]db.UserItemsQuickModel, error)
}
