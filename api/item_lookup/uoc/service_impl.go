package uoc

import (
	"miltechserver/api/response"
	"strings"
)

type ServiceImpl struct {
	repo Repository
}

func NewService(repo Repository) *ServiceImpl {
	return &ServiceImpl{repo: repo}
}

func (service *ServiceImpl) LookupByPage(page int) (response.UOCPageResponse, error) {
	return service.repo.SearchByPage(page)
}

func (service *ServiceImpl) LookupSpecific(uoc string) (response.UOCPageResponse, error) {
	uocData, err := service.repo.SearchSpecific(strings.ToUpper(uoc))
	if err != nil {
		return response.UOCPageResponse{}, err
	}

	return response.UOCPageResponse{
		UOCs:       uocData,
		Count:      len(uocData),
		Page:       1,
		TotalPages: 1,
		IsLastPage: true,
	}, nil
}

func (service *ServiceImpl) LookupByModel(model string) (response.UOCPageResponse, error) {
	uocData, err := service.repo.SearchByModel(strings.ToUpper(model))
	if err != nil {
		return response.UOCPageResponse{}, err
	}

	return response.UOCPageResponse{
		UOCs:       uocData,
		Count:      len(uocData),
		Page:       1,
		TotalPages: 1,
		IsLastPage: true,
	}, nil
}
