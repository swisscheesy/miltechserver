package subscriptions

import (
	"context"
	"miltechserver/api/user_pmcs/shared"
	"miltechserver/bootstrap"
)

type Service interface {
	Install(context.Context, *bootstrap.User, string, string, string) (*MutationResult, string, error)
	Unsubscribe(context.Context, *bootstrap.User, string, string) (*MutationResult, string, error)
	GetInstalledRelease(context.Context, *bootstrap.User, string, string) (*shared.InstalledChecklistRelease, string, error)
}
