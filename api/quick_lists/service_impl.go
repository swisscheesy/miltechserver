package quick_lists

type ServiceImpl struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &ServiceImpl{repo: repo}
}

func (service *ServiceImpl) GetQuickListClothing() (QuickListsClothingResponse, error) {
	clothingData, err := service.repo.GetQuickListClothing()
	if err != nil {
		return QuickListsClothingResponse{}, err
	}
	return clothingData, nil
}

func (service *ServiceImpl) GetQuickListWheels() (QuickListsWheelsResponse, error) {
	wheelsData, err := service.repo.GetQuickListWheels()
	if err != nil {
		return QuickListsWheelsResponse{}, err
	}
	return wheelsData, nil
}

func (service *ServiceImpl) GetQuickListBatteries() (QuickListsBatteryResponse, error) {
	batteriesData, err := service.repo.GetQuickListBatteries()
	if err != nil {
		return QuickListsBatteryResponse{}, err
	}
	return batteriesData, nil
}
