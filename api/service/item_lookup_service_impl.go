package service

import (
	"miltechserver/api/repository"
)

type ItemLookupServiceImpl struct {
	ItemLookupService repository.ItemLookupRepository
}

func NewItemLookupServiceImpl(itemLookupService repository.ItemLookupRepository) *ItemLookupServiceImpl {
	return &ItemLookupServiceImpl{ItemLookupService: itemLookupService}
}

func (service *ItemLookupServiceImpl) LookupLINByPage(page int) ([]string, error) {
	return service.ItemLookupService.SearchLINByPage(page)
}
