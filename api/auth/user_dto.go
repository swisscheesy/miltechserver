package auth

import "time"

type UserDto struct {
	UID       string    `json:"uid"`
	Email     string    `json:"email"`
	Username  string    `json:"username"`
	CreatedAt time.Time `json:"created_at"`
	IsEnabled bool      `json:"is_enabled"`
	LastLogin time.Time `json:"last_login"`
}
