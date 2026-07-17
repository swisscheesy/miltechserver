package pmcs_sbs_progress

import "miltechserver/bootstrap"

// Temporary compatibility type - to be replaced in Task 3
type FaultListResponse struct {
	Faults []FaultResponse `json:"faults"`
	Count  int             `json:"count"`
}

type Service interface {
	ListFaults(user *bootstrap.User, equipmentID string, guideManual string) (*FaultListResponse, error)
	UpsertFault(user *bootstrap.User, equipmentID string, req FaultRequest) (*FaultResponse, error)
	DeleteFault(user *bootstrap.User, equipmentID string, req DeleteFaultRequest) error
	DeleteFaults(user *bootstrap.User, equipmentID string, req BulkDeleteFaultRequest) (*BulkDeleteFaultResponse, error)
}
