package calendar

import (
	"miltechserver/api/request"
	"miltechserver/api/response"
	"miltechserver/bootstrap"
)

type Service interface {
	GetCalendarServices(user *bootstrap.User, shopID string, req request.GetCalendarServicesRequest) (*response.CalendarServicesResponse, error)
}
