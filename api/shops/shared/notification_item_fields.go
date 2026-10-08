package shared

import (
	"strings"
	"unicode/utf8"
)

// Clients display a NULL unit as "EA" and a NULL nickname as "". Treating the
// stored NULLs as those values when comparing keeps a re-save of a legacy item
// from being recorded as a change the user never made.
const (
	DefaultNotificationItemUnit      = "EA"
	MaxNotificationItemNicknameRunes = 50
	MaxNotificationItemUnitRunes     = 50 // unit_of_measure is varchar(50)
)

// ValidNotificationItemFields checks the optional nickname and unit sent with
// a direct notification item. Nil means the client did not send the field,
// which is always valid. Unit codes are deliberately not allow-listed so a
// newer client can add codes without a server release.
func ValidNotificationItemFields(nickname, unitOfMeasure *string) bool {
	if nickname != nil && utf8.RuneCountInString(*nickname) > MaxNotificationItemNicknameRunes {
		return false
	}
	if unitOfMeasure != nil {
		if strings.TrimSpace(*unitOfMeasure) == "" || utf8.RuneCountInString(*unitOfMeasure) > MaxNotificationItemUnitRunes {
			return false
		}
	}
	return true
}

func EffectiveNotificationItemNickname(nickname *string) string {
	if nickname == nil {
		return ""
	}
	return *nickname
}

func EffectiveNotificationItemUnit(unitOfMeasure *string) string {
	if unitOfMeasure == nil {
		return DefaultNotificationItemUnit
	}
	return *unitOfMeasure
}
