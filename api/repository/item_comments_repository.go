package repository

import (
	"miltechserver/.gen/miltech_ng/public/model"

	"github.com/google/uuid"
)

type ItemCommentWithAuthor struct {
	model.ItemComments
	AuthorDisplayName *string `sql:"author_display_name"`
}

type ItemCommentsRepository interface {
	GetCommentsByNiin(niin string) ([]ItemCommentWithAuthor, error)
	GetCommentByID(commentID uuid.UUID) (*model.ItemComments, error)
	CreateComment(comment model.ItemComments) (*model.ItemComments, error)
	UpdateCommentText(commentID uuid.UUID, text string) (*model.ItemComments, error)
	FlagComment(flag model.ItemCommentFlags) error
}
