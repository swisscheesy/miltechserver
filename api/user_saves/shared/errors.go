package shared

import "errors"

var (
	ErrUserNotFound = errors.New("valid user not found")
)

var (
	ErrItemNotFound      = errors.New("item not found")
	ErrItemAlreadyExists = errors.New("item already exists")
	ErrInvalidItemID     = errors.New("invalid item ID")
)

var (
	ErrCategoryNotFound      = errors.New("category not found")
	ErrCategoryAlreadyExists = errors.New("category already exists")
	ErrCategoryNotEmpty      = errors.New("category is not empty")
)

var (
	ErrImageNotFound     = errors.New("image not found")
	ErrImageUploadFailed = errors.New("failed to upload image")
	ErrImageDeleteFailed = errors.New("failed to delete image")
	ErrInvalidTableType  = errors.New("invalid table type")
	ErrImageTooLarge     = errors.New("image exceeds maximum size")
)

var (
	ErrBulkOperationPartialFailure = errors.New("bulk operation partially failed")
)
