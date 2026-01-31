package lin

import (
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/response"
)

type Repository interface {
	SearchByPage(page int) (response.LINPageResponse, error)
	SearchByNIIN(niin string) ([]model.LookupLinNiin, error)
	SearchNIINByLIN(lin string) ([]model.LookupLinNiin, error)
}
