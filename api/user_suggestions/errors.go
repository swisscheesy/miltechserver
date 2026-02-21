package user_suggestions

import "errors"

var (
	ErrUnauthorized       = errors.New("unauthorized user")
	ErrForbidden          = errors.New("not authorized to modify this suggestion")
	ErrSuggestionNotFound = errors.New("suggestion not found")
	ErrInvalidTitle       = errors.New("title must be between 1 and 200 characters")
	ErrInvalidDescription = errors.New("description must be between 1 and 2000 characters")
	ErrInvalidDirection   = errors.New("vote direction must be 1 or -1")
	ErrInvalidID          = errors.New("invalid suggestion ID")
)
