package calendar

import (
	"context"
	"time"

	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/bootstrap"
)

type Repository interface {
	GetInDateRange(ctx context.Context, user *bootstrap.User, shopID string, startDate, endDate time.Time, equipmentID *string) ([]model.EquipmentServices, error)
}
