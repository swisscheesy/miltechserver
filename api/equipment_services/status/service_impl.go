package status

import (
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

func (service *ServiceImpl) GetOverdue(user *bootstrap.User, shopID string, req request.GetOverdueServicesRequest) (*response.OverdueServicesResponse, error) {
	if user == nil {
		return nil, shared.ErrUnauthorizedUser
	}

	if err := service.authorization.RequireShopMember(user, shopID); err != nil {
		return nil, err
	}

	overdueServices, err := service.repo.GetOverdue(user, shopID, req.EquipmentID, req.Limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get overdue services: %w", err)
	}

	usernameCache := shared.NewUsernameCache(service.usernameResolver)
	responses := make([]response.OverdueServiceResponse, len(overdueServices))
	for i, svc := range overdueServices {
		username, _ := usernameCache.GetUsernameByUserID(svc.CreatedBy)
		responses[i] = response.OverdueServiceResponse{
			EquipmentServiceResponse: shared.MapServiceToResponse(svc.EquipmentServices, username),
			DaysOverdue:              svc.DaysCount,
		}
	}

	return &response.OverdueServicesResponse{
		OverdueServices: responses,
		TotalCount:      int64(len(responses)),
	}, nil
}

func (service *ServiceImpl) GetDueSoon(user *bootstrap.User, shopID string, req request.GetDueSoonServicesRequest) (*response.DueSoonServicesResponse, error) {
	if user == nil {
		return nil, shared.ErrUnauthorizedUser
	}

	if err := service.authorization.RequireShopMember(user, shopID); err != nil {
		return nil, err
	}

	dueSoonServices, err := service.repo.GetDueSoon(user, shopID, req.DaysAhead, req.EquipmentID, req.Limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get due soon services: %w", err)
	}

	usernameCache := shared.NewUsernameCache(service.usernameResolver)
	responses := make([]response.DueSoonServiceResponse, len(dueSoonServices))
	for i, svc := range dueSoonServices {
		username, _ := usernameCache.GetUsernameByUserID(svc.CreatedBy)
		responses[i] = response.DueSoonServiceResponse{
			EquipmentServiceResponse: shared.MapServiceToResponse(svc.EquipmentServices, username),
			DaysUntilDue:             svc.DaysCount,
		}
	}

	return &response.DueSoonServicesResponse{
		DueSoonServices: responses,
		TotalCount:      int64(len(responses)),
	}, nil
}

var _ Service = (*ServiceImpl)(nil)
