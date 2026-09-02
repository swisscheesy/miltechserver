package lin

import (
	"context"
	"miltechserver/api/response"
	"strings"
)

type ServiceImpl struct {
	repo Repository
}

func NewService(repo Repository) *ServiceImpl {
	return &ServiceImpl{repo: repo}
}

func (service *ServiceImpl) LookupByPage(ctx context.Context, page int) (response.LINPageResponse, error) {
	return service.repo.SearchByPage(page)
}

func (service *ServiceImpl) LookupByNIIN(ctx context.Context, niin string) (response.LINPageResponse, error) {
	linData, err := service.repo.SearchByNIIN(niin)
	if err != nil {
		return response.LINPageResponse{}, err
	}

	return response.LINPageResponse{
		Lins:       linData,
		Count:      len(linData),
		Page:       1,
		TotalPages: 1,
		IsLastPage: true,
	}, nil
}

func (service *ServiceImpl) LookupNIINByLIN(ctx context.Context, lin string) (response.LINPageResponse, error) {
	linData, err := service.repo.SearchNIINByLIN(strings.ToUpper(lin))
	if err != nil {
		return response.LINPageResponse{}, err
	}

	return response.LINPageResponse{
		Lins:       linData,
		Count:      len(linData),
		Page:       1,
		TotalPages: 1,
		IsLastPage: true,
	}, nil
}
