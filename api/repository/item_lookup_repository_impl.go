package repository

import (
	"context"
	"miltechserver/prisma/db"
)

var returnCount = 20

type ItemLokupRepositoryImpl struct {
	Db *db.PrismaClient
}

func NewItemLookupRepositoryImpl(db *db.PrismaClient) *ItemLokupRepositoryImpl {
	return &ItemLokupRepositoryImpl{Db: db}
}

func (repo *ItemLokupRepositoryImpl) SearchLINByPage(ctx context.Context, page int) ([]string, error) {

	//linData, _ := repo.Db.ArmyLineItemNumber.FindMany().Take(returnCount).Skip(returnCount * (page - 1)).Exec(ctx)

	return []string{}, nil
}
