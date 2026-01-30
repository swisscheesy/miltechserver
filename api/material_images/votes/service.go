package votes

import "miltechserver/bootstrap"

type Service interface {
	Vote(user *bootstrap.User, imageID string, voteType string) error
	RemoveVote(user *bootstrap.User, imageID string) error
}
