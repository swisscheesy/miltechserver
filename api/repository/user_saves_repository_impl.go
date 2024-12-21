package repository

import (
	"database/sql"
	"errors"
	"fmt"
	. "github.com/go-jet/jet/v2/postgres"
	"log/slog"
	"miltechserver/.gen/miltech_ng/public/model"
	. "miltechserver/.gen/miltech_ng/public/table"
	"miltechserver/bootstrap"
	"time"
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

func (repo *UserSavesRepositoryImpl) UpsertQuickSaveItemByUser(user *bootstrap.User, quick model.UserItemsQuick) error {
	stmt := UserItemsQuick.INSERT(UserItemsQuick.UserID, UserItemsQuick.Niin, UserItemsQuick.ItemName,
		UserItemsQuick.ImageLocation, UserItemsQuick.ItemComment, UserItemsQuick.SaveTime).
		MODEL(quick).
		ON_CONFLICT(UserItemsQuick.UserID, UserItemsQuick.Niin).
		DO_UPDATE(
			SET(
				UserItemsQuick.Niin.SET(String(quick.Niin)),
				UserItemsQuick.ItemName.SET(String(*quick.ItemName)),
				UserItemsQuick.ImageLocation.SET(String(*quick.ImageLocation)),
				UserItemsQuick.ItemComment.SET(String(*quick.ItemComment)),
				UserItemsQuick.SaveTime.SET(Timestamp(time.Now().Year(),
					time.Now().Month(), time.Now().Day(), time.Now().Hour(),
					time.Now().Minute(), 0))). // This is super ugly
				WHERE(UserItemsQuick.UserID.EQ(String(user.UserID)).
					AND(
						UserItemsQuick.Niin.EQ(String(quick.Niin))))).
		RETURNING(UserItemsQuick.AllColumns)

	err := stmt.Query(repo.Db, &quick)

	if err != nil {
		return errors.New("error saving quick item")
	} else {
		slog.Info("quick save item saved", "user_id", user.UserID, "niin", quick.Niin)
		return nil
	}
}

func (repo *UserSavesRepositoryImpl) DeleteQuickSaveItemByUser(user *bootstrap.User, quick model.UserItemsQuick) error {
	stmt := UserItemsQuick.
		DELETE().
		WHERE(UserItemsQuick.UserID.EQ(String(user.UserID)).
			AND(UserItemsQuick.Niin.EQ(String(quick.Niin))))

	_, err := stmt.Exec(repo.Db)

	if err != nil {
		return errors.New("error deleting quick item")
	} else {
		slog.Info("quick save item deleted", "user_id", user.UserID, "niin", quick.Niin)
		return nil
	}
}

func (repo *UserSavesRepositoryImpl) UpsertQuickSaveItemListByUser(user *bootstrap.User, quickItems []model.UserItemsQuick) error {

	var failedNiins []string
	for _, val := range quickItems {
		stmt := UserItemsQuick.INSERT(UserItemsQuick.UserID, UserItemsQuick.Niin, UserItemsQuick.ItemName,
			UserItemsQuick.ImageLocation, UserItemsQuick.ItemComment, UserItemsQuick.SaveTime).
			MODEL(val).
			ON_CONFLICT(UserItemsQuick.UserID, UserItemsQuick.Niin).
			DO_UPDATE(
				SET(
					UserItemsQuick.ImageLocation.
						SET(String(*val.ImageLocation)),
					UserItemsQuick.ItemComment.
						SET(String(*val.ItemComment)),
					UserItemsQuick.SaveTime.
						SET(Timestamp(time.Now().Year(),
							time.Now().Month(), time.Now().Day(), time.Now().Hour(),
							time.Now().Minute(), 0))).
					WHERE(
						UserItemsQuick.UserID.EQ(String(user.UserID)).
							AND(UserItemsQuick.Niin.EQ(String(val.Niin)))))

		err := stmt.Query(repo.Db, &quickItems)

		if err != nil {
			failedNiins = append(failedNiins, val.Niin)
		}

	}

	if len(failedNiins) > 0 {
		return errors.New(fmt.Sprintf("failed to save following items: %s", failedNiins))
	} else {
		slog.Info("quick save item list inserted", "user_id", user.UserID)
		return nil
	}

}

func (repo *UserSavesRepositoryImpl) DeleteAllQuickSaveItemsByUser(user *bootstrap.User) error {
	stmt := UserItemsQuick.DELETE().
		WHERE(UserItemsQuick.UserID.EQ(String(user.UserID)))

	_, err := stmt.Exec(repo.Db)

	if err != nil {
		return errors.New("error deleting quick items")
	} else {
		slog.Info("all quick save items deleted", "user_id", user.UserID)
		return nil
	}
}
