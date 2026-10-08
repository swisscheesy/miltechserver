package core

import (
	"context"
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/bootstrap"
)

type Repository interface {
	Create(ctx context.Context, user *bootstrap.User, service model.EquipmentServices) (*model.EquipmentServices, error)
	GetByID(ctx context.Context, user *bootstrap.User, serviceID string) (*model.EquipmentServices, error)
	Update(ctx context.Context, user *bootstrap.User, service model.EquipmentServices) (*model.EquipmentServices, error)
	Delete(ctx context.Context, user *bootstrap.User, shopID, serviceID string) error
}
