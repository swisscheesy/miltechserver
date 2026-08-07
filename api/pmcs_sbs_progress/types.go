package pmcs_sbs_progress

import (
	"time"

	"github.com/google/uuid"
)

type InspectionRequest struct {
	GuideManual   string    `json:"guide_manual"`
	PerformedDate time.Time `json:"performed_date"`
	Notes         *string   `json:"notes"`
}

type ListInspectionsRequest struct {
	GuideManual string `form:"guide_manual"`
	Limit       int    `form:"limit,default=1000" binding:"omitempty,min=1,max=1000"`
	Offset      int    `form:"offset,default=0" binding:"omitempty,min=0"`
}

type FaultRequest struct {
	GuideManual      string    `json:"guide_manual"`
	PerformedDate    time.Time `json:"performed_date"`
	SectionID        string    `json:"section_id"`
	ItemIndex        int32     `json:"item_index"`
	ItemNo           string    `json:"item_no"`
	Status           string    `json:"status"`
	FaultText        string    `json:"fault_text"`
	CorrectiveAction string    `json:"corrective_action"`
}

type DeleteFaultRequest struct {
	SectionID string `json:"section_id"`
	ItemIndex int32  `json:"item_index"`
}

type BulkDeleteFaultRequest struct {
	Faults []BulkDeleteFaultItemRequest `json:"faults"`
}

type BulkDeleteFaultItemRequest struct {
	SectionID string `json:"section_id"`
	ItemIndex int32  `json:"item_index"`
}

type FaultResponse struct {
	PmcsID           uuid.UUID `json:"pmcs_id"`
	SectionID        string    `json:"section_id"`
	ItemIndex        int32     `json:"item_index"`
	ItemNo           string    `json:"item_no"`
	Status           string    `json:"status"`
	FaultText        string    `json:"fault_text"`
	CorrectiveAction string    `json:"corrective_action"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type CreateCommentRequest struct {
	Text string `json:"text"`
}

type UpdateCommentRequest struct {
	Text string `json:"text"`
}

type CommentResponse struct {
	ID             uuid.UUID  `json:"id"`
	PmcsID         uuid.UUID  `json:"pmcs_id"`
	AuthorID       string     `json:"author_id"`
	AuthorUsername *string    `json:"author_username,omitempty"`
	Text           string     `json:"text"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      *time.Time `json:"updated_at,omitempty"`
}

type InspectionResponse struct {
	ID                  uuid.UUID         `json:"id"`
	EquipmentID         string            `json:"equipment_id"`
	SourceType          string            `json:"source_type"`
	GuideManual         string            `json:"guide_manual"`
	PerformedDate       time.Time         `json:"performed_date"`
	PerformedBy         *string           `json:"performed_by,omitempty"`
	PerformedByUsername *string           `json:"performed_by_username,omitempty"`
	Notes               *string           `json:"notes,omitempty"`
	CreatedAt           time.Time         `json:"created_at"`
	UpdatedAt           time.Time         `json:"updated_at"`
	Faults              []FaultResponse   `json:"faults"`
	Comments            []CommentResponse `json:"comments"`
}

type InspectionSummaryResponse struct {
	ID                  uuid.UUID `json:"id"`
	SourceType          string    `json:"source_type"`
	GuideManual         string    `json:"guide_manual"`
	PerformedDate       time.Time `json:"performed_date"`
	FaultCount          int       `json:"fault_count"`
	CommentCount        int       `json:"comment_count"`
	CreatedAt           time.Time `json:"created_at"`
	PerformedBy         *string   `json:"performed_by,omitempty"`
	PerformedByUsername *string   `json:"performed_by_username,omitempty"`
}

type InspectionListResponse struct {
	Inspections []InspectionSummaryResponse `json:"inspections"`
	Count       int                         `json:"count"`
}

type BulkDeleteFaultResponse struct {
	RequestedCount int `json:"requested_count"`
	DeletedCount   int `json:"deleted_count"`
}
