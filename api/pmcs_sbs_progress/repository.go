package pmcs_sbs_progress

import (
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/bootstrap"
)

type Repository interface {
	ListEquipment(user *bootstrap.User) ([]model.PmcsSbsEquipment, error)
	GetEquipmentAggregate(user *bootstrap.User, equipmentID string) (*EquipmentAggregate, error)
	UpsertEquipment(user *bootstrap.User, equipment model.PmcsSbsEquipment) (*model.PmcsSbsEquipment, error)
	DeleteEquipment(user *bootstrap.User, equipmentID string) error
	UpsertCompletion(user *bootstrap.User, completion model.PmcsSbsCompletions) (*model.PmcsSbsCompletions, error)
	BatchCompletions(user *bootstrap.User, equipmentID string, upserts []model.PmcsSbsCompletions, deletes []CompletionKey) (*BatchCompletionsResult, error)
	DeleteCompletion(user *bootstrap.User, equipmentID string, sectionID string, itemIndex int32, stepID string) error
	UpsertFault(user *bootstrap.User, fault model.PmcsSbsFaults) (*model.PmcsSbsFaults, error)
	DeleteFault(user *bootstrap.User, equipmentID string, sectionID string, itemIndex int32) error
	Sync(user *bootstrap.User, changeSet SyncChangeSet) (*SyncResult, error)
}

type EquipmentAggregate struct {
	Equipment   model.PmcsSbsEquipment
	Completions []model.PmcsSbsCompletions
	Faults      []model.PmcsSbsFaults
}

type SyncChangeSet struct {
	UpsertEquipment    []model.PmcsSbsEquipment
	DeleteEquipmentIDs []string
	UpsertCompletions  []model.PmcsSbsCompletions
	DeleteCompletions  []CompletionKey
	UpsertFaults       []model.PmcsSbsFaults
	DeleteFaults       []FaultKey
}

type CompletionKey struct {
	EquipmentID string
	SectionID   string
	ItemIndex   int32
	StepID      string
}

type BatchCompletionsResult struct {
	UpsertedCount int64
	DeletedCount  int64
}

type FaultKey struct {
	EquipmentID string
	SectionID   string
	ItemIndex   int32
}

type SyncResult struct {
	Equipment           []EquipmentAggregate
	DeletedEquipmentIDs []string
}
