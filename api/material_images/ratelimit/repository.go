package ratelimit

import "time"

type Repository interface {
	CheckLimit(userID string, niin string) (bool, *time.Time, error)
	UpdateLimit(userID string, niin string) error
	CleanupOld(olderThan time.Time) error
}
