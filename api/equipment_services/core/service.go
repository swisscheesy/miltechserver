package core

import (
	"miltechserver/api/request"
	"miltechserver/api/response"
	"miltechserver/bootstrap"
)

type Service interface {
	Create(user *bootstrap.User, req request.CreateEquipmentServiceRequest) (*response.EquipmentServiceResponse, error)
	GetByID(user *bootstrap.User, shopID, serviceID string) (*response.EquipmentServiceResponse, error)
	Update(user *bootstrap.User, shopID string, req request.UpdateEquipmentServiceRequest) (*response.EquipmentServiceResponse, error)
	Delete(user *bootstrap.User, shopID, serviceID string) error
}
