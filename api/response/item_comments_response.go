package response

import "time"

type ItemCommentResponse struct {
	ID                string    `json:"id"`
	CommentNiin       string    `json:"comment_niin"`
	AuthorID          string    `json:"author_id"`
	AuthorDisplayName string    `json:"author_display_name"`
	Text              string    `json:"text"`
	ParentID          *string   `json:"parent_id"`
	CreatedAt         time.Time `json:"created_at"`
}
