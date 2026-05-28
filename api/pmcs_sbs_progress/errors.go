package pmcs_sbs_progress

import "errors"

var (
	ErrInvalidID          = errors.New("invalid id")
	ErrInvalidRequest     = errors.New("invalid request")
	ErrInvalidBlobPath    = errors.New("invalid equipment manual blob path")
	ErrInvalidStatus      = errors.New("invalid fault status")
	ErrInvalidSyncRequest = errors.New("invalid sync request")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrNotFound           = errors.New("pmcs sbs equipment not found")
)
