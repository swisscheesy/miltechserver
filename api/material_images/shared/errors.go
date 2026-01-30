package shared

import "errors"

var (
	ErrImageNotFound  = errors.New("image not found")
	ErrUnauthorized   = errors.New("unauthorized")
	ErrForbidden      = errors.New("unauthorized: you can only delete your own images")
	ErrInvalidNIIN    = errors.New("NIIN must be exactly 9 characters")
	ErrRateLimited    = errors.New("rate limit exceeded")
	ErrInvalidVote    = errors.New("invalid vote type")
	ErrVoteNotFound   = errors.New("vote not found")
	ErrInvalidReason  = errors.New("invalid flag reason")
	ErrAlreadyFlagged = errors.New("you have already flagged this image")
)
