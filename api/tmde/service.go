package tmde

import (
	"context"
	"miltechserver/.gen/miltech_ng/public/model"
)

type Service interface {
	LookupByNIIN(ctx context.Context, niin string) (model.TmdeIntervalMat, error)
	GetAllPaginated(ctx context.Context, page int) (TmdePageResponse, error)
}
