package queries

import (
	"context"

	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/request"
	"miltechserver/bootstrap"
)

type Repository interface {
	GetByShop(ctx context.Context, user *bootstrap.User, shopID string, filters request.GetEquipmentServicesRequest) ([]model.EquipmentServices, int64, error)
	GetByEquipment(ctx context.Context, user *bootstrap.User, equipmentID string, req request.GetEquipmentServicesRequest) ([]model.EquipmentServices, int64, error)
}
