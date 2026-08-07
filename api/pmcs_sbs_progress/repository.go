package pmcs_sbs_progress

import (
	"time"

	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/bootstrap"

	"github.com/google/uuid"
)

type Repository interface {
	EnsureInspection(user *bootstrap.User, inspection model.PmcsSbsInspections) (*model.PmcsSbsInspections, error)
	GetInspection(user *bootstrap.User, equipmentID string, pmcsID uuid.UUID) (*InspectionDetail, []model.PmcsSbsFaults, []CommentWithAuthor, error)
	ListInspections(user *bootstrap.User, equipmentID string, guideManual string, limit int, offset int) ([]InspectionSummary, error)
	DeleteInspection(user *bootstrap.User, equipmentID string, pmcsID uuid.UUID) error
	LookupUsername(userID string) (*string, error)

	UpsertFault(user *bootstrap.User, inspection model.PmcsSbsInspections, fault model.PmcsSbsFaults) (*model.PmcsSbsFaults, error)
	DeleteFault(user *bootstrap.User, equipmentID string, key FaultKey) error
	DeleteFaults(user *bootstrap.User, equipmentID string, pmcsID uuid.UUID, keys []FaultKey) (int64, error)

	CreateComment(user *bootstrap.User, equipmentID string, pmcsID uuid.UUID, text string) (*CommentWithAuthor, error)
	GetComment(commentID uuid.UUID) (*CommentWithAuthor, error)
	UpdateComment(commentID uuid.UUID, text string) (*CommentWithAuthor, error)
}

type InspectionDetail struct {
	model.PmcsSbsInspections
	PerformedByUsername *string
}

type FaultKey struct {
	PmcsID    uuid.UUID
	SectionID string
	ItemIndex int32
}

type InspectionSummary struct {
	ID                   uuid.UUID
	SourceType           string
	GuideManual          *string
	CustomChecklistID    *uuid.UUID
	CustomRevisionID     *uuid.UUID
	CustomRevisionNumber *int32
	CustomChecklistName  *string
	PerformedDate        time.Time
	FaultCount           int
	CommentCount         int
	CreatedAt            time.Time
	PerformedBy          *string
	PerformedByUsername  *string
}

type CommentWithAuthor struct {
	model.PmcsSbsInspectionComments
	AuthorUsername *string
}
