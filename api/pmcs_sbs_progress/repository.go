package pmcs_sbs_progress

import (
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/bootstrap"
)

type Repository interface {
	ListFaults(user *bootstrap.User, equipmentID string, guideManual string) ([]model.PmcsSbsFaults, error)
	UpsertFault(user *bootstrap.User, fault model.PmcsSbsFaults) (*model.PmcsSbsFaults, error)
	DeleteFault(user *bootstrap.User, key FaultKey) error
}

type FaultKey struct {
	EquipmentID string
	GuideManual string
	SectionID   string
	ItemIndex   int32
}
