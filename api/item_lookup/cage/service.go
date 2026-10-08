package cage

import (
	"context"
	"miltechserver/.gen/miltech_ng/public/model"
)

type Service interface {
	LookupByCode(ctx context.Context, cage string) ([]model.CageAddress, error)
}
