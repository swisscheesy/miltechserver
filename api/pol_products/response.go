package pol_products

import "miltechserver/.gen/miltech_ng/public/model"

type PolProductsResponse struct {
	Products []model.PolProducts `json:"products"`
	Count    int                 `json:"count"`
}
