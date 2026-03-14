package help

import "errors"

var (
	ErrHelpNotFound = errors.New("help not found")
	ErrInvalidCode  = errors.New("invalid help code")
)
