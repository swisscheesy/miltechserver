package community

import (
	"context"

	"miltechserver/bootstrap"
)

type Service interface {
	Release(
		ctx context.Context,
		user *bootstrap.User,
		checklistID string,
		revisionID string,
		ifMatch string,
	) (*ReleaseMutationResult, string, error)
	Retire(
		ctx context.Context,
		user *bootstrap.User,
		checklistID string,
		ifMatch string,
	) (*ReleaseMutationResult, string, error)
}
