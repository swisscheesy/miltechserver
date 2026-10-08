package shared

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

func TimePtr(value time.Time) *time.Time {
	return &value
}

func NullTimePtr(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	return &value.Time
}

func NullStringPtr(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	return &value.String
}

func NullBoolPtr(value sql.NullBool) *bool {
	if !value.Valid {
		return nil
	}
	return &value.Bool
}

func NullInt32Ptr(value sql.NullInt64) *int32 {
	if !value.Valid {
		return nil
	}
	intValue := int32(value.Int64)
	return &intValue
}

func ErrorsIsNoRows(err error) bool {
	return errors.Is(NormalizeNoRows(err), ErrNotFound)
}

func Placeholders(count int) string {
	values := make([]string, count)
	for i := range values {
		values[i] = fmt.Sprintf("$%d", i+1)
	}
	return strings.Join(values, ", ")
}
