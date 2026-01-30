package completion

import (
	"time"

	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/bootstrap"
)

type Repository interface {
	Complete(user *bootstrap.User, serviceID string, completionDate *time.Time) (*model.EquipmentServices, error)
}
