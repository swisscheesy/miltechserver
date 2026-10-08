package pol_products

import "context"

type Service interface {
	GetPolProducts(ctx context.Context) (PolProductsResponse, error)
}
