package item_comments

import (
	"context"
	"miltechserver/bootstrap"
)

type Service interface {
	GetCommentsByNiin(ctx context.Context, niin string) ([]CommentResponse, error)
	CreateComment(ctx context.Context, user *bootstrap.User, niin string, text string, parentID *string) (*CommentResponse, error)
	UpdateComment(ctx context.Context, user *bootstrap.User, niin string, commentID string, text string) (*CommentResponse, error)
	DeleteComment(ctx context.Context, user *bootstrap.User, niin string, commentID string) (*CommentResponse, error)
	FlagComment(ctx context.Context, user *bootstrap.User, niin string, commentID string) error
}
