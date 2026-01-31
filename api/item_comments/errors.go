package item_comments

import "errors"

var (
	ErrInvalidNiin     = errors.New("invalid NIIN")
	ErrInvalidText     = errors.New("invalid comment text")
	ErrCommentNotFound = errors.New("comment not found")
	ErrUnauthorized    = errors.New("unauthorized user")
	ErrForbidden       = errors.New("user not authorized")
	ErrInvalidParent   = errors.New("invalid parent comment")
)
