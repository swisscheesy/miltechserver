package quick_lists

import "context"

type ServiceImpl struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &ServiceImpl{repo: repo}
}

func (service *ServiceImpl) GetQuickListClothing(ctx context.Context) (QuickListsClothingResponse, error) {
	clothingData, err := service.repo.GetQuickListClothing()
	if err != nil {
		return QuickListsClothingResponse{}, err
	}
	return clothingData, nil
}

func (service *ServiceImpl) GetQuickListWheels(ctx context.Context) (QuickListsWheelsResponse, error) {
	wheelsData, err := service.repo.GetQuickListWheels()
	if err != nil {
		return QuickListsWheelsResponse{}, err
	}
	return wheelsData, nil
}

func (service *ServiceImpl) GetQuickListBatteries(ctx context.Context) (QuickListsBatteryResponse, error) {
	batteriesData, err := service.repo.GetQuickListBatteries()
	if err != nil {
		return QuickListsBatteryResponse{}, err
	}
	return batteriesData, nil
}
