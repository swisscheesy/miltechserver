package service

import (
	"miltechserver/api/repository"
	"miltechserver/api/response"
)

type ItemDetailedServiceImpl struct {
	ItemDetailedRepository repository.ItemDetailedRepository
}

func NewItemDetailedServiceImpl(itemDetailedServiceRepository repository.ItemDetailedRepository) *ItemDetailedServiceImpl {
	return &ItemDetailedServiceImpl{ItemDetailedRepository: itemDetailedServiceRepository}
}

func (service *ItemDetailedServiceImpl) FindDetailedItem(niin string) (response.DetailedResponse, error) {
	return service.ItemDetailedRepository.GetDetailedItemData(niin)
}
