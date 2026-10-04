package core

import (
	"context"
	"miltechserver/api/request"
	"miltechserver/api/response"
	"miltechserver/bootstrap"
)

type Service interface {
	Create(ctx context.Context, user *bootstrap.User, shopID string, req request.CreateEquipmentServiceRequest) (*response.EquipmentServiceResponse, error)
	GetByID(ctx context.Context, user *bootstrap.User, shopID, serviceID string) (*response.EquipmentServiceResponse, error)
	Update(ctx context.Context, user *bootstrap.User, shopID string, req request.UpdateEquipmentServiceRequest) (*response.EquipmentServiceResponse, error)
	Delete(ctx context.Context, user *bootstrap.User, shopID, serviceID string) error
}
