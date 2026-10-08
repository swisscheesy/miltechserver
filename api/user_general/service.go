package user_general

import (
	"context"

	"miltechserver/api/auth"
	"miltechserver/bootstrap"
)

type Service interface {
	UpsertUser(ctx context.Context, user *bootstrap.User, userDto auth.UserDto) error
	DeleteUser(ctx context.Context, uid string) error
	UpdateUserDisplayName(ctx context.Context, uid string, displayName string) error
}
