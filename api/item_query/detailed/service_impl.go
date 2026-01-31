package detailed

import "miltechserver/api/response"

type ServiceImpl struct {
	repo Repository
}

func NewService(repo Repository) *ServiceImpl {
	return &ServiceImpl{repo: repo}
}

func (service *ServiceImpl) FindDetailedItem(niin string) (response.DetailedResponse, error) {
	return service.repo.GetDetailedItemData(niin)
}
