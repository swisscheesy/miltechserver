package pmcs_sbs_progress

import "miltechserver/bootstrap"

type Service interface {
	EnsureInspection(user *bootstrap.User, equipmentID string, pmcsID string, req InspectionRequest) (*InspectionResponse, error)
	GetInspection(user *bootstrap.User, equipmentID string, pmcsID string) (*InspectionResponse, error)
	ListInspections(user *bootstrap.User, equipmentID string, req ListInspectionsRequest) (*InspectionListResponse, error)
	DeleteInspection(user *bootstrap.User, equipmentID string, pmcsID string) error

	UpsertFault(user *bootstrap.User, equipmentID string, pmcsID string, req FaultRequest) (*FaultResponse, error)
	DeleteFault(user *bootstrap.User, equipmentID string, pmcsID string, req DeleteFaultRequest) error
	DeleteFaults(user *bootstrap.User, equipmentID string, pmcsID string, req BulkDeleteFaultRequest) (*BulkDeleteFaultResponse, error)

	CreateComment(user *bootstrap.User, equipmentID string, pmcsID string, req CreateCommentRequest) (*CommentResponse, error)
	UpdateComment(user *bootstrap.User, equipmentID string, pmcsID string, commentID string, req UpdateCommentRequest) (*CommentResponse, error)
	DeleteComment(user *bootstrap.User, equipmentID string, pmcsID string, commentID string) (*CommentResponse, error)
}
