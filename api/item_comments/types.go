package item_comments

import (
	"time"

	"miltechserver/.gen/miltech_ng/public/model"
)

// Request types

type CreateRequest struct {
	Text     string  `json:"text"`
	ParentID *string `json:"parent_id"`
}

type UpdateRequest struct {
	Text string `json:"text"`
}

// Response types

type CommentResponse struct {
	ID                string    `json:"id"`
	CommentNiin       string    `json:"comment_niin"`
	AuthorID          string    `json:"author_id"`
	AuthorDisplayName string    `json:"author_display_name"`
	Text              string    `json:"text"`
	ParentID          *string   `json:"parent_id"`
	CreatedAt         time.Time `json:"created_at"`
}

// Internal types

type CommentWithAuthor struct {
	model.ItemComments
	AuthorDisplayName *string
}
