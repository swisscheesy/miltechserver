package repository

import (
	"context"
	"errors"
	"miltechserver/helper"
	"miltechserver/model"
	"miltechserver/prisma/db"
)

type ItemQueryRepositoryImpl struct {
	Db *db.PrismaClient
}

func NewItemQueryRepositoryImpl(db *db.PrismaClient) *ItemQueryRepositoryImpl {
	return &ItemQueryRepositoryImpl{Db: db}
}

func (repo *ItemQueryRepositoryImpl) ShortItemSearchNiin(ctx context.Context, niin string) (model.ShortItem, error) {
	item, err := repo.Db.Nsn.FindFirst(db.Nsn.Niin.Equals(niin)).Exec(ctx)
	helper.ErrorPanic(err)

	name, _ := item.ItemName()
	itemNiin := item.Niin
	fsc, _ := item.Fsc()
	//hasAmdfData := false
	//hasFlisData := false

	itemData := model.ShortItem{
		ItemName:    name,
		Niin:        itemNiin,
		Fsc:         fsc,
		HasAmdfData: false,
		HasFlisData: false,
	}

	if errors.Is(err, db.ErrNotFound) {
		return itemData, errors.New("item not found")
	} else if err != nil {
		return itemData, errors.New("internal error occurred")
	} else {
		return itemData, nil
	}

}

//func (repo *ItemQueryRepositoryImpl) ShortItemSearchPart(ctx context.Context, part string) (response.StandardResponse, error) {
//	// Implementation here
//	return response.StandardResponse{}, nil
//}
//
//func (repo *ItemQueryRepositoryImpl) DetailedItemSearchNiin(ctx context.Context, niin string) (response.StandardResponse, error) {
//	// Implementation here
//	return response.StandardResponse{}, nil
//}
