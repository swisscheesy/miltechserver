package docs_equipment

import "errors"

var (
	ErrNotFound        = errors.New("no equipment details found")
	ErrEmptyParam      = errors.New("required parameter is empty")
	ErrInvalidPage     = errors.New("page must be greater than 0")
	ErrEmptyBlobPath   = errors.New("blob path cannot be empty")
	ErrInvalidBlobPath = errors.New("invalid blob path: must start with docs_equipment/images/")
	ErrInvalidFileType = errors.New("invalid file type: only image files are allowed")
	ErrImageNotFound   = errors.New("image not found")
	ErrBlobListFailed  = errors.New("failed to list blobs from Azure")
	ErrSASGenFailed    = errors.New("failed to generate download URL")
)
