package community

import (
	"context"

	"miltechserver/api/user_pmcs/shared"
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
	Browse(
		ctx context.Context,
		after string,
		limit string,
		model string,
	) (*shared.CommunityPage, error)
	BrowsePublic(
		ctx context.Context,
		after string,
		limit string,
		model string,
		sort string,
	) (*shared.PublicCommunityPage, error)
	BrowseAuthenticated(
		ctx context.Context,
		user *bootstrap.User,
		after string,
		limit string,
		model string,
		sort string,
	) (*shared.AuthenticatedCommunityPage, error)
	PutVote(
		ctx context.Context,
		user *bootstrap.User,
		checklistID string,
		direction int16,
	) (*shared.CommunityVoteMutation, error)
	DeleteVote(
		ctx context.Context,
		user *bootstrap.User,
		checklistID string,
	) (*shared.CommunityVoteMutation, error)
	GetCurrentRelease(
		ctx context.Context,
		checklistID string,
	) (*shared.PublicChecklistRelease, string, error)
}
