package service

import (
	"miltechserver/api/response"
	"miltechserver/bootstrap"
)

type ItemCommentsService interface {
	GetCommentsByNiin(niin string) ([]response.ItemCommentResponse, error)
	CreateComment(user *bootstrap.User, niin string, text string, parentID *string) (*response.ItemCommentResponse, error)
	UpdateComment(user *bootstrap.User, niin string, commentID string, text string) (*response.ItemCommentResponse, error)
	DeleteComment(user *bootstrap.User, niin string, commentID string) (*response.ItemCommentResponse, error)
	FlagComment(user *bootstrap.User, niin string, commentID string) error
}
