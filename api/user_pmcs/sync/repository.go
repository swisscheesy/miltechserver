package sync

import (
	"context"

	"miltechserver/api/user_pmcs/shared"
)

const (
	ChangeKindChecklist    = "checklist"
	ChangeKindSubscription = "subscription"
)

type AccountDelta struct {
	FromCursor     int64           `json:"from_cursor"`
	ThroughCursor  int64           `json:"through_cursor"`
	AccountVersion int64           `json:"account_version"`
	HasMore        bool            `json:"has_more"`
	Changes        []AccountChange `json:"changes"`
}

type AccountChange struct {
	AccountChangeVersion int64                             `json:"account_change_version"`
	Kind                 string                            `json:"kind"`
	Checklist            *shared.ChecklistAggregate        `json:"checklist,omitempty"`
	Subscription         *shared.Subscription              `json:"subscription,omitempty"`
	Installed            *shared.InstalledChecklistRelease `json:"installed,omitempty"`
}

type Repository interface {
	GetDelta(
		ctx context.Context,
		userUID string,
		after int64,
		limit int,
		byteLimit int,
	) (*AccountDelta, error)
}
