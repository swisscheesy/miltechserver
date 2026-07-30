package user_general

import (
	"context"
	"database/sql"

	"miltechserver/api/auth"
	"miltechserver/bootstrap"
)

type AccountCleaner interface {
	CleanupAccount(ctx context.Context, tx *sql.Tx, uid string) error
}

type Repository interface {
	UpsertUser(user *bootstrap.User, userDto auth.UserDto) error
	DeleteUser(ctx context.Context, uid string) error
	UpdateUserDisplayName(uid string, displayName string) error
}
