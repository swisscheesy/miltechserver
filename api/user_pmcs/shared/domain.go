package shared

import (
	"time"

	"github.com/google/uuid"
)

type RevisionInput struct {
	ID             uuid.UUID      `json:"id"`
	RevisionNumber *int32         `json:"revision_number,omitempty"`
	Name           string         `json:"name"`
	Description    string         `json:"description"`
	Models         []ModelInput   `json:"models"`
	Sections       []SectionInput `json:"sections"`
}

type ModelInput struct {
	DisplayText    string `json:"display_text"`
	NormalizedText string `json:"-"`
}

type SectionInput struct {
	ID       uuid.UUID    `json:"id"`
	Position int32        `json:"position"`
	Title    string       `json:"title"`
	Models   []ModelInput `json:"models"`
	Items    []ItemInput  `json:"items"`
}

type ItemInput struct {
	ID                        uuid.UUID            `json:"id"`
	Position                  int32                `json:"position"`
	Interval                  string               `json:"interval"`
	ItemToBeCheckedOrServiced string               `json:"item_to_be_checked_or_serviced"`
	PerformedBy               string               `json:"performed_by"`
	Notices                   []NoticeInput        `json:"notices"`
	ProcedureSteps            []ProcedureStepInput `json:"procedure_steps"`
}

type NoticeInput struct {
	ID         uuid.UUID `json:"id"`
	Position   int32     `json:"position"`
	Type       *string   `json:"type"`
	NoticeText string    `json:"notice_text"`
}

type ProcedureStepInput struct {
	ID           uuid.UUID `json:"id"`
	Position     int32     `json:"position"`
	StepText     string    `json:"step_text"`
	FaultFoundIf string    `json:"fault_found_if"`
}

type ModelValue struct {
	DisplayText    string `json:"display_text"`
	NormalizedText string `json:"normalized_text"`
}

type Section struct {
	ID       uuid.UUID    `json:"id"`
	Position int32        `json:"position"`
	Title    string       `json:"title"`
	Models   []ModelValue `json:"models"`
	Items    []Item       `json:"items"`
}

type Item struct {
	ID                        uuid.UUID            `json:"id"`
	Position                  int32                `json:"position"`
	Interval                  string               `json:"interval"`
	ItemToBeCheckedOrServiced string               `json:"item_to_be_checked_or_serviced"`
	PerformedBy               string               `json:"performed_by"`
	Notices                   []NoticeInput        `json:"notices"`
	ProcedureSteps            []ProcedureStepInput `json:"procedure_steps"`
}

type Revision struct {
	ID             uuid.UUID    `json:"id"`
	RevisionNumber *int32       `json:"revision_number,omitempty"`
	Name           string       `json:"name"`
	Description    string       `json:"description"`
	Models         []ModelValue `json:"models"`
	Sections       []Section    `json:"sections"`
	State          string       `json:"state"`
	CreatedAt      time.Time    `json:"created_at"`
	UpdatedAt      time.Time    `json:"updated_at"`
	PublishedAt    *time.Time   `json:"published_at,omitempty"`
}

type CommunitySourceSummary struct {
	Status                      string     `json:"status"`
	CurrentReleaseRevisionID    *uuid.UUID `json:"current_release_revision_id,omitempty"`
	LatestReleaseRevisionNumber int32      `json:"latest_release_revision_number"`
	FirstReleasedAt             time.Time  `json:"first_released_at"`
	UpdatedAt                   time.Time  `json:"updated_at"`
	RetiredAt                   *time.Time `json:"retired_at,omitempty"`
}

type ChecklistAggregate struct {
	ID                   uuid.UUID               `json:"id"`
	SyncVersion          int64                   `json:"sync_version"`
	AccountChangeVersion int64                   `json:"account_change_version"`
	CreatedAt            time.Time               `json:"created_at"`
	UpdatedAt            time.Time               `json:"updated_at"`
	DeletedAt            *time.Time              `json:"deleted_at,omitempty"`
	Draft                *Revision               `json:"draft,omitempty"`
	Publication          *Revision               `json:"publication,omitempty"`
	Community            *CommunitySourceSummary `json:"community,omitempty"`
}

type Subscription struct {
	ChecklistID          uuid.UUID  `json:"checklist_id"`
	InstalledRevisionID  *uuid.UUID `json:"installed_revision_id,omitempty"`
	SyncVersion          int64      `json:"sync_version"`
	AccountChangeVersion int64      `json:"account_change_version"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
	DeletedAt            *time.Time `json:"deleted_at,omitempty"`
}

type InstalledChecklistRelease struct {
	ChecklistID        uuid.UUID `json:"checklist_id"`
	SourceStatus       string    `json:"source_status"`
	CreatorDisplayName string    `json:"creator_display_name"`
	ReleasedAt         time.Time `json:"released_at"`
	Revision           Revision  `json:"revision"`
}

type CommunityBrowseFilter struct {
	After           *CommunityCursor
	Limit           int
	NormalizedModel string
}

type PublicCommunitySummary struct {
	ChecklistID        uuid.UUID    `json:"checklist_id"`
	RevisionID         uuid.UUID    `json:"revision_id"`
	RevisionNumber     int32        `json:"revision_number"`
	Name               string       `json:"name"`
	Description        string       `json:"description"`
	Models             []ModelValue `json:"models"`
	CreatorDisplayName string       `json:"creator_display_name"`
	ReleasedAt         time.Time    `json:"released_at"`
	UpdatedAt          time.Time    `json:"updated_at"`
}

type CommunityPage struct {
	NextCursor *string                  `json:"next_cursor,omitempty"`
	HasMore    bool                     `json:"has_more"`
	Items      []PublicCommunitySummary `json:"items"`
}

type PublicChecklistRelease struct {
	ChecklistID        uuid.UUID `json:"checklist_id"`
	CreatorDisplayName string    `json:"creator_display_name"`
	ReleasedAt         time.Time `json:"released_at"`
	Revision           Revision  `json:"revision"`
}
