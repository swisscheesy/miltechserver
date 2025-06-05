package repository

import (
	"database/sql"
	"errors"
	"log/slog"
	. "miltechserver/.gen/miltech_ng/public/table"
	"miltechserver/api/auth"
	"miltechserver/bootstrap"
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
		DO_NOTHING().
		RETURNING(Users.AllColumns)

	err := stmt.Query(repo.Db, &userDto)
	if err != nil {
		return errors.New("error inserting user for registration: " + err.Error())
	}

	slog.Info("user for registered", "user_id", userDto.UID)
	return nil
}
