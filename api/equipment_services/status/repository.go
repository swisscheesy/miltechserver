package status

import (
	"context"
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/bootstrap"
)

type ServiceWithDays struct {
	model.EquipmentServices
	DaysCount int
}

type Repository interface {
	GetOverdue(ctx context.Context, user *bootstrap.User, shopID string, equipmentID *string, limit int) ([]ServiceWithDays, error)
	GetDueSoon(ctx context.Context, user *bootstrap.User, shopID string, daysAhead int, equipmentID *string, limit int) ([]ServiceWithDays, error)
}
