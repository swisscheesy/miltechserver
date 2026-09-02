package votes

import (
	"context"
	"miltechserver/bootstrap"
)

type Service interface {
	Vote(ctx context.Context, user *bootstrap.User, imageID string, voteType string) error
	RemoveVote(ctx context.Context, user *bootstrap.User, imageID string) error
}
