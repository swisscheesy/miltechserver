package library

import "errors"

var (
	ErrEmptyVehicleName = errors.New("vehicle name cannot be empty")
	ErrEmptyBlobPath    = errors.New("blob path cannot be empty")
	ErrInvalidBlobPath  = errors.New("invalid blob path: must start with pmcs/ or bii/")
	ErrInvalidFileType  = errors.New("invalid file type: only PDF files can be downloaded")
	ErrDocumentNotFound = errors.New("document not found")
	ErrBlobListFailed   = errors.New("failed to list blobs")
	ErrSASGenFailed     = errors.New("failed to generate download URL")
)
