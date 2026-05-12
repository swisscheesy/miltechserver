package sb_700_20

import "errors"

var (
	ErrNotFound    = errors.New("no records found")
	ErrEmptyParam  = errors.New("required parameter is empty")
	ErrInvalidPage = errors.New("page number must be greater than 0")
)
