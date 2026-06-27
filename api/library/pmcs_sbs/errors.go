package pmcs_sbs

import "errors"

var (
	ErrEmptyFolderName  = errors.New("folder name cannot be empty")
	ErrEmptyBlobPath    = errors.New("blob path cannot be empty")
	ErrEmptyImageName   = errors.New("image name cannot be empty")
	ErrInvalidBlobPath  = errors.New("invalid blob path: must start with pmcs_sbs/")
	ErrInvalidImageName = errors.New(
		"invalid image name: must be an extensionless PNG basename",
	)
	ErrInvalidFileType = errors.New("invalid file type: only JSON files are accessible")
	ErrFileNotFound    = errors.New("file not found")
	ErrBlobListFailed  = errors.New("failed to list blobs")
	ErrBlobReadFailed  = errors.New("failed to read blob content")
	ErrInvalidJSON     = errors.New("blob content is not valid JSON")
	ErrBlobTooLarge    = errors.New("blob content exceeds maximum allowed size")
)
