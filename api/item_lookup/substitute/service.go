package substitute

import (
	"context"
	"miltechserver/.gen/miltech_ng/public/model"
)

type Service interface {
	LookupAll(ctx context.Context) ([]model.ArmySubstituteLin, error)
}
