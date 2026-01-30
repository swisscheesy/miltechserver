package status

import (
	"miltechserver/api/request"
	"miltechserver/api/response"
	"miltechserver/bootstrap"
)

type Service interface {
	GetOverdue(user *bootstrap.User, shopID string, req request.GetOverdueServicesRequest) (*response.OverdueServicesResponse, error)
	GetDueSoon(user *bootstrap.User, shopID string, req request.GetDueSoonServicesRequest) (*response.DueSoonServicesResponse, error)
}
