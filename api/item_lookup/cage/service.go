package cage

import "miltechserver/.gen/miltech_ng/public/model"

type Service interface {
	LookupByCode(cage string) ([]model.CageAddress, error)
}
