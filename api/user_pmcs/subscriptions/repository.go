package subscriptions

import (
	"context"

	"github.com/google/uuid"

	"miltechserver/api/user_pmcs/shared"
)

type Repository interface {
	Install(context.Context, string, uuid.UUID, shared.Precondition) (*MutationResult, error)
	Unsubscribe(context.Context, string, uuid.UUID, shared.Precondition) (*MutationResult, error)
	GetInstalledRelease(context.Context, string, uuid.UUID, uuid.UUID) (*shared.InstalledChecklistRelease, error)
}

type MutationResult struct {
	Subscription shared.Subscription
	Installed    *shared.InstalledChecklistRelease
	Created      bool
	Idempotent   bool
}
