package core

import (
	"context"
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

func (service *ServiceImpl) Create(ctx context.Context, user *bootstrap.User, shopID string, req request.CreateEquipmentServiceRequest) (*response.EquipmentServiceResponse, error) {
	if user == nil {
		return nil, shared.ErrUnauthorizedUser
	}

	if req.ServiceHours != nil && *req.ServiceHours < 0 {
		return nil, shared.ErrServiceHoursNegative
	}

	actualShopID, err := service.authorization.GetShopIDForEquipment(ctx, user, req.EquipmentID)
	if err != nil {
		return nil, fmt.Errorf("equipment access validation failed: %w", err)
	}

	if actualShopID != shopID {
		return nil, shared.ErrShopMismatch
	}
	if req.ListID != "" {
		listShopID, err := service.authorization.GetShopIDForList(ctx, user, req.ListID)
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

	createdService, err := service.repo.Create(ctx, user, equipmentService)
	if err != nil {
		slog.Error("Failed to create equipment service", "error", err, "user_id", user.UserID)
		return nil, fmt.Errorf("failed to create equipment service: %w", err)
	}

	username, err := service.usernameResolver.GetUsernameByUserID(ctx, createdService.CreatedBy)
	if err != nil {
		return nil, fmt.Errorf("resolve equipment service username: %w", err)
	}

	result := shared.MapServiceToResponse(*createdService, username)
	slog.Info("Equipment service created successfully", "service_id", createdService.ID, "user_id", user.UserID)
	return &result, nil
}

func (service *ServiceImpl) GetByID(ctx context.Context, user *bootstrap.User, shopID, serviceID string) (*response.EquipmentServiceResponse, error) {
	if user == nil {
		return nil, shared.ErrUnauthorizedUser
	}

	actualShopID, err := service.authorization.RequireServiceAccessByID(ctx, user, serviceID)
	if err != nil {
		return nil, err
	}

	if actualShopID != shopID {
		return nil, shared.ErrShopMismatch
	}
	equipmentService, err := service.repo.GetByID(ctx, user, serviceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get equipment service: %w", err)
	}

	if equipmentService.ShopID != shopID {
		return nil, shared.ErrShopMismatch
	}
	username, err := service.usernameResolver.GetUsernameByUserID(ctx, equipmentService.CreatedBy)
	if err != nil {
		return nil, fmt.Errorf("resolve equipment service username: %w", err)
	}

	result := shared.MapServiceToResponse(*equipmentService, username)
	return &result, nil
}

func (service *ServiceImpl) Update(ctx context.Context, user *bootstrap.User, shopID string, req request.UpdateEquipmentServiceRequest) (*response.EquipmentServiceResponse, error) {
	if user == nil {
		return nil, shared.ErrUnauthorizedUser
	}

	if req.ServiceHours != nil && *req.ServiceHours < 0 {
		return nil, shared.ErrServiceHoursNegative
	}

	canModify, err := service.authorization.CanUserModifyService(ctx, user, shopID, req.ServiceID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify modify permissions: %w", err)
	}
	if !canModify {
		return nil, shared.ErrModifyDenied
	}

	now := time.Now()
	updateService := model.EquipmentServices{
		ID:             req.ServiceID,
		ShopID:         shopID,
		Description:    req.Description,
		ServiceType:    req.ServiceType,
		ListID:         req.ListID,
		IsCompleted:    req.IsCompleted,
		ServiceDate:    req.ServiceDate,
		ServiceHours:   req.ServiceHours,
		UpdatedAt:      now,
		CompletionDate: req.CompletionDate,
	}

	updatedService, err := service.repo.Update(ctx, user, updateService)
	if err != nil {
		slog.Error("Failed to update equipment service", "error", err, "service_id", req.ServiceID, "user_id", user.UserID)
		return nil, fmt.Errorf("failed to update equipment service: %w", err)
	}

	username, err := service.usernameResolver.GetUsernameByUserID(ctx, updatedService.CreatedBy)
	if err != nil {
		return nil, fmt.Errorf("resolve equipment service username: %w", err)
	}

	result := shared.MapServiceToResponse(*updatedService, username)
	return &result, nil
}

func (service *ServiceImpl) Delete(ctx context.Context, user *bootstrap.User, shopID, serviceID string) error {
	if user == nil {
		return shared.ErrUnauthorizedUser
	}

	canDelete, err := service.authorization.CanUserDeleteService(ctx, user, shopID, serviceID)
	if err != nil {
		return fmt.Errorf("failed to verify delete permissions: %w", err)
	}
	if !canDelete {
		return shared.ErrDeleteDenied
	}

	err = service.repo.Delete(ctx, user, shopID, serviceID)
	if err != nil {
		slog.Error("Failed to delete equipment service", "error", err, "service_id", serviceID, "user_id", user.UserID)
		return fmt.Errorf("failed to delete equipment service: %w", err)
	}

	slog.Info("Equipment service deleted successfully", "service_id", serviceID, "user_id", user.UserID)
	return nil
}

var _ Service = (*ServiceImpl)(nil)
