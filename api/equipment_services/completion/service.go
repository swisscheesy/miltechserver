package completion

import (
	"miltechserver/api/request"
	"miltechserver/api/response"
	"miltechserver/bootstrap"
)

type Service interface {
	Complete(user *bootstrap.User, shopID, serviceID string, req request.CompleteEquipmentServiceRequest) (*response.EquipmentServiceResponse, error)
}
