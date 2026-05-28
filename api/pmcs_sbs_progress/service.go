package pmcs_sbs_progress

import "miltechserver/bootstrap"

type Service interface {
	ListEquipment(user *bootstrap.User) (*EquipmentListResponse, error)
	GetEquipment(user *bootstrap.User, equipmentID string) (*EquipmentAggregateResponse, error)
	UpsertEquipment(user *bootstrap.User, equipmentID string, req EquipmentRequest) (*EquipmentResponse, error)
	DeleteEquipment(user *bootstrap.User, equipmentID string) error
	UpsertCompletion(user *bootstrap.User, equipmentID string, req CompletionRequest) (*CompletionResponse, error)
	DeleteCompletion(user *bootstrap.User, equipmentID string, req DeleteCompletionRequest) error
	UpsertFault(user *bootstrap.User, equipmentID string, req FaultRequest) (*FaultResponse, error)
	DeleteFault(user *bootstrap.User, equipmentID string, req DeleteFaultRequest) error
	Sync(user *bootstrap.User, req SyncRequest) (*SyncResponse, error)
}
