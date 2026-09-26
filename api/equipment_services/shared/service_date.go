package shared

import (
	"errors"
	"time"
)

// ParseServiceDate accepts a Gregorian date in years 0001 through 9999.
// UTC carries calendar components only; callers must not convert it to a viewer zone.
func ParseServiceDate(value string) (time.Time, error) {
	date, err := time.Parse("2006-01-02", value)
	if err != nil || date.Year() < 1 || date.Format("2006-01-02") != value {
		return time.Time{}, errors.New("invalid service date")
	}
	return date, nil
}
