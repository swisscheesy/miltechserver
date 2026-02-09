package pol_products

type Repository interface {
	GetPolProducts() (PolProductsResponse, error)
}
