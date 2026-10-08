package completion

import (
	"context"
	"miltechserver/api/request"
	"miltechserver/api/response"
	"miltechserver/bootstrap"
)

type Service interface {
	Complete(ctx context.Context, user *bootstrap.User, shopID, serviceID string, req request.CompleteEquipmentServiceRequest) (*response.EquipmentServiceResponse, error)
}
