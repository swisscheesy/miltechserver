package cage

import "miltechserver/.gen/miltech_ng/public/model"

type Repository interface {
	SearchByCode(cage string) ([]model.CageAddress, error)
}
