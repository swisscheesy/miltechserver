package eic

import "errors"

var (
	ErrNotFound    = errors.New("no EIC items found")
	ErrEmptyParam  = errors.New("required parameter is empty")
	ErrInvalidPage = errors.New("page number must be greater than 0")
)
