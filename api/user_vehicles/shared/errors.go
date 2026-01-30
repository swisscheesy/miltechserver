package shared

import "errors"

// User errors
var (
	ErrUserNotFound = errors.New("valid user not found")
)

// Vehicle errors
var (
	ErrVehicleNotFound = errors.New("vehicle not found")
)

// Notification errors
var (
	ErrNotificationNotFound = errors.New("notification not found")
)

// Item errors
var (
	ErrItemNotFound = errors.New("item not found")
)
