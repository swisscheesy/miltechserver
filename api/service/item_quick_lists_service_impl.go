package service

import (
	"miltechserver/api/repository"
	"miltechserver/api/response"
)

type ItemQuickListsServiceImpl struct {
	ItemQuickListsRepository repository.ItemQuickListsRepository
}

func NewItemQuickListsServiceImpl(itemQuickListsRepository repository.ItemQuickListsRepository) *ItemQuickListsServiceImpl {
	return &ItemQuickListsServiceImpl{ItemQuickListsRepository: itemQuickListsRepository}
}

func (service *ItemQuickListsServiceImpl) GetQuickListClothing() (response.QuickListsClothingResponse, error) {
	clothingData, err := service.ItemQuickListsRepository.GetQuickListClothing()
	if err != nil {
		return response.QuickListsClothingResponse{}, err
	}
	return clothingData, nil
}

func (service *ItemQuickListsServiceImpl) GetQuickListWheels() (response.QuickListsWheelsResponse, error) {
	wheelsData, err := service.ItemQuickListsRepository.GetQuickListWheels()
	if err != nil {
		return response.QuickListsWheelsResponse{}, err
	}
	return wheelsData, nil
}

func (service *ItemQuickListsServiceImpl) GetQuickListBatteries() (response.QuickListsBatteryResponse, error) {
	batteriesData, err := service.ItemQuickListsRepository.GetQuickListBatteries()
	if err != nil {
		return response.QuickListsBatteryResponse{}, err
	}
	return batteriesData, nil
}
