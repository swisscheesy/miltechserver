package quick

import (
	"database/sql"
	"errors"
	"log/slog"
	"miltechserver/.gen/miltech_ng/public/model"
	. "miltechserver/.gen/miltech_ng/public/table"
	"miltechserver/bootstrap"

	. "github.com/go-jet/jet/v2/postgres"
)

type RepositoryImpl struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *RepositoryImpl {
	return &RepositoryImpl{db: db}
}

func (repo *RepositoryImpl) GetByUser(user *bootstrap.User) ([]model.UserItemsQuick, error) {
	var items []model.UserItemsQuick

	stmt := SELECT(UserItemsQuick.AllColumns).
		FROM(UserItemsQuick).
		WHERE(UserItemsQuick.UserID.EQ(String(user.UserID)))

	err := stmt.Query(repo.db, &items)
	if err != nil {
		return nil, errors.New("user saves not found")
	}

	slog.Info("quick saves retrieved for user", "user_id", user.UserID)
	return items, nil
}

func (repo *RepositoryImpl) Upsert(user *bootstrap.User, quick model.UserItemsQuick) error {
	stmt := UserItemsQuick.INSERT(UserItemsQuick.UserID, UserItemsQuick.Niin, UserItemsQuick.ItemName,
		UserItemsQuick.Image, UserItemsQuick.ItemComment, UserItemsQuick.SaveTime, UserItemsQuick.LastUpdated, UserItemsQuick.ID, UserItemsQuick.Nickname).
		MODEL(quick).
		ON_CONFLICT(UserItemsQuick.UserID, UserItemsQuick.Niin, UserItemsQuick.ID).
		DO_UPDATE(
			SET(
				UserItemsQuick.LastUpdated.SET(TimestampT(*quick.LastUpdated)),
				UserItemsQuick.Image.SET(String(*quick.Image)),
				UserItemsQuick.ItemComment.SET(String(*quick.ItemComment)),
				UserItemsQuick.Nickname.SET(String(*quick.Nickname))).
				WHERE(UserItemsQuick.UserID.EQ(String(user.UserID)).
					AND(UserItemsQuick.ID.EQ(String(quick.ID))).
					AND(UserItemsQuick.UserID.EQ(String(user.UserID))).
					AND(UserItemsQuick.Niin.EQ(String(quick.Niin))))).
		RETURNING(UserItemsQuick.AllColumns)

	err := stmt.Query(repo.db, &quick)
	if err != nil {
		return errors.New("error saving quick item " + err.Error())
	}

	slog.Info("quick save item saved", "user_id", user.UserID, "niin", quick.Niin)
	return nil
}

func (repo *RepositoryImpl) UpsertBatch(user *bootstrap.User, quickItems []model.UserItemsQuick) error {
	for _, val := range quickItems {
		stmt := UserItemsQuick.INSERT(UserItemsQuick.UserID, UserItemsQuick.Niin, UserItemsQuick.ItemName,
			UserItemsQuick.Image, UserItemsQuick.ItemComment, UserItemsQuick.SaveTime, UserItemsQuick.LastUpdated, UserItemsQuick.ID, UserItemsQuick.Nickname).
			MODEL(val).
			ON_CONFLICT(UserItemsQuick.UserID, UserItemsQuick.Niin, UserItemsQuick.ID).
			DO_UPDATE(
				SET(
					UserItemsQuick.LastUpdated.SET(TimestampT(*val.LastUpdated)),
					UserItemsQuick.Image.SET(String(*val.Image)),
					UserItemsQuick.ItemComment.SET(String(*val.ItemComment)),
					UserItemsQuick.Nickname.SET(String(*val.Nickname))).
					WHERE(UserItemsQuick.UserID.EQ(String(user.UserID)).
						AND(UserItemsQuick.ID.EQ(String(val.ID))).
						AND(UserItemsQuick.Niin.EQ(String(val.Niin))))).
			RETURNING(UserItemsQuick.AllColumns)

		err := stmt.Query(repo.db, &val)
		if err != nil {
			return errors.New("error saving quick items")
		}
	}

	slog.Info("quick save item list saved", "user_id", user.UserID)
	return nil
}

func (repo *RepositoryImpl) Delete(user *bootstrap.User, quick model.UserItemsQuick) error {
	stmt := UserItemsQuick.
		DELETE().
		WHERE(UserItemsQuick.UserID.EQ(String(user.UserID)).
			AND(UserItemsQuick.Niin.EQ(String(quick.Niin))))

	_, err := stmt.Exec(repo.db)
	if err != nil {
		return errors.New("error deleting quick item: " + err.Error())
	}

	slog.Info("quick save item and associated image deleted", "user_id", user.UserID, "niin", quick.Niin)
	return nil
}

func (repo *RepositoryImpl) DeleteAll(user *bootstrap.User) error {
	stmt := UserItemsQuick.DELETE().
		WHERE(UserItemsQuick.UserID.EQ(String(user.UserID)))

	_, err := stmt.Exec(repo.db)
	if err != nil {
		return errors.New("error deleting all quick items: " + err.Error())
	}

	slog.Info("all quick save items and associated images deleted", "user_id", user.UserID)
	return nil
}
