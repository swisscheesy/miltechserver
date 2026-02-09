package pol_products

type ServiceImpl struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &ServiceImpl{repo: repo}
}

func (service *ServiceImpl) GetPolProducts() (PolProductsResponse, error) {
	return service.repo.GetPolProducts()
}
