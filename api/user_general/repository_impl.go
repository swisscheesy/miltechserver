package user_general

import (
	"database/sql"
	"fmt"
	"log/slog"

	. "github.com/go-jet/jet/v2/postgres"

	"miltechserver/.gen/miltech_ng/public/table"
	"miltechserver/api/auth"
	"miltechserver/bootstrap"
)

type RepositoryImpl struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *RepositoryImpl {
	return &RepositoryImpl{db: db}
}

func (repo *RepositoryImpl) UpsertUser(user *bootstrap.User, userDto auth.UserDto) error {
	stmt := table.Users.INSERT(
		table.Users.UID,
		table.Users.Email,
		table.Users.Username,
		table.Users.CreatedAt,
		table.Users.IsEnabled,
		table.Users.LastLogin,
	).
		MODEL(userDto).
		ON_CONFLICT(table.Users.UID).
		DO_UPDATE(
			SET(
				table.Users.Email.SET(String(userDto.Email)),
				table.Users.Username.SET(String(userDto.Username)),
				table.Users.LastLogin.SET(TimestampT(userDto.LastLogin)),
				table.Users.IsEnabled.SET(Bool(userDto.IsEnabled)),
			).
				WHERE(table.Users.UID.EQ(String(userDto.UID))),
		).
		RETURNING(table.Users.AllColumns)

	_, err := stmt.Exec(repo.db)
	if err != nil {
		return fmt.Errorf("error upserting user: %w", err)
	}

	slog.Info("user UPDATED", "user_id", userDto.UID, "user_email", userDto.Email, "user_username", userDto.Username)
	return nil
}

func (repo *RepositoryImpl) DeleteUser(uid string) error {
	stmt := table.Users.DELETE().WHERE(table.Users.UID.EQ(String(uid)))

	result, err := stmt.Exec(repo.db)
	if err != nil {
		return fmt.Errorf("error deleting user: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error getting rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrUserNotFound
	}

	slog.Info("user DELETED", "user_id", uid)
	return nil
}

func (repo *RepositoryImpl) UpdateUserDisplayName(uid string, displayName string) error {
	stmt := table.Users.UPDATE(table.Users.Username).
		SET(String(displayName)).
		WHERE(table.Users.UID.EQ(String(uid)))

	result, err := stmt.Exec(repo.db)
	if err != nil {
		return fmt.Errorf("error updating user display name: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error getting rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrUserNotFound
	}

	slog.Info("user display name UPDATED", "user_id", uid, "display_name", displayName)
	return nil
}
