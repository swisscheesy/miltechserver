package repository

import (
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/request"
	"miltechserver/api/response"
	"miltechserver/bootstrap"
	"time"
)

type EquipmentServicesRepository interface {
	// CRUD Operations
	CreateEquipmentService(user *bootstrap.User, service model.EquipmentServices) (*model.EquipmentServices, error)
	GetEquipmentServiceByID(user *bootstrap.User, serviceID string) (*model.EquipmentServices, error)
	UpdateEquipmentService(user *bootstrap.User, service model.EquipmentServices) (*model.EquipmentServices, error)
	DeleteEquipmentService(user *bootstrap.User, serviceID string) error
	CompleteEquipmentService(user *bootstrap.User, serviceID string, completionDate *time.Time) (*model.EquipmentServices, error)
	
	// Query Operations
	GetEquipmentServices(user *bootstrap.User, shopID string, filters request.GetEquipmentServicesRequest) ([]model.EquipmentServices, int64, error)
	GetServicesByEquipment(user *bootstrap.User, equipmentID string, limit, offset int, startDate, endDate *time.Time) ([]model.EquipmentServices, int64, error)
	GetServicesInDateRange(user *bootstrap.User, shopID string, startDate, endDate time.Time, equipmentID *string) ([]model.EquipmentServices, error)
	GetOverdueServices(user *bootstrap.User, shopID string, equipmentID *string, limit int) ([]response.OverdueServiceResponse, error)
	GetServicesDueSoon(user *bootstrap.User, shopID string, daysAhead int, equipmentID *string, limit int) ([]response.DueSoonServiceResponse, error)
	
	// Validation Helpers
	ValidateServiceOwnership(user *bootstrap.User, serviceID string) (bool, error)
	ValidateServiceAccess(user *bootstrap.User, shopID string) (bool, error)         // Validates shop membership
	ValidateEquipmentAccess(user *bootstrap.User, equipmentID string) (string, error) // Returns shopID
	ValidateListAccess(user *bootstrap.User, listID string) (string, error)          // Returns shopID
	
	// Username Lookup
	GetUsernameByUserID(userID string) (string, error) // Dynamically fetch current username
}