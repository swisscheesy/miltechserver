package user_suggestions

import (
	"strings"
	"time"

	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/bootstrap"

	"github.com/google/uuid"
)

type ServiceImpl struct {
	repo Repository
}

func NewService(repo Repository) *ServiceImpl {
	return &ServiceImpl{repo: repo}
}

func (s *ServiceImpl) CreateSuggestion(user *bootstrap.User, title, description string) (*SuggestionResponse, error) {
	if user == nil {
		return nil, ErrUnauthorized
	}

	trimmedTitle := strings.TrimSpace(title)
	if len(trimmedTitle) == 0 || len(trimmedTitle) > 200 {
		return nil, ErrInvalidTitle
	}

	trimmedDesc := strings.TrimSpace(description)
	if len(trimmedDesc) == 0 || len(trimmedDesc) > 2000 {
		return nil, ErrInvalidDescription
	}

	suggestion := model.UserSuggestions{
		UserID:      user.UserID,
		Title:       trimmedTitle,
		Description: trimmedDesc,
		Status:      "Submitted",
	}

	created, err := s.repo.Create(suggestion)
	if err != nil {
		return nil, err
	}

	resp := mapSuggestionToResponse(created, user.Username, 0, nil)
	return &resp, nil
}

func (s *ServiceImpl) GetAllSuggestions(currentUser *bootstrap.User) ([]SuggestionResponse, error) {
	return nil, nil
}

func (s *ServiceImpl) UpdateSuggestion(user *bootstrap.User, suggestionID, title, description string) (*SuggestionResponse, error) {
	return nil, nil
}

func (s *ServiceImpl) DeleteSuggestion(user *bootstrap.User, suggestionID string) error {
	return nil
}

func (s *ServiceImpl) Vote(user *bootstrap.User, suggestionID string, direction int16) error {
	return nil
}

func (s *ServiceImpl) RemoveVote(user *bootstrap.User, suggestionID string) error {
	return nil
}

func mapSuggestionToResponse(s *model.UserSuggestions, username string, score int, myVote *int16) SuggestionResponse {
	resp := SuggestionResponse{
		ID:          s.ID.String(),
		UserID:      s.UserID,
		Username:    username,
		Title:       s.Title,
		Description: s.Description,
		Status:      s.Status,
		Score:       score,
		MyVote:      myVote,
		CreatedAt:   s.CreatedAt.Format(time.RFC3339),
	}

	if s.UpdatedAt != nil {
		formatted := s.UpdatedAt.Format(time.RFC3339)
		resp.UpdatedAt = &formatted
	}

	return resp
}

// parseID is a shared helper to avoid repeating uuid.Parse + ErrInvalidID mapping.
func parseID(suggestionID string) (uuid.UUID, error) {
	id, err := uuid.Parse(suggestionID)
	if err != nil {
		return uuid.UUID{}, ErrInvalidID
	}
	return id, nil
}
