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
	voterID := ""
	if currentUser != nil {
		voterID = currentUser.UserID
	}

	suggestions, err := s.repo.GetAllWithScores(voterID)
	if err != nil {
		return nil, err
	}

	results := make([]SuggestionResponse, 0, len(suggestions))
	for _, sug := range suggestions {
		username := "Unknown"
		if sug.Username != nil {
			username = *sug.Username
		}

		resp := SuggestionResponse{
			ID:          sug.ID.String(),
			UserID:      sug.UserID,
			Username:    username,
			Title:       sug.Title,
			Description: sug.Description,
			Status:      sug.Status,
			Score:       sug.Score,
			CreatedAt:   sug.CreatedAt.Format(time.RFC3339),
		}

		if sug.UpdatedAt != nil {
			formatted := sug.UpdatedAt.Format(time.RFC3339)
			resp.UpdatedAt = &formatted
		}

		// Only include MyVote when user is authenticated
		if currentUser != nil {
			resp.MyVote = sug.MyVote
		}

		results = append(results, resp)
	}

	return results, nil
}

func (s *ServiceImpl) UpdateSuggestion(user *bootstrap.User, suggestionID, title, description string) (*SuggestionResponse, error) {
	if user == nil {
		return nil, ErrUnauthorized
	}

	id, err := parseID(suggestionID)
	if err != nil {
		return nil, err
	}

	existing, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, ErrSuggestionNotFound
	}
	if existing.UserID != user.UserID {
		return nil, ErrForbidden
	}

	trimmedTitle := strings.TrimSpace(title)
	if len(trimmedTitle) == 0 || len(trimmedTitle) > 200 {
		return nil, ErrInvalidTitle
	}

	trimmedDesc := strings.TrimSpace(description)
	if len(trimmedDesc) == 0 || len(trimmedDesc) > 2000 {
		return nil, ErrInvalidDescription
	}

	updated, err := s.repo.Update(id, trimmedTitle, trimmedDesc)
	if err != nil {
		return nil, err
	}

	resp := mapSuggestionToResponse(updated, user.Username, 0, nil)
	return &resp, nil
}

func (s *ServiceImpl) DeleteSuggestion(user *bootstrap.User, suggestionID string) error {
	if user == nil {
		return ErrUnauthorized
	}

	id, err := parseID(suggestionID)
	if err != nil {
		return err
	}

	existing, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}
	if existing == nil {
		return ErrSuggestionNotFound
	}
	if existing.UserID != user.UserID {
		return ErrForbidden
	}

	return s.repo.Delete(id)
}

func (s *ServiceImpl) Vote(user *bootstrap.User, suggestionID string, direction int16) error {
	if user == nil {
		return ErrUnauthorized
	}

	id, err := parseID(suggestionID)
	if err != nil {
		return err
	}

	if direction != 1 && direction != -1 {
		return ErrInvalidDirection
	}

	existing, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}
	if existing == nil {
		return ErrSuggestionNotFound
	}

	// Check if the user already has a vote on this suggestion.
	// If they do, remove it (toggle off) to prevent ±2 score jumps
	// when switching vote directions.
	existingVote, err := s.repo.GetVote(id, user.UserID)
	if err != nil {
		return err
	}
	if existingVote != nil {
		return s.repo.DeleteVote(id, user.UserID)
	}

	return s.repo.UpsertVote(id, user.UserID, direction)
}

func (s *ServiceImpl) RemoveVote(user *bootstrap.User, suggestionID string) error {
	if user == nil {
		return ErrUnauthorized
	}

	id, err := parseID(suggestionID)
	if err != nil {
		return err
	}

	return s.repo.DeleteVote(id, user.UserID)
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
