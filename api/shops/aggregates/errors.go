package aggregates

import "errors"

var (
	ErrUnauthorized         = errors.New("unauthorized")
	ErrAccessDenied         = errors.New("access denied")
	ErrInvalidLimit         = errors.New("invalid limit")
	ErrInvalidInclude       = errors.New("invalid include")
	ErrAggregateUnavailable = errors.New("failed to retrieve shops aggregate")
)
