package serialized

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

func (repo *RepositoryImpl) GetByUser(user *bootstrap.User) ([]model.UserItemsSerialized, error) {
	var items []model.UserItemsSerialized

	stmt := SELECT(UserItemsSerialized.AllColumns).
		FROM(UserItemsSerialized).
		WHERE(UserItemsSerialized.UserID.EQ(String(user.UserID)))

	err := stmt.Query(repo.db, &items)
	if err != nil {
		return nil, errors.New("user saves not found")
	}

	slog.Info("serialized saves retrieved for user", "user_id", user.UserID)
	return items, nil
}

func (repo *RepositoryImpl) Upsert(user *bootstrap.User, item model.UserItemsSerialized) error {
	stmt := UserItemsSerialized.INSERT(UserItemsSerialized.UserID, UserItemsSerialized.Niin, UserItemsSerialized.ItemName,
		UserItemsSerialized.Image, UserItemsSerialized.SaveTime, UserItemsSerialized.ItemComment, UserItemsSerialized.LastUpdated, UserItemsSerialized.ID, UserItemsSerialized.Nickname).
		MODEL(item).
		ON_CONFLICT(UserItemsSerialized.UserID, UserItemsSerialized.Niin, UserItemsSerialized.ID).
		DO_UPDATE(
			SET(UserItemsSerialized.Image.
				SET(String(*item.Image)),
				UserItemsSerialized.LastUpdated.SET(TimestampT(*item.LastUpdated)),
				UserItemsSerialized.ItemComment.SET(String(*item.ItemComment)),
				UserItemsSerialized.Nickname.SET(String(*item.Nickname)))).
		RETURNING(UserItemsSerialized.AllColumns)

	err := stmt.Query(repo.db, &item)
	if err != nil {
		return errors.New("error saving serialized item " + err.Error())
	}

	slog.Info("serialized save item saved", "user_id", user.UserID, "niin", item.Niin)
	return nil
}

func (repo *RepositoryImpl) UpsertBatch(user *bootstrap.User, items []model.UserItemsSerialized) error {
	for _, val := range items {
		stmt := UserItemsSerialized.INSERT(UserItemsSerialized.UserID, UserItemsSerialized.Niin, UserItemsSerialized.ItemName,
			UserItemsSerialized.Image, UserItemsSerialized.SaveTime, UserItemsSerialized.ItemComment, UserItemsSerialized.LastUpdated, UserItemsSerialized.ID, UserItemsSerialized.Nickname).
			MODEL(val).
			ON_CONFLICT(UserItemsSerialized.UserID, UserItemsSerialized.Niin, UserItemsSerialized.ID).
			DO_UPDATE(SET(UserItemsSerialized.Image.
				SET(String(*val.Image)),
				UserItemsSerialized.LastUpdated.SET(TimestampT(*val.LastUpdated)),
				UserItemsSerialized.ItemComment.SET(String(*val.ItemComment)),
				UserItemsSerialized.Nickname.SET(String(*val.Nickname)))).
			RETURNING(UserItemsSerialized.AllColumns)

		err := stmt.Query(repo.db, &val)
		if err != nil {
			return errors.New("error saving serialized items")
		}
	}

	slog.Info("serialized save item list saved", "user_id", user.UserID)
	return nil
}

func (repo *RepositoryImpl) Delete(user *bootstrap.User, item model.UserItemsSerialized) error {
	stmt := UserItemsSerialized.
		DELETE().
		WHERE(UserItemsSerialized.UserID.EQ(String(user.UserID)).
			AND(UserItemsSerialized.Niin.EQ(String(item.Niin))))

	_, err := stmt.Exec(repo.db)
	if err != nil {
		return errors.New("error deleting serialized item: " + err.Error())
	}

	slog.Info("serialized save item and associated image deleted", "user_id", user.UserID, "niin", item.Niin)
	return nil
}

func (repo *RepositoryImpl) DeleteAll(user *bootstrap.User) error {
	stmt := UserItemsSerialized.DELETE().
		WHERE(UserItemsSerialized.UserID.EQ(String(user.UserID)))

	_, err := stmt.Exec(repo.db)
	if err != nil {
		return errors.New("error deleting all serialized items: " + err.Error())
	}

	slog.Info("all serialized save items and associated images deleted", "user_id", user.UserID)
	return nil
}
