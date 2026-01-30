package calendar

import (
	"time"

	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/bootstrap"
)

type Repository interface {
	GetInDateRange(user *bootstrap.User, shopID string, startDate, endDate time.Time, equipmentID *string) ([]model.EquipmentServices, error)
}
