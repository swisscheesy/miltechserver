package service

import (
	"context"
	"miltechserver/api/repository"
	"miltechserver/model"
	"miltechserver/prisma/db"
	"strings"
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

func (service *ItemLookupServiceImpl) LookupNIINByLIN(ctx context.Context, lin string) ([]db.LookupLinNiinModel, error) {
	niinData, err := service.ItemLookupService.SearchNIINByLIN(ctx, strings.ToUpper(lin))

	if err != nil {
		return nil, err
	}

	return niinData, nil
}

func (service *ItemLookupServiceImpl) LookupUOCByPage(ctx context.Context, page int) (model.UOCPageResponse, error) {
	uocData, err := service.ItemLookupService.SearchUOCByPage(ctx, page)

	if err != nil {
		return model.UOCPageResponse{}, err
	}

	return uocData, nil
}

func (service *ItemLookupServiceImpl) LookupSpecificUOC(ctx context.Context, uoc string) ([]db.LookupUocModel, error) {
	uocData, err := service.ItemLookupService.SearchSpecificUOC(ctx, uoc)

	if err != nil {
		return nil, err
	}

	return uocData, nil
}

func (service *ItemLookupServiceImpl) LookupUOCByModel(ctx context.Context, model string) ([]db.LookupUocModel, error) {
	uocData, err := service.ItemLookupService.SearchUOCByModel(ctx, model)

	if err != nil {
		return nil, err
	}

	return uocData, nil
}
