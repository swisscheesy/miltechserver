package pmcs_sbs_progress

import (
	"context"
	"miltechserver/bootstrap"
)

type Service interface {
	EnsureInspection(ctx context.Context, user *bootstrap.User, equipmentID string, pmcsID string, req InspectionRequest) (*InspectionResponse, error)
	GetInspection(ctx context.Context, user *bootstrap.User, equipmentID string, pmcsID string) (*InspectionResponse, error)
	ListInspections(ctx context.Context, user *bootstrap.User, equipmentID string, req ListInspectionsRequest) (*InspectionListResponse, error)
	DeleteInspection(ctx context.Context, user *bootstrap.User, equipmentID string, pmcsID string) error

	UpsertFault(ctx context.Context, user *bootstrap.User, equipmentID string, pmcsID string, req FaultRequest) (*FaultResponse, error)
	DeleteFault(ctx context.Context, user *bootstrap.User, equipmentID string, pmcsID string, req DeleteFaultRequest) error
	DeleteFaults(ctx context.Context, user *bootstrap.User, equipmentID string, pmcsID string, req BulkDeleteFaultRequest) (*BulkDeleteFaultResponse, error)

	CreateComment(ctx context.Context, user *bootstrap.User, equipmentID string, pmcsID string, req CreateCommentRequest) (*CommentResponse, error)
	UpdateComment(ctx context.Context, user *bootstrap.User, equipmentID string, pmcsID string, commentID string, req UpdateCommentRequest) (*CommentResponse, error)
	DeleteComment(ctx context.Context, user *bootstrap.User, equipmentID string, pmcsID string, commentID string) (*CommentResponse, error)
}
