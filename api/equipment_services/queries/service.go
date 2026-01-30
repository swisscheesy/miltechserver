package queries

import (
	"time"

	"miltechserver/api/request"
	"miltechserver/api/response"
	"miltechserver/bootstrap"
)

type Service interface {
	GetByShop(user *bootstrap.User, shopID string, req request.GetEquipmentServicesRequest) (*response.PaginatedEquipmentServicesResponse, error)
	GetByEquipment(user *bootstrap.User, equipmentID string, limit, offset int, startDate, endDate *time.Time) (*response.PaginatedEquipmentServicesResponse, error)
}
