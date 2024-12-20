package service

import (
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/repository"
	"miltechserver/api/response"
	"strings"
)

type ItemLookupServiceImpl struct {
	ItemLookupService repository.ItemLookupRepository
}

func NewItemLookupServiceImpl(itemLookupRepository repository.ItemLookupRepository) *ItemLookupServiceImpl {
	return &ItemLookupServiceImpl{ItemLookupService: itemLookupRepository}
}

func (service *ItemLookupServiceImpl) LookupLINByPage(page int) (response.LINPageResponse, error) {
	linData, err := service.ItemLookupService.SearchLINByPage(page)

	if err != nil {
		return response.LINPageResponse{}, err
	}

	return linData, nil

}

func (service *ItemLookupServiceImpl) LookupLINByNIIN(niin string) ([]model.LookupLinNiin, error) {
	linData, err := service.ItemLookupService.SearchLINByNIIN(niin)

	if err != nil {
		return nil, err
	}

	return linData, nil

}

func (service *ItemLookupServiceImpl) LookupNIINByLIN(lin string) ([]model.LookupLinNiin, error) {
	niinData, err := service.ItemLookupService.SearchNIINByLIN(strings.ToUpper(lin))

	if err != nil {
		return nil, err
	}

	return niinData, nil
}

func (service *ItemLookupServiceImpl) LookupUOCByPage(page int) (response.UOCPageResponse, error) {
	uocData, err := service.ItemLookupService.SearchUOCByPage(page)

	if err != nil {
		return response.UOCPageResponse{}, err
	}

	return uocData, nil
}

func (service *ItemLookupServiceImpl) LookupSpecificUOC(uoc string) ([]model.LookupUoc, error) {
	uocData, err := service.ItemLookupService.SearchSpecificUOC(strings.ToUpper(uoc))

	if err != nil {
		return nil, err
	}

	return uocData, nil
}

//func (service *ItemLookupServiceImpl) LookupUOCByModel(model string) ([]db.LookupUocModel, error) {
//	uocData, err := service.ItemLookupService.SearchUOCByModel(ctx, model)
//
//	if err != nil {
//		return nil, err
//	}
//
//	return uocData, nil
//}
