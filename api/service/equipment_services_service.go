package service

import (
	"miltechserver/api/request"
	"miltechserver/api/response"
	"miltechserver/bootstrap"
	"time"
)

type EquipmentServicesService interface {
	// CRUD Operations
	CreateEquipmentService(user *bootstrap.User, request request.CreateEquipmentServiceRequest) (*response.EquipmentServiceResponse, error)
	GetEquipmentServiceByID(user *bootstrap.User, shopID, serviceID string) (*response.EquipmentServiceResponse, error)
	UpdateEquipmentService(user *bootstrap.User, shopID string, request request.UpdateEquipmentServiceRequest) (*response.EquipmentServiceResponse, error)
	DeleteEquipmentService(user *bootstrap.User, shopID, serviceID string) error
	CompleteEquipmentService(user *bootstrap.User, shopID, serviceID string, request request.CompleteEquipmentServiceRequest) (*response.EquipmentServiceResponse, error)
	
	// Query Operations
	GetEquipmentServices(user *bootstrap.User, shopID string, request request.GetEquipmentServicesRequest) (*response.PaginatedEquipmentServicesResponse, error)
	GetServicesByEquipment(user *bootstrap.User, equipmentID string, limit, offset int, startDate, endDate *time.Time) (*response.PaginatedEquipmentServicesResponse, error)
	GetServicesInDateRange(user *bootstrap.User, shopID string, request request.GetCalendarServicesRequest) (*response.CalendarServicesResponse, error)
	GetOverdueServices(user *bootstrap.User, shopID string, request request.GetOverdueServicesRequest) (*response.OverdueServicesResponse, error)
	GetServicesDueSoon(user *bootstrap.User, shopID string, request request.GetDueSoonServicesRequest) (*response.DueSoonServicesResponse, error)
}