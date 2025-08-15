package response

import (
	"time"
)

type MaterialImageResponse struct {
	ID               string    `json:"id"`
	NIIN            string    `json:"niin"`
	UserID          string    `json:"user_id"`
	Username        string    `json:"username"`
	ImageData       []byte    `json:"image_data"`
	OriginalFilename string    `json:"original_filename"`
	FileSizeBytes   int64     `json:"file_size_bytes"`
	MimeType        string    `json:"mime_type"`
	UploadDate      time.Time `json:"upload_date"`
	UpvoteCount     int       `json:"upvote_count"`
	DownvoteCount   int       `json:"downvote_count"`
	NetVotes        int       `json:"net_votes"`
	IsFlagged       bool      `json:"is_flagged"`
	UserVote        *string   `json:"user_vote,omitempty"`
	CanDelete       bool      `json:"can_delete"`
}

type PaginatedImagesResponse struct {
	Images      []MaterialImageResponse `json:"images"`
	TotalCount  int64                  `json:"total_count"`
	Page        int                    `json:"page"`
	PageSize    int                    `json:"page_size"`
	TotalPages  int                    `json:"total_pages"`
}

type ImageUploadResponse struct {
	Success bool                  `json:"success"`
	Message string                `json:"message"`
	Image   *MaterialImageResponse `json:"image,omitempty"`
}

type ImageVoteResponse struct {
	Success       bool   `json:"success"`
	Message       string `json:"message"`
	UpvoteCount   int    `json:"upvote_count"`
	DownvoteCount int    `json:"downvote_count"`
	NetVotes      int    `json:"net_votes"`
}

type ImageFlagResponse struct {
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	FlagCount int    `json:"flag_count"`
	IsFlagged bool   `json:"is_flagged"`
}