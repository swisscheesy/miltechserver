package queries

import (
	"time"

	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/request"
	"miltechserver/bootstrap"
)

type Repository interface {
	GetByShop(user *bootstrap.User, shopID string, filters request.GetEquipmentServicesRequest) ([]model.EquipmentServices, int64, error)
	GetByEquipment(user *bootstrap.User, equipmentID string, limit, offset int, startDate, endDate *time.Time) ([]model.EquipmentServices, int64, error)
}
