package pol_products

import "context"

type ServiceImpl struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &ServiceImpl{repo: repo}
}

func (service *ServiceImpl) GetPolProducts(ctx context.Context) (PolProductsResponse, error) {
	return service.repo.GetPolProducts()
}
