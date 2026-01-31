package item_comments

import (
	"miltechserver/.gen/miltech_ng/public/model"

	"github.com/google/uuid"
)

type Repository interface {
	GetCommentsByNiin(niin string) ([]CommentWithAuthor, error)
	GetCommentByID(commentID uuid.UUID) (*model.ItemComments, error)
	CreateComment(comment model.ItemComments) (*model.ItemComments, error)
	UpdateCommentText(commentID uuid.UUID, text string) (*model.ItemComments, error)
	FlagComment(flag model.ItemCommentFlags) error
}
