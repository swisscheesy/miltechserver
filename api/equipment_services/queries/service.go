package queries

import (
	"context"

	"miltechserver/api/request"
	"miltechserver/api/response"
	"miltechserver/bootstrap"
)

type Service interface {
	GetByShop(ctx context.Context, user *bootstrap.User, shopID string, req request.GetEquipmentServicesRequest) (*response.PaginatedEquipmentServicesResponse, error)
	GetByEquipment(ctx context.Context, user *bootstrap.User, equipmentID string, req request.GetEquipmentServicesRequest) (*response.PaginatedEquipmentServicesResponse, error)
}
