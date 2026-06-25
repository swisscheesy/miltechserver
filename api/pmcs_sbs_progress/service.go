package pmcs_sbs_progress

import "miltechserver/bootstrap"

type Service interface {
	ListFaults(user *bootstrap.User, equipmentID string, guideManual string) (*FaultListResponse, error)
	UpsertFault(user *bootstrap.User, equipmentID string, req FaultRequest) (*FaultResponse, error)
	DeleteFault(user *bootstrap.User, equipmentID string, req DeleteFaultRequest) error
}
