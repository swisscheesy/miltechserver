package core

import (
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/bootstrap"
)

type Repository interface {
	Create(user *bootstrap.User, service model.EquipmentServices) (*model.EquipmentServices, error)
	GetByID(user *bootstrap.User, serviceID string) (*model.EquipmentServices, error)
	Update(user *bootstrap.User, service model.EquipmentServices) (*model.EquipmentServices, error)
	Delete(user *bootstrap.User, serviceID string) error
}
