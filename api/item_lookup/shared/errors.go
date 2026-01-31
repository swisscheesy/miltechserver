package shared

import "errors"

var (
	ErrNotFound    = errors.New("no items found")
	ErrEmptyParam  = errors.New("required parameter is empty")
	ErrInvalidPage = errors.New("page number must be greater than 0")
)
