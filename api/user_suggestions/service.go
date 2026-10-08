package user_suggestions

import (
	"context"
	"miltechserver/bootstrap"
)

type Service interface {
	GetAllSuggestions(ctx context.Context, currentUser *bootstrap.User) ([]SuggestionResponse, error)
	CreateSuggestion(ctx context.Context, user *bootstrap.User, title, description string) (*SuggestionResponse, error)
	UpdateSuggestion(ctx context.Context, user *bootstrap.User, suggestionID, title, description string) (*SuggestionResponse, error)
	DeleteSuggestion(ctx context.Context, user *bootstrap.User, suggestionID string) error
	Vote(ctx context.Context, user *bootstrap.User, suggestionID string, direction int16) error
	RemoveVote(ctx context.Context, user *bootstrap.User, suggestionID string) error
}
