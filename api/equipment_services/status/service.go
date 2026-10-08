package status

import (
	"context"
	"miltechserver/api/request"
	"miltechserver/api/response"
	"miltechserver/bootstrap"
)

type Service interface {
	GetOverdue(ctx context.Context, user *bootstrap.User, shopID string, req request.GetOverdueServicesRequest) (*response.OverdueServicesResponse, error)
	GetDueSoon(ctx context.Context, user *bootstrap.User, shopID string, req request.GetDueSoonServicesRequest) (*response.DueSoonServicesResponse, error)
}
