package service

import (
	"context"
	"miltechserver/api/repository"
	"miltechserver/model"
	"miltechserver/prisma/db"
)

type ItemLookupServiceImpl struct {
	ItemLookupService repository.ItemLookupRepository
}

func NewItemLookupServiceImpl(itemLookupRepository repository.ItemLookupRepository) *ItemLookupServiceImpl {
	return &ItemLookupServiceImpl{ItemLookupService: itemLookupRepository}
}

func (service *ItemLookupServiceImpl) LookupLINByPage(ctx context.Context, page int) (model.LINPageResponse, error) {
	linData, err := service.ItemLookupService.SearchLINByPage(ctx, page)

	if err != nil {
		return model.LINPageResponse{}, err
	}

	return linData, nil

}

func (service *ItemLookupServiceImpl) LookupLINByNIIN(ctx context.Context, niin string) ([]db.LookupLinNiinModel, error) {
	linData, err := service.ItemLookupService.SearchLINByNIIN(ctx, niin)

	if err != nil {
		return nil, err
	}

	return linData, nil

}
