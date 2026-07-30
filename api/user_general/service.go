package user_general

import (
	"context"

	"miltechserver/api/auth"
	"miltechserver/bootstrap"
)

type Service interface {
	UpsertUser(user *bootstrap.User, userDto auth.UserDto) error
	DeleteUser(ctx context.Context, uid string) error
	UpdateUserDisplayName(uid string, displayName string) error
}
