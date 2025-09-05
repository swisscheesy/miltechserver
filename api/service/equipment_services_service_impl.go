package service

import (
	"errors"
	"fmt"
	"log/slog"
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/repository"
	"miltechserver/api/request"
	"miltechserver/api/response"
	"miltechserver/bootstrap"
	"time"

	"github.com/google/uuid"
)

type EquipmentServicesServiceImpl struct {
	EquipmentServicesRepository repository.EquipmentServicesRepository
	ShopsRepository             repository.ShopsRepository
}

func NewEquipmentServicesServiceImpl(
	equipmentServicesRepo repository.EquipmentServicesRepository,
	shopsRepo repository.ShopsRepository,
) *EquipmentServicesServiceImpl {
	return &EquipmentServicesServiceImpl{
		EquipmentServicesRepository: equipmentServicesRepo,
		ShopsRepository:             shopsRepo,
	}
}

// CreateEquipmentService creates a new equipment service
func (service *EquipmentServicesServiceImpl) CreateEquipmentService(user *bootstrap.User, request request.CreateEquipmentServiceRequest) (*response.EquipmentServiceResponse, error) {
	if user == nil {
		return nil, errors.New("unauthorized user")
	}

	// Validate service hours if provided (must be non-negative)
	if request.ServiceHours != nil && *request.ServiceHours < 0 {
		return nil, errors.New("service_hours must be non-negative")
	}

	// Create service model
	now := time.Now()
	equipmentService := model.EquipmentServices{
		ID:           uuid.New().String(),
		EquipmentID:  request.EquipmentID,
		ListID:       request.ListID,
		Description:  request.Description,
		ServiceType:  request.ServiceType,
		CreatedBy:    user.UserID,
		IsCompleted:  request.IsCompleted,
		CreatedAt:    now,
		UpdatedAt:    now,
		ServiceDate:  request.ServiceDate,
		ServiceHours: request.ServiceHours,
	}

	// Handle completion_date logic: auto-set when is_completed is true, clear when false
	if request.IsCompleted {
		if request.CompletionDate != nil {
			equipmentService.CompletionDate = request.CompletionDate
		} else {
			equipmentService.CompletionDate = &now // Auto-set to current time
		}
	} else {
		equipmentService.CompletionDate = nil // Clear when not completed
	}

	createdService, err := service.EquipmentServicesRepository.CreateEquipmentService(user, equipmentService)
	if err != nil {
		slog.Error("Failed to create equipment service", "error", err, "user_id", user.UserID)
		return nil, fmt.Errorf("failed to create equipment service: %w", err)
	}

	// Get current username dynamically
	username, err := service.EquipmentServicesRepository.GetUsernameByUserID(createdService.CreatedBy)
	if err != nil {
		slog.Warn("Failed to get username, using fallback", "user_id", createdService.CreatedBy, "error", err)
		username = "Unknown User"
	}

	response := &response.EquipmentServiceResponse{
		ID:                createdService.ID,
		ShopID:            createdService.ShopID,
		EquipmentID:       createdService.EquipmentID,
		ListID:            createdService.ListID,
		Description:       createdService.Description,
		ServiceType:       createdService.ServiceType,
		CreatedBy:         createdService.CreatedBy,
		CreatedByUsername: username,
		IsCompleted:       createdService.IsCompleted,
		CreatedAt:         createdService.CreatedAt,
		UpdatedAt:         createdService.UpdatedAt,
		ServiceDate:       createdService.ServiceDate,
		ServiceHours:      createdService.ServiceHours,
		CompletionDate:    createdService.CompletionDate,
	}

	slog.Info("Equipment service created successfully", "service_id", createdService.ID, "user_id", user.UserID)
	return response, nil
}

// GetEquipmentServiceByID retrieves a specific equipment service by ID
func (service *EquipmentServicesServiceImpl) GetEquipmentServiceByID(user *bootstrap.User, shopID, serviceID string) (*response.EquipmentServiceResponse, error) {
	if user == nil {
		return nil, errors.New("unauthorized user")
	}

	// Repository layer now handles shop membership validation
	equipmentService, err := service.EquipmentServicesRepository.GetEquipmentServiceByID(user, serviceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get equipment service: %w", err)
	}

	// Get current username dynamically
	username, err := service.EquipmentServicesRepository.GetUsernameByUserID(equipmentService.CreatedBy)
	if err != nil {
		username = "Unknown User"
	}

	response := &response.EquipmentServiceResponse{
		ID:                equipmentService.ID,
		ShopID:            equipmentService.ShopID,
		EquipmentID:       equipmentService.EquipmentID,
		ListID:            equipmentService.ListID,
		Description:       equipmentService.Description,
		ServiceType:       equipmentService.ServiceType,
		CreatedBy:         equipmentService.CreatedBy,
		CreatedByUsername: username,
		IsCompleted:       equipmentService.IsCompleted,
		CreatedAt:         equipmentService.CreatedAt,
		UpdatedAt:         equipmentService.UpdatedAt,
		ServiceDate:       equipmentService.ServiceDate,
		ServiceHours:      equipmentService.ServiceHours,
		CompletionDate:    equipmentService.CompletionDate,
	}

	return response, nil
}

// UpdateEquipmentService updates an existing equipment service
func (service *EquipmentServicesServiceImpl) UpdateEquipmentService(user *bootstrap.User, shopID string, request request.UpdateEquipmentServiceRequest) (*response.EquipmentServiceResponse, error) {
	if user == nil {
		return nil, errors.New("unauthorized user")
	}

	// Validate service hours if provided
	if request.ServiceHours != nil && *request.ServiceHours < 0 {
		return nil, errors.New("service_hours must be non-negative")
	}

	// Check if user can modify this service
	canModify, err := service.canUserModifyService(user, shopID, request.ServiceID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify modify permissions: %w", err)
	}
	if !canModify {
		return nil, errors.New("access denied: only service creators or shop admins can modify services")
	}

	// Create update model
	now := time.Now()
	updateService := model.EquipmentServices{
		ID:           request.ServiceID,
		Description:  request.Description,
		ServiceType:  request.ServiceType,
		ListID:       request.ListID,
		IsCompleted:  request.IsCompleted,
		ServiceDate:  request.ServiceDate,
		ServiceHours: request.ServiceHours,
		UpdatedAt:    now,
	}

	// Handle completion_date logic: auto-set when is_completed is true, clear when false
	if request.IsCompleted {
		if request.CompletionDate != nil {
			updateService.CompletionDate = request.CompletionDate
		} else {
			updateService.CompletionDate = &now // Auto-set to current time
		}
	} else {
		updateService.CompletionDate = nil // Clear when not completed
	}

	updatedService, err := service.EquipmentServicesRepository.UpdateEquipmentService(user, updateService)
	if err != nil {
		slog.Error("Failed to update equipment service", "error", err, "service_id", request.ServiceID, "user_id", user.UserID)
		return nil, fmt.Errorf("failed to update equipment service: %w", err)
	}

	// Get current username dynamically
	username, err := service.EquipmentServicesRepository.GetUsernameByUserID(updatedService.CreatedBy)
	if err != nil {
		username = "Unknown User"
	}

	response := &response.EquipmentServiceResponse{
		ID:                updatedService.ID,
		ShopID:            updatedService.ShopID,
		EquipmentID:       updatedService.EquipmentID,
		ListID:            updatedService.ListID,
		Description:       updatedService.Description,
		ServiceType:       updatedService.ServiceType,
		CreatedBy:         updatedService.CreatedBy,
		CreatedByUsername: username,
		IsCompleted:       updatedService.IsCompleted,
		CreatedAt:         updatedService.CreatedAt,
		UpdatedAt:         updatedService.UpdatedAt,
		ServiceDate:       updatedService.ServiceDate,
		ServiceHours:      updatedService.ServiceHours,
		CompletionDate:    updatedService.CompletionDate,
	}

	return response, nil
}

// DeleteEquipmentService deletes an equipment service
func (service *EquipmentServicesServiceImpl) DeleteEquipmentService(user *bootstrap.User, shopID, serviceID string) error {
	if user == nil {
		return errors.New("unauthorized user")
	}

	// Check ownership or admin permissions (business logic layer)
	canDelete, err := service.canUserDeleteService(user, shopID, serviceID)
	if err != nil {
		return fmt.Errorf("failed to verify delete permissions: %w", err)
	}
	if !canDelete {
		return errors.New("access denied: only service creators or shop admins can delete services")
	}

	// Repository layer handles shop membership validation
	err = service.EquipmentServicesRepository.DeleteEquipmentService(user, serviceID)
	if err != nil {
		slog.Error("Failed to delete equipment service", "error", err, "service_id", serviceID, "user_id", user.UserID)
		return fmt.Errorf("failed to delete equipment service: %w", err)
	}

	slog.Info("Equipment service deleted successfully", "service_id", serviceID, "user_id", user.UserID)
	return nil
}

// GetEquipmentServices retrieves equipment services with filtering and pagination
func (service *EquipmentServicesServiceImpl) GetEquipmentServices(user *bootstrap.User, shopID string, request request.GetEquipmentServicesRequest) (*response.PaginatedEquipmentServicesResponse, error) {
	if user == nil {
		return nil, errors.New("unauthorized user")
	}

	// Check if user is shop member
	isMember, err := service.ShopsRepository.IsUserMemberOfShop(user, shopID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify shop membership: %w", err)
	}
	if !isMember {
		return nil, errors.New("access denied: user is not a member of this shop")
	}

	services, totalCount, err := service.EquipmentServicesRepository.GetEquipmentServices(user, shopID, request)
	if err != nil {
		return nil, fmt.Errorf("failed to get equipment services: %w", err)
	}

	// Convert to response format with usernames
	responseServices := make([]response.EquipmentServiceResponse, len(services))
	for i, svc := range services {
		username, err := service.EquipmentServicesRepository.GetUsernameByUserID(svc.CreatedBy)
		if err != nil {
			username = "Unknown User"
		}

		responseServices[i] = response.EquipmentServiceResponse{
			ID:                svc.ID,
			ShopID:            svc.ShopID,
			EquipmentID:       svc.EquipmentID,
			ListID:            svc.ListID,
			Description:       svc.Description,
			ServiceType:       svc.ServiceType,
			CreatedBy:         svc.CreatedBy,
			CreatedByUsername: username,
			IsCompleted:       svc.IsCompleted,
			CreatedAt:         svc.CreatedAt,
			UpdatedAt:         svc.UpdatedAt,
			ServiceDate:       svc.ServiceDate,
			ServiceHours:      svc.ServiceHours,
			CompletionDate:    svc.CompletionDate,
		}
	}

	return &response.PaginatedEquipmentServicesResponse{
		Services:   responseServices,
		TotalCount: totalCount,
		HasMore:    int64(request.Offset+request.Limit) < totalCount,
	}, nil
}

// GetServicesByEquipment retrieves services for a specific equipment
func (service *EquipmentServicesServiceImpl) GetServicesByEquipment(user *bootstrap.User, equipmentID string, limit, offset int, startDate, endDate *time.Time) (*response.PaginatedEquipmentServicesResponse, error) {
	if user == nil {
		return nil, errors.New("unauthorized user")
	}

	services, totalCount, err := service.EquipmentServicesRepository.GetServicesByEquipment(user, equipmentID, limit, offset, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get services by equipment: %w", err)
	}

	// Convert to response format with usernames
	responseServices := make([]response.EquipmentServiceResponse, len(services))
	for i, svc := range services {
		username, err := service.EquipmentServicesRepository.GetUsernameByUserID(svc.CreatedBy)
		if err != nil {
			username = "Unknown User"
		}

		responseServices[i] = response.EquipmentServiceResponse{
			ID:                svc.ID,
			ShopID:            svc.ShopID,
			EquipmentID:       svc.EquipmentID,
			ListID:            svc.ListID,
			Description:       svc.Description,
			ServiceType:       svc.ServiceType,
			CreatedBy:         svc.CreatedBy,
			CreatedByUsername: username,
			IsCompleted:       svc.IsCompleted,
			CreatedAt:         svc.CreatedAt,
			UpdatedAt:         svc.UpdatedAt,
			ServiceDate:       svc.ServiceDate,
			ServiceHours:      svc.ServiceHours,
			CompletionDate:    svc.CompletionDate,
		}
	}

	return &response.PaginatedEquipmentServicesResponse{
		Services:   responseServices,
		TotalCount: totalCount,
		HasMore:    int64(offset+limit) < totalCount,
	}, nil
}

// GetServicesInDateRange retrieves services within a specific date range for calendar view
func (service *EquipmentServicesServiceImpl) GetServicesInDateRange(user *bootstrap.User, shopID string, request request.GetCalendarServicesRequest) (*response.CalendarServicesResponse, error) {
	if user == nil {
		return nil, errors.New("unauthorized user")
	}

	// Check if user is shop member
	isMember, err := service.ShopsRepository.IsUserMemberOfShop(user, shopID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify shop membership: %w", err)
	}
	if !isMember {
		return nil, errors.New("access denied: user is not a member of this shop")
	}

	// Parse dates
	startDate, err := time.Parse(time.RFC3339, request.StartDate)
	if err != nil {
		return nil, fmt.Errorf("invalid start_date format: %w", err)
	}

	endDate, err := time.Parse(time.RFC3339, request.EndDate)
	if err != nil {
		return nil, fmt.Errorf("invalid end_date format: %w", err)
	}

	services, err := service.EquipmentServicesRepository.GetServicesInDateRange(user, shopID, startDate, endDate, request.EquipmentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get services in date range: %w", err)
	}

	// Convert to response format with usernames
	responseServices := make([]response.EquipmentServiceResponse, len(services))
	for i, svc := range services {
		username, err := service.EquipmentServicesRepository.GetUsernameByUserID(svc.CreatedBy)
		if err != nil {
			username = "Unknown User"
		}

		responseServices[i] = response.EquipmentServiceResponse{
			ID:                svc.ID,
			ShopID:            svc.ShopID,
			EquipmentID:       svc.EquipmentID,
			ListID:            svc.ListID,
			Description:       svc.Description,
			ServiceType:       svc.ServiceType,
			CreatedBy:         svc.CreatedBy,
			CreatedByUsername: username,
			IsCompleted:       svc.IsCompleted,
			CreatedAt:         svc.CreatedAt,
			UpdatedAt:         svc.UpdatedAt,
			ServiceDate:       svc.ServiceDate,
			ServiceHours:      svc.ServiceHours,
			CompletionDate:    svc.CompletionDate,
		}
	}

	return &response.CalendarServicesResponse{
		DateRange: struct {
			StartDate time.Time `json:"start_date"`
			EndDate   time.Time `json:"end_date"`
		}{
			StartDate: startDate,
			EndDate:   endDate,
		},
		Services:   responseServices,
		TotalCount: int64(len(responseServices)),
	}, nil
}

// GetOverdueServices retrieves services that are overdue
func (service *EquipmentServicesServiceImpl) GetOverdueServices(user *bootstrap.User, shopID string, request request.GetOverdueServicesRequest) (*response.OverdueServicesResponse, error) {
	if user == nil {
		return nil, errors.New("unauthorized user")
	}

	// Check if user is shop member
	isMember, err := service.ShopsRepository.IsUserMemberOfShop(user, shopID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify shop membership: %w", err)
	}
	if !isMember {
		return nil, errors.New("access denied: user is not a member of this shop")
	}

	overdueServices, err := service.EquipmentServicesRepository.GetOverdueServices(user, shopID, request.EquipmentID, request.Limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get overdue services: %w", err)
	}

	return &response.OverdueServicesResponse{
		OverdueServices: overdueServices,
		TotalCount:      int64(len(overdueServices)),
	}, nil
}

// GetServicesDueSoon retrieves services that are due soon
func (service *EquipmentServicesServiceImpl) GetServicesDueSoon(user *bootstrap.User, shopID string, request request.GetDueSoonServicesRequest) (*response.DueSoonServicesResponse, error) {
	if user == nil {
		return nil, errors.New("unauthorized user")
	}

	// Check if user is shop member
	isMember, err := service.ShopsRepository.IsUserMemberOfShop(user, shopID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify shop membership: %w", err)
	}
	if !isMember {
		return nil, errors.New("access denied: user is not a member of this shop")
	}

	dueSoonServices, err := service.EquipmentServicesRepository.GetServicesDueSoon(user, shopID, request.DaysAhead, request.EquipmentID, request.Limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get due soon services: %w", err)
	}

	return &response.DueSoonServicesResponse{
		DueSoonServices: dueSoonServices,
		TotalCount:      int64(len(dueSoonServices)),
	}, nil
}

// Helper method for permission checking (modify permissions)
func (service *EquipmentServicesServiceImpl) canUserModifyService(user *bootstrap.User, shopID, serviceID string) (bool, error) {
	// Check if user is shop admin
	isAdmin, err := service.ShopsRepository.IsUserShopAdmin(user, shopID)
	if err != nil {
		return false, err
	}
	if isAdmin {
		return true, nil
	}

	// Check if user owns the service
	ownsService, err := service.EquipmentServicesRepository.ValidateServiceOwnership(user, serviceID)
	if err != nil {
		return false, err
	}

	return ownsService, nil
}

// CompleteEquipmentService marks a service as completed with optional completion date
func (service *EquipmentServicesServiceImpl) CompleteEquipmentService(user *bootstrap.User, shopID, serviceID string, request request.CompleteEquipmentServiceRequest) (*response.EquipmentServiceResponse, error) {
	if user == nil {
		return nil, errors.New("unauthorized user")
	}

	// Check if user can modify this service
	canModify, err := service.canUserModifyService(user, shopID, serviceID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify modify permissions: %w", err)
	}
	if !canModify {
		return nil, errors.New("access denied: only service creators or shop admins can modify services")
	}

	// Use repository method to complete the service
	completedService, err := service.EquipmentServicesRepository.CompleteEquipmentService(user, serviceID, request.CompletionDate)
	if err != nil {
		slog.Error("Failed to complete equipment service", "error", err, "service_id", serviceID, "user_id", user.UserID)
		return nil, fmt.Errorf("failed to complete equipment service: %w", err)
	}

	// Get current username dynamically
	username, err := service.EquipmentServicesRepository.GetUsernameByUserID(completedService.CreatedBy)
	if err != nil {
		slog.Warn("Failed to get username, using fallback", "user_id", completedService.CreatedBy, "error", err)
		username = "Unknown User"
	}

	response := &response.EquipmentServiceResponse{
		ID:                completedService.ID,
		ShopID:            completedService.ShopID,
		EquipmentID:       completedService.EquipmentID,
		ListID:            completedService.ListID,
		Description:       completedService.Description,
		ServiceType:       completedService.ServiceType,
		CreatedBy:         completedService.CreatedBy,
		CreatedByUsername: username,
		IsCompleted:       completedService.IsCompleted,
		CreatedAt:         completedService.CreatedAt,
		UpdatedAt:         completedService.UpdatedAt,
		ServiceDate:       completedService.ServiceDate,
		ServiceHours:      completedService.ServiceHours,
		CompletionDate:    completedService.CompletionDate,
	}

	slog.Info("Equipment service completed successfully", "service_id", serviceID, "user_id", user.UserID)
	return response, nil
}

// Helper method for permission checking (delete permissions)
func (service *EquipmentServicesServiceImpl) canUserDeleteService(user *bootstrap.User, shopID, serviceID string) (bool, error) {
	// Check if user is shop admin
	isAdmin, err := service.ShopsRepository.IsUserShopAdmin(user, shopID)
	if err != nil {
		return false, err
	}
	if isAdmin {
		return true, nil
	}

	// Check if user owns the service
	ownsService, err := service.EquipmentServicesRepository.ValidateServiceOwnership(user, serviceID)
	if err != nil {
		return false, err
	}

	return ownsService, nil
}
