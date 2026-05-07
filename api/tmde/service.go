package tmde

import "miltechserver/.gen/miltech_ng/public/model"

type Service interface {
	LookupByNIIN(niin string) (model.TmdeRequirements, error)
	GetAllPaginated(page int) (TmdePageResponse, error)
}
