package help

import (
	"context"
	"miltechserver/.gen/miltech_ng/public/model"
)

type Service interface {
	FindByCode(ctx context.Context, code string) (model.Help, error)
}
