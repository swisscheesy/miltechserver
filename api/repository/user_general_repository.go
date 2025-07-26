package repository

import (
	"miltechserver/api/auth"
	"miltechserver/bootstrap"
)

type UserGeneralRepository interface {
	UpsertUser(user *bootstrap.User, userDto auth.UserDto) error
	DeleteUser(uid string) error
	UpdateUserDisplayName(uid string, displayName string) error
}
