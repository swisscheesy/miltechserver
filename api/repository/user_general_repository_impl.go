package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	. "miltechserver/.gen/miltech_ng/public/table"
	"miltechserver/api/auth"
	"miltechserver/bootstrap"

	. "github.com/go-jet/jet/v2/postgres"
)

type UserGeneralRepositoryImpl struct {
	Db *sql.DB
}

func NewUserGeneralRepositoryImpl(db *sql.DB) *UserGeneralRepositoryImpl {
	return &UserGeneralRepositoryImpl{Db: db}
}

func (repo *UserGeneralRepositoryImpl) UpsertUser(user *bootstrap.User, userDto auth.UserDto) error {
	stmt := Users.INSERT(Users.UID, Users.Email, Users.Username, Users.CreatedAt, Users.IsEnabled, Users.LastLogin).
		MODEL(userDto).
		ON_CONFLICT(Users.UID).
		DO_UPDATE(
			SET(
				Users.Email.SET(String(userDto.Email)),
				Users.Username.SET(String(userDto.Username)),
				Users.LastLogin.SET(TimestampT(userDto.LastLogin)),
				Users.IsEnabled.SET(Bool(userDto.IsEnabled))).
				WHERE(Users.UID.EQ(String(userDto.UID)))).
		RETURNING(Users.AllColumns)

	_, err := stmt.Exec(repo.Db)
	if err != nil {
		return errors.New("error upserting user: " + err.Error())
	}

	slog.Info("user UPDATED", "user_id", userDto.UID, "user_email", userDto.Email, "user_username", userDto.Username)
	return nil
}

func (repo *UserGeneralRepositoryImpl) DeleteUser(uid string) error {
	stmt := Users.DELETE().WHERE(Users.UID.EQ(String(uid)))

	result, err := stmt.Exec(repo.Db)
	if err != nil {
		return fmt.Errorf("error deleting user: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error getting rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return errors.New("user not found")
	}

	slog.Info("user DELETED", "user_id", uid)
	return nil
}
