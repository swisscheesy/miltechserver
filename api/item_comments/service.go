package item_comments

import "miltechserver/bootstrap"

type Service interface {
	GetCommentsByNiin(niin string) ([]CommentResponse, error)
	CreateComment(user *bootstrap.User, niin string, text string, parentID *string) (*CommentResponse, error)
	UpdateComment(user *bootstrap.User, niin string, commentID string, text string) (*CommentResponse, error)
	DeleteComment(user *bootstrap.User, niin string, commentID string) (*CommentResponse, error)
	FlagComment(user *bootstrap.User, niin string, commentID string) error
}
