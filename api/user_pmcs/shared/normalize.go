package shared

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

func NormalizeModel(value string) (string, error) {
	if !utf8.ValidString(value) {
		return "", NewValidationFailed("model text must contain valid UTF-8", nil)
	}

	var normalized strings.Builder
	normalized.Grow(len(value))
	hasText := false
	pendingSpace := false
	for _, character := range value {
		if unicode.IsSpace(character) {
			pendingSpace = hasText
			continue
		}
		if pendingSpace {
			normalized.WriteByte(' ')
			pendingSpace = false
		}
		normalized.WriteRune(unicode.ToLower(character))
		hasText = true
	}
	return normalized.String(), nil
}
