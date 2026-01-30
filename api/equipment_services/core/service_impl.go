package core

import (
	"fmt"
	"log/slog"
	"time"

	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/equipment_services/shared"
	"miltechserver/api/request"
	"miltechserver/api/response"
	"miltechserver/bootstrap"

	"github.com/google/uuid"
)

type ServiceImpl struct {
	repo             Repository
	authorization    *shared.Authorization
	usernameResolver shared.UsernameResolver
}

func NewService(repo Repository, authorization *shared.Authorization, usernameResolver shared.UsernameResolver) *ServiceImpl {
	return &ServiceImpl{
		repo:             repo,
		authorization:    authorization,
		usernameResolver: usernameResolver,
	}
}

func (service *ServiceImpl) Create(user *bootstrap.User, req request.CreateEquipmentServiceRequest) (*response.EquipmentServiceResponse, error) {
	if user == nil {
		return nil, shared.ErrUnauthorizedUser
	}

	if req.ServiceHours != nil && *req.ServiceHours < 0 {
		return nil, shared.ErrServiceHoursNegative
	}

	shopID, err := service.authorization.GetShopIDForEquipment(user, req.EquipmentID)
	if err != nil {
		return nil, fmt.Errorf("equipment access validation failed: %w", err)
	}

	if req.ListID != "" {
		listShopID, err := service.authorization.GetShopIDForList(user, req.ListID)
		if err != nil {
			return nil, fmt.Errorf("list access validation failed: %w", err)
		}
		if shopID != listShopID {
			return nil, shared.ErrShopMismatch
		}
	}

	now := time.Now()
	equipmentService := model.EquipmentServices{
		ID:           uuid.New().String(),
		ShopID:       shopID,
		EquipmentID:  req.EquipmentID,
		ListID:       req.ListID,
		Description:  req.Description,
		ServiceType:  req.ServiceType,
		CreatedBy:    user.UserID,
		IsCompleted:  req.IsCompleted,
		CreatedAt:    now,
		UpdatedAt:    now,
		ServiceDate:  req.ServiceDate,
		ServiceHours: req.ServiceHours,
	}

	if req.IsCompleted {
		if req.CompletionDate != nil {
			equipmentService.CompletionDate = req.CompletionDate
		} else {
			equipmentService.CompletionDate = &now
		}
	} else {
		equipmentService.CompletionDate = nil
	}

	createdService, err := service.repo.Create(user, equipmentService)
	if err != nil {
		slog.Error("Failed to create equipment service", "error", err, "user_id", user.UserID)
		return nil, fmt.Errorf("failed to create equipment service: %w", err)
	}

	username, err := service.usernameResolver.GetUsernameByUserID(createdService.CreatedBy)
	if err != nil {
		slog.Warn("Failed to get username, using fallback", "user_id", createdService.CreatedBy, "error", err)
		username = "Unknown User"
	}

	result := shared.MapServiceToResponse(*createdService, username)
	slog.Info("Equipment service created successfully", "service_id", createdService.ID, "user_id", user.UserID)
	return &result, nil
}

func (service *ServiceImpl) GetByID(user *bootstrap.User, shopID, serviceID string) (*response.EquipmentServiceResponse, error) {
	if user == nil {
		return nil, shared.ErrUnauthorizedUser
	}

	_, err := service.authorization.RequireServiceAccessByID(user, serviceID)
	if err != nil {
		return nil, err
	}

	equipmentService, err := service.repo.GetByID(user, serviceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get equipment service: %w", err)
	}

	username, err := service.usernameResolver.GetUsernameByUserID(equipmentService.CreatedBy)
	if err != nil {
		username = "Unknown User"
	}

	result := shared.MapServiceToResponse(*equipmentService, username)
	return &result, nil
}

func (service *ServiceImpl) Update(user *bootstrap.User, shopID string, req request.UpdateEquipmentServiceRequest) (*response.EquipmentServiceResponse, error) {
	if user == nil {
		return nil, shared.ErrUnauthorizedUser
	}

	if req.ServiceHours != nil && *req.ServiceHours < 0 {
		return nil, shared.ErrServiceHoursNegative
	}

	canModify, err := service.authorization.CanUserModifyService(user, shopID, req.ServiceID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify modify permissions: %w", err)
	}
	if !canModify {
		return nil, shared.ErrModifyDenied
	}

	now := time.Now()
	updateService := model.EquipmentServices{
		ID:           req.ServiceID,
		Description:  req.Description,
		ServiceType:  req.ServiceType,
		ListID:       req.ListID,
		IsCompleted:  req.IsCompleted,
		ServiceDate:  req.ServiceDate,
		ServiceHours: req.ServiceHours,
		UpdatedAt:    now,
	}

	if req.IsCompleted {
		if req.CompletionDate != nil {
			updateService.CompletionDate = req.CompletionDate
		} else {
			updateService.CompletionDate = &now
		}
	} else {
		updateService.CompletionDate = nil
	}

	updatedService, err := service.repo.Update(user, updateService)
	if err != nil {
		slog.Error("Failed to update equipment service", "error", err, "service_id", req.ServiceID, "user_id", user.UserID)
		return nil, fmt.Errorf("failed to update equipment service: %w", err)
	}

	username, err := service.usernameResolver.GetUsernameByUserID(updatedService.CreatedBy)
	if err != nil {
		username = "Unknown User"
	}

	result := shared.MapServiceToResponse(*updatedService, username)
	return &result, nil
}

func (service *ServiceImpl) Delete(user *bootstrap.User, shopID, serviceID string) error {
	if user == nil {
		return shared.ErrUnauthorizedUser
	}

	canDelete, err := service.authorization.CanUserDeleteService(user, shopID, serviceID)
	if err != nil {
		return fmt.Errorf("failed to verify delete permissions: %w", err)
	}
	if !canDelete {
		return shared.ErrDeleteDenied
	}

	err = service.repo.Delete(user, serviceID)
	if err != nil {
		slog.Error("Failed to delete equipment service", "error", err, "service_id", serviceID, "user_id", user.UserID)
		return fmt.Errorf("failed to delete equipment service: %w", err)
	}

	slog.Info("Equipment service deleted successfully", "service_id", serviceID, "user_id", user.UserID)
	return nil
}

var _ Service = (*ServiceImpl)(nil)
