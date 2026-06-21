package pmcs_sbs_progress

import "errors"

var (
	ErrInvalidID      = errors.New("invalid id")
	ErrInvalidRequest = errors.New("invalid request")
	ErrInvalidStatus  = errors.New("invalid fault status")
	ErrUnauthorized   = errors.New("unauthorized")
	ErrNotFound       = errors.New("pmcs sbs equipment not found")
)
