package service

import (
	"github.com/gin-gonic/gin"
	"miltechserver/api/repository"
	"miltechserver/api/response"
	"miltechserver/prisma/db"
	"strings"
)

type ItemLookupServiceImpl struct {
	ItemLookupService repository.ItemLookupRepository
}

func NewItemLookupServiceImpl(itemLookupRepository repository.ItemLookupRepository) *ItemLookupServiceImpl {
	return &ItemLookupServiceImpl{ItemLookupService: itemLookupRepository}
}

func (service *ItemLookupServiceImpl) LookupLINByPage(ctx *gin.Context, page int) (response.LINPageResponse, error) {
	linData, err := service.ItemLookupService.SearchLINByPage(ctx, page)

	if err != nil {
		return response.LINPageResponse{}, err
	}

	return linData, nil

}

func (service *ItemLookupServiceImpl) LookupLINByNIIN(ctx *gin.Context, niin string) ([]db.LookupLinNiinModel, error) {
	linData, err := service.ItemLookupService.SearchLINByNIIN(ctx, niin)

	if err != nil {
		return nil, err
	}

	return linData, nil

}

func (service *ItemLookupServiceImpl) LookupNIINByLIN(ctx *gin.Context, lin string) ([]db.LookupLinNiinModel, error) {
	niinData, err := service.ItemLookupService.SearchNIINByLIN(ctx, strings.ToUpper(lin))

	if err != nil {
		return nil, err
	}

	return niinData, nil
}

func (service *ItemLookupServiceImpl) LookupUOCByPage(ctx *gin.Context, page int) (response.UOCPageResponse, error) {
	uocData, err := service.ItemLookupService.SearchUOCByPage(ctx, page)

	if err != nil {
		return response.UOCPageResponse{}, err
	}

	return uocData, nil
}

func (service *ItemLookupServiceImpl) LookupSpecificUOC(ctx *gin.Context, uoc string) ([]db.LookupUocModel, error) {
	uocData, err := service.ItemLookupService.SearchSpecificUOC(ctx, uoc)

	if err != nil {
		return nil, err
	}

	return uocData, nil
}

func (service *ItemLookupServiceImpl) LookupUOCByModel(ctx *gin.Context, model string) ([]db.LookupUocModel, error) {
	uocData, err := service.ItemLookupService.SearchUOCByModel(ctx, model)

	if err != nil {
		return nil, err
	}

	return uocData, nil
}
