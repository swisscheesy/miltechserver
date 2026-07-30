package shared

import "net/http"

type APIError struct {
	Status  int
	Code    string
	Message string
	Details map[string]any
	Cause   error
}

func (err *APIError) Error() string {
	return err.Code + ": " + err.Message
}

func (err *APIError) Unwrap() error {
	return err.Cause
}

func NewInvalidRequest(message string, details map[string]any) *APIError {
	return newAPIError(http.StatusBadRequest, "invalid_request", message, details)
}

func NewInvalidPrecondition(message string, details map[string]any) *APIError {
	return newAPIError(http.StatusBadRequest, "invalid_precondition", message, details)
}

func NewAuthenticationRequired(message string, details map[string]any) *APIError {
	return newAPIError(http.StatusUnauthorized, "authentication_required", message, details)
}

func NewForbidden(message string, details map[string]any) *APIError {
	return newAPIError(http.StatusForbidden, "forbidden", message, details)
}

func NewResourceNotFound(message string, details map[string]any) *APIError {
	return newAPIError(http.StatusNotFound, "resource_not_found", message, details)
}

func NewAccountNotInitialized(message string, details map[string]any) *APIError {
	return newAPIError(http.StatusConflict, "account_not_initialized", message, details)
}

func NewInvalidTransition(message string, details map[string]any) *APIError {
	return newAPIError(http.StatusConflict, "invalid_transition", message, details)
}

func NewStalePrecondition(message string, details map[string]any) *APIError {
	return newAPIError(http.StatusPreconditionFailed, "stale_precondition", message, details)
}

func NewContentTooLarge(message string, details map[string]any) *APIError {
	return newAPIError(http.StatusRequestEntityTooLarge, "content_too_large", message, details)
}

func NewValidationFailed(message string, details map[string]any) *APIError {
	return newAPIError(http.StatusUnprocessableEntity, "validation_failed", message, details)
}

func NewPreconditionRequired(message string, details map[string]any) *APIError {
	return newAPIError(http.StatusPreconditionRequired, "precondition_required", message, details)
}

func NewRateLimited(message string, details map[string]any) *APIError {
	return newAPIError(http.StatusTooManyRequests, "rate_limited", message, details)
}

func NewInternalError(message string, details map[string]any) *APIError {
	return newAPIError(http.StatusInternalServerError, "internal_error", message, details)
}

func newAPIError(status int, code, message string, details map[string]any) *APIError {
	return &APIError{Status: status, Code: code, Message: message, Details: details}
}
