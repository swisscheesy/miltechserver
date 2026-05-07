// api/tmde/errors.go
package tmde

import "errors"

var (
	ErrNotFound    = errors.New("no TMDE requirements found")
	ErrEmptyParam  = errors.New("required parameter is empty")
	ErrInvalidPage = errors.New("page number must be greater than 0")
)
