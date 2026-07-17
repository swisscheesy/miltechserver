package pmcs_sbs_progress

import (
	"time"

	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/bootstrap"

	"github.com/google/uuid"
)

type Repository interface {
	EnsureInspection(user *bootstrap.User, inspection model.PmcsSbsInspections) (*model.PmcsSbsInspections, error)
	GetInspection(user *bootstrap.User, equipmentID string, pmcsID uuid.UUID) (*model.PmcsSbsInspections, []model.PmcsSbsFaults, error)
	ListInspections(user *bootstrap.User, equipmentID string, guideManual string, limit int, offset int) ([]InspectionSummary, error)
	DeleteInspection(user *bootstrap.User, equipmentID string, pmcsID uuid.UUID) error

	UpsertFault(user *bootstrap.User, inspection model.PmcsSbsInspections, fault model.PmcsSbsFaults) (*model.PmcsSbsFaults, error)
	DeleteFault(user *bootstrap.User, equipmentID string, key FaultKey) error
	DeleteFaults(user *bootstrap.User, equipmentID string, pmcsID uuid.UUID, keys []FaultKey) (int64, error)
}

type FaultKey struct {
	PmcsID    uuid.UUID
	SectionID string
	ItemIndex int32
}

type InspectionSummary struct {
	ID            uuid.UUID
	GuideManual   string
	PerformedDate time.Time
	FaultCount    int
	CreatedAt     time.Time
}
