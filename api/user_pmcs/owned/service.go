package owned

import (
	"context"

	"miltechserver/api/user_pmcs/shared"
	"miltechserver/bootstrap"
)

type Service interface {
	Get(
		ctx context.Context,
		user *bootstrap.User,
		checklistID string,
	) (*shared.ChecklistAggregate, string, error)
	Create(
		ctx context.Context,
		user *bootstrap.User,
		checklistID string,
		draft shared.RevisionInput,
		ifNoneMatch string,
	) (*MutationResult, string, error)
	PutDraft(
		ctx context.Context,
		user *bootstrap.User,
		checklistID string,
		revisionID string,
		draft shared.RevisionInput,
		ifMatch string,
	) (*MutationResult, string, error)
	DeleteDraft(
		ctx context.Context,
		user *bootstrap.User,
		checklistID string,
		revisionID string,
		ifMatch string,
	) (*MutationResult, string, error)
	DeleteChecklist(
		ctx context.Context,
		user *bootstrap.User,
		checklistID string,
		ifMatch string,
	) (*MutationResult, string, error)
	Publish(
		ctx context.Context,
		user *bootstrap.User,
		checklistID string,
		revisionID string,
		revision shared.RevisionInput,
		ifMatch string,
	) (*MutationResult, string, error)
	GetRevision(
		ctx context.Context,
		user *bootstrap.User,
		checklistID string,
		revisionID string,
	) (*HistoricalRevision, string, error)
}
