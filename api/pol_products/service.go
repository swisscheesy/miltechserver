package pol_products

type Service interface {
	GetPolProducts() (PolProductsResponse, error)
}
