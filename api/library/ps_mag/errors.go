package ps_mag

import "errors"

var (
	ErrEmptyBlobPath   = errors.New("blob path cannot be empty")
	ErrInvalidBlobPath = errors.New("invalid blob path: must start with ps-mag/")
	ErrInvalidFileType = errors.New("invalid file type: only PDF files can be downloaded")
	ErrIssueNotFound   = errors.New("issue not found")
	ErrBlobListFailed  = errors.New("failed to list blobs from Azure")
	ErrSASGenFailed    = errors.New("failed to generate download URL")
	ErrInvalidPage     = errors.New("page must be greater than 0")
	ErrInvalidOrder    = errors.New("order must be 'asc' or 'desc'")
	ErrQueryTooShort   = errors.New("search query must be at least 3 characters")
)
