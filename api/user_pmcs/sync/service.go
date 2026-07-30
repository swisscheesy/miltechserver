package sync

import (
	"context"

	"miltechserver/bootstrap"
)

type Service interface {
	GetDelta(
		ctx context.Context,
		user *bootstrap.User,
		after string,
		limit string,
	) (*AccountDelta, error)
}
