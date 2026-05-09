package tmde

import "miltechserver/.gen/miltech_ng/public/model"

type Repository interface {
	GetByNIIN(niin string) (model.TmdeIntervalMat, error)
	GetAllPaginated(page int) (TmdePageResponse, error)
}
