package user_suggestions

import (
	"database/sql"

	"miltechserver/.gen/miltech_ng/public/model"

	"github.com/google/uuid"
)

type RepositoryImpl struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *RepositoryImpl {
	return &RepositoryImpl{db: db}
}

// Stub implementations — replaced with full implementations in Task 9.

func (r *RepositoryImpl) GetAllWithScores(voterID string) ([]SuggestionWithScore, error) {
	panic("not implemented")
}

func (r *RepositoryImpl) GetByID(id uuid.UUID) (*model.UserSuggestions, error) {
	panic("not implemented")
}

func (r *RepositoryImpl) Create(suggestion model.UserSuggestions) (*model.UserSuggestions, error) {
	panic("not implemented")
}

func (r *RepositoryImpl) Update(id uuid.UUID, title, description string) (*model.UserSuggestions, error) {
	panic("not implemented")
}

func (r *RepositoryImpl) Delete(id uuid.UUID) error {
	panic("not implemented")
}

func (r *RepositoryImpl) UpsertVote(suggestionID uuid.UUID, voterID string, direction int16) error {
	panic("not implemented")
}

func (r *RepositoryImpl) DeleteVote(suggestionID uuid.UUID, voterID string) error {
	panic("not implemented")
}
