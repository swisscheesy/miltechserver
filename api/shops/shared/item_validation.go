package shared

import (
	"errors"
	"strings"
)

// ValidationError distinguishes rejected input from persistence failures.
type ValidationError struct{}

func (*ValidationError) Error() string { return "invalid request" }

func IsValidationError(err error) bool {
	var validation *ValidationError
	return errors.As(err, &validation)
}

func ValidateItemFields(niin, nomenclature string, quantity int32) error {
	if strings.TrimSpace(niin) == "" || strings.TrimSpace(nomenclature) == "" || quantity <= 0 {
		return &ValidationError{}
	}
	return nil
}

func ValidateNotificationType(value string) error {
	switch value {
	case "M1", "PM", "MW":
		return nil
	}
	return &ValidationError{}
}

func ValidateNotificationFields(title, notificationType string) error {
	if strings.TrimSpace(title) == "" {
		return &ValidationError{}
	}
	return ValidateNotificationType(notificationType)
}

func ValidateItemEnrichment(nickname, unit *string) error {
	if !ValidNotificationItemFields(nickname, unit) {
		return &ValidationError{}
	}
	return nil
}
