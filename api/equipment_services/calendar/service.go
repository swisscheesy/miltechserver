package calendar

import (
	"context"
	"miltechserver/api/request"
	"miltechserver/api/response"
	"miltechserver/bootstrap"
)

type Service interface {
	GetCalendarServices(ctx context.Context, user *bootstrap.User, shopID string, req request.GetCalendarServicesRequest) (*response.CalendarServicesResponse, error)
}
