package completion

import (
	"fmt"
	"log/slog"

	"miltechserver/api/equipment_services/shared"
	"miltechserver/api/request"
	"miltechserver/api/response"
	"miltechserver/bootstrap"
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

func (service *ServiceImpl) Complete(user *bootstrap.User, shopID, serviceID string, req request.CompleteEquipmentServiceRequest) (*response.EquipmentServiceResponse, error) {
	if user == nil {
		return nil, shared.ErrUnauthorizedUser
	}

	canModify, err := service.authorization.CanUserModifyService(user, shopID, serviceID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify modify permissions: %w", err)
	}
	if !canModify {
		return nil, shared.ErrModifyDenied
	}

	completedService, err := service.repo.Complete(user, serviceID, req.CompletionDate)
	if err != nil {
		slog.Error("Failed to complete equipment service", "error", err, "service_id", serviceID, "user_id", user.UserID)
		return nil, fmt.Errorf("failed to complete equipment service: %w", err)
	}

	username, err := service.usernameResolver.GetUsernameByUserID(completedService.CreatedBy)
	if err != nil {
		slog.Warn("Failed to get username, using fallback", "user_id", completedService.CreatedBy, "error", err)
		username = "Unknown User"
	}

	result := shared.MapServiceToResponse(*completedService, username)
	slog.Info("Equipment service completed successfully", "service_id", serviceID, "user_id", user.UserID)
	return &result, nil
}

var _ Service = (*ServiceImpl)(nil)
