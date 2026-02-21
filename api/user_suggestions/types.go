package user_suggestions

import (
	"time"

	"github.com/google/uuid"
)

// --- Requests ---

type CreateSuggestionRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

type UpdateSuggestionRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

type VoteRequest struct {
	Direction int16 `json:"direction"`
}

// --- Responses ---

type SuggestionResponse struct {
	ID          string  `json:"id"`
	UserID      string  `json:"user_id"`
	Username    string  `json:"username"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Status      string  `json:"status"`
	Score       int     `json:"score"`
	MyVote      *int16  `json:"my_vote,omitempty"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   *string `json:"updated_at,omitempty"`
}

// --- Internal types ---

type SuggestionWithScore struct {
	ID          uuid.UUID
	UserID      string
	Title       string
	Description string
	Status      string
	CreatedAt   time.Time
	UpdatedAt   *time.Time
	Username    *string
	Score       int
	MyVote      *int16
}
