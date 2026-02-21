package user_suggestions

import "miltechserver/bootstrap"

type Service interface {
	GetAllSuggestions(currentUser *bootstrap.User) ([]SuggestionResponse, error)
	CreateSuggestion(user *bootstrap.User, title, description string) (*SuggestionResponse, error)
	UpdateSuggestion(user *bootstrap.User, suggestionID, title, description string) (*SuggestionResponse, error)
	DeleteSuggestion(user *bootstrap.User, suggestionID string) error
	Vote(user *bootstrap.User, suggestionID string, direction int16) error
	RemoveVote(user *bootstrap.User, suggestionID string) error
}
