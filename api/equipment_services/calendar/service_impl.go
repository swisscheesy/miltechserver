package calendar

import (
	"fmt"
	"time"

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

func (service *ServiceImpl) GetCalendarServices(user *bootstrap.User, shopID string, req request.GetCalendarServicesRequest) (*response.CalendarServicesResponse, error) {
	if user == nil {
		return nil, shared.ErrUnauthorizedUser
	}

	if err := service.authorization.RequireShopMember(user, shopID); err != nil {
		return nil, err
	}

	startDate, err := time.Parse(time.RFC3339, req.StartDate)
	if err != nil {
		return nil, fmt.Errorf("invalid start_date format: %w", err)
	}

	endDate, err := time.Parse(time.RFC3339, req.EndDate)
	if err != nil {
		return nil, fmt.Errorf("invalid end_date format: %w", err)
	}

	services, err := service.repo.GetInDateRange(user, shopID, startDate, endDate, req.EquipmentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get services in date range: %w", err)
	}

	usernameCache := shared.NewUsernameCache(service.usernameResolver)
	responseServices := shared.MapServicesToResponses(services, usernameCache)

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

var _ Service = (*ServiceImpl)(nil)
