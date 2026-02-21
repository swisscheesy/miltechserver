package user_suggestions

import (
	"miltechserver/.gen/miltech_ng/public/model"

	"github.com/google/uuid"
)

type Repository interface {
	GetAllWithScores(voterID string) ([]SuggestionWithScore, error)
	GetByID(id uuid.UUID) (*model.UserSuggestions, error)
	Create(suggestion model.UserSuggestions) (*model.UserSuggestions, error)
	Update(id uuid.UUID, title, description string) (*model.UserSuggestions, error)
	Delete(id uuid.UUID) error
	UpsertVote(suggestionID uuid.UUID, voterID string, direction int16) error
	DeleteVote(suggestionID uuid.UUID, voterID string) error
}
