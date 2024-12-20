package repository

import (
	"database/sql"
	"errors"
	. "github.com/go-jet/jet/v2/postgres"
	"log/slog"
	"miltechserver/.gen/miltech_ng/public/model"
	. "miltechserver/.gen/miltech_ng/public/table"
	"miltechserver/bootstrap"
)

type UserSavesRepositoryImpl struct {
	Db *sql.DB
}

func NewUserSavesRepositoryImpl(db *sql.DB) *UserSavesRepositoryImpl {
	return &UserSavesRepositoryImpl{Db: db}
}

func (repo *UserSavesRepositoryImpl) GetQuickSaveItemsByUserId(user *bootstrap.User) ([]model.UserItemsQuick, error) {
	var items []model.UserItemsQuick

	if user != nil {
		stmt := SELECT(
			UserItemsQuick.AllColumns).FROM(UserItemsQuick).WHERE(UserItemsQuick.UserID.EQ(String(user.UserID)))

		err := stmt.Query(repo.Db, &items)
		if err != nil {
			return nil, errors.New("user saves not found")
		} else {
			slog.Info("quick saves retrieved for user", "user_id", user.UserID)
			return items, nil
		}
	} else {
		return nil, errors.New("valid user not found")
	}

}
