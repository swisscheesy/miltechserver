package service

import (
	"miltechserver/api/auth"
	"miltechserver/bootstrap"
)

type UserGeneralService interface {
	UpsertUser(user *bootstrap.User, userDto auth.UserDto) error
	DeleteUser(uid string) error
	UpdateUserDisplayName(uid string, displayName string) error
}
