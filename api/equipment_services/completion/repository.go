package completion

import (
	"context"
	"time"

	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/bootstrap"
)

type Repository interface {
	Complete(ctx context.Context, user *bootstrap.User, shopID, serviceID string, completionDate *time.Time) (*model.EquipmentServices, error)
}
