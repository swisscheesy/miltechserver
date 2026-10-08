package queries

import (
	"context"
	"fmt"

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

func (service *ServiceImpl) GetByShop(ctx context.Context, user *bootstrap.User, shopID string, req request.GetEquipmentServicesRequest) (*response.PaginatedEquipmentServicesResponse, error) {
	if user == nil {
		return nil, shared.ErrUnauthorizedUser
	}

	if err := service.authorization.RequireShopMember(ctx, user, shopID); err != nil {
		return nil, err
	}

	services, totalCount, err := service.repo.GetByShop(ctx, user, shopID, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get equipment services: %w", err)
	}

	usernameCache := shared.NewUsernameCache(service.usernameResolver)
	responseServices, err := shared.MapServicesToResponses(ctx, services, usernameCache)
	if err != nil {
		return nil, fmt.Errorf("resolve equipment service usernames: %w", err)
	}

	return &response.PaginatedEquipmentServicesResponse{
		Services:   responseServices,
		TotalCount: totalCount,
		HasMore:    int64(req.Offset+req.Limit) < totalCount,
	}, nil
}

func (service *ServiceImpl) GetByEquipment(ctx context.Context, user *bootstrap.User, equipmentID string, req request.GetEquipmentServicesRequest) (*response.PaginatedEquipmentServicesResponse, error) {
	if user == nil {
		return nil, shared.ErrUnauthorizedUser
	}

	_, err := service.authorization.GetShopIDForEquipment(ctx, user, equipmentID)
	if err != nil {
		return nil, fmt.Errorf("equipment access validation failed: %w", err)
	}

	services, totalCount, err := service.repo.GetByEquipment(ctx, user, equipmentID, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get services by equipment: %w", err)
	}

	usernameCache := shared.NewUsernameCache(service.usernameResolver)
	responseServices, err := shared.MapServicesToResponses(ctx, services, usernameCache)
	if err != nil {
		return nil, fmt.Errorf("resolve equipment service usernames: %w", err)
	}

	return &response.PaginatedEquipmentServicesResponse{
		Services:   responseServices,
		TotalCount: totalCount,
		HasMore:    int64(req.Offset+req.Limit) < totalCount,
	}, nil
}

var _ Service = (*ServiceImpl)(nil)
