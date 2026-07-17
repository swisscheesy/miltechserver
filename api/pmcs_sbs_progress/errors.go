package pmcs_sbs_progress

import "errors"

var (
	ErrInvalidID          = errors.New("invalid id")
	ErrInvalidPmcsID      = errors.New("invalid pmcs id")
	ErrInvalidGuideManual = errors.New("invalid guide manual")
	ErrInvalidRequest     = errors.New("invalid request")
	ErrInvalidStatus      = errors.New("invalid fault status")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrNotFound           = errors.New("pmcs sbs equipment not found")
	ErrInspectionNotFound = errors.New("pmcs sbs inspection not found")
	ErrInspectionConflict = errors.New("pmcs sbs inspection conflict")
)
