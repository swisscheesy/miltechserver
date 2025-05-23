package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"miltechserver/.gen/miltech_ng/public/model"
	. "miltechserver/.gen/miltech_ng/public/table"
	"miltechserver/bootstrap"
	"miltechserver/helper"

	. "github.com/go-jet/jet/v2/postgres"
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
		UserItemsQuick.ImageLocation, UserItemsQuick.ItemComment, UserItemsQuick.SaveTime, UserItemsQuick.LastUpdated, UserItemsQuick.ID).
		MODEL(quick).
		ON_CONFLICT(UserItemsQuick.UserID, UserItemsQuick.Niin).
		DO_UPDATE(
			SET(
				UserItemsQuick.UserID.SET(String(user.UserID)),
				UserItemsQuick.LastUpdated.SET(TimestampT(*quick.LastUpdated)),
				UserItemsQuick.ImageLocation.SET(String(*quick.ImageLocation)),
				UserItemsQuick.ItemComment.SET(String(*quick.ItemComment)),
				UserItemsQuick.ID.SET(String(quick.ID)),
				UserItemsQuick.SaveTime.SET(TimestampT(*quick.SaveTime))).
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
			UserItemsQuick.ImageLocation, UserItemsQuick.ItemComment, UserItemsQuick.SaveTime, UserItemsQuick.LastUpdated, UserItemsQuick.ID).
			MODEL(val).
			ON_CONFLICT(UserItemsQuick.UserID, UserItemsQuick.Niin).
			DO_UPDATE(
				SET(
					UserItemsQuick.ImageLocation.
						SET(String(*val.ImageLocation)),
					UserItemsQuick.ItemComment.
						SET(String(*val.ItemComment)),
					UserItemsQuick.SaveTime.
						SET(TimestampT(*val.SaveTime)),
					UserItemsQuick.LastUpdated.
						SET(TimestampT(*val.LastUpdated))).
					WHERE(
						UserItemsQuick.UserID.EQ(String(user.UserID)).
							AND(UserItemsQuick.Niin.EQ(String(val.Niin)))))

		err := stmt.Query(repo.Db, &quickItems)

		if err != nil {
			failedNiins = append(failedNiins, val.Niin)
		}

	}

	if len(failedNiins) > 0 {
		return fmt.Errorf(fmt.Sprintf("failed to save following items: %s", failedNiins))
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

func (repo *UserSavesRepositoryImpl) GetSerializedItemsByUserId(user *bootstrap.User) ([]model.UserItemsSerialized, error) {
	var items []model.UserItemsSerialized

	if user != nil {
		stmt := SELECT(
			UserItemsSerialized.AllColumns).
			FROM(UserItemsSerialized).
			WHERE(UserItemsSerialized.UserID.EQ(String(user.UserID)))

		err := stmt.Query(repo.Db, &items)
		if err != nil {
			return nil, errors.New(fmt.Sprintf("error retrieving serialized saves for user %s", user.UserID))
		} else {
			slog.Info("serialized saves retrieved for user", "user_id", user.UserID)
			return items, nil
		}
	} else {
		return nil, errors.New("valid user not found")
	}
}

func (repo *UserSavesRepositoryImpl) UpsertSerializedSaveItemByUser(user *bootstrap.User, serializedItem model.UserItemsSerialized) error {
	stmt := UserItemsSerialized.INSERT(UserItemsSerialized.UserID, UserItemsSerialized.Niin, UserItemsSerialized.ItemName,
		UserItemsSerialized.Serial,
		UserItemsSerialized.ImageLocation, UserItemsSerialized.SaveTime, UserItemsSerialized.ItemComment, UserItemsSerialized.LastUpdated).
		MODEL(serializedItem).
		ON_CONFLICT(UserItemsSerialized.UserID, UserItemsSerialized.Niin, UserItemsSerialized.Serial).
		DO_UPDATE(
			SET(UserItemsSerialized.ImageLocation.
				SET(String(*serializedItem.ImageLocation)),
				UserItemsSerialized.ItemComment.
					SET(String(*serializedItem.ItemComment)),
				UserItemsSerialized.SaveTime.
					SET(TimestampT(*serializedItem.SaveTime)),
				UserItemsSerialized.LastUpdated.
					SET(TimestampT(*serializedItem.LastUpdated))).
				WHERE(UserItemsSerialized.UserID.EQ(String(user.UserID)).
					AND(UserItemsSerialized.Niin.EQ(String(serializedItem.Niin))))).
		RETURNING(
			UserItemsSerialized.AllColumns)

	err := stmt.Query(repo.Db, &serializedItem)

	if err != nil {
		return errors.New("error saving serialized item")
	}

	slog.Info("serialized save item saved", "user_id", user.UserID, "niin", serializedItem.Niin)
	return nil
}

func (repo *UserSavesRepositoryImpl) DeleteAllSerializedItemsByUser(user *bootstrap.User) error {
	stmt := UserItemsSerialized.DELETE().
		WHERE(UserItemsSerialized.UserID.EQ(String(user.UserID)))

	_, err := stmt.Exec(repo.Db)

	if err != nil {
		return errors.New("error deleting serialized items")
	} else {
		slog.Info("all serialized save items deleted", "user_id", user.UserID)
		return nil
	}
}

func (repo *UserSavesRepositoryImpl) UpsertSerializedSaveItemListByUser(user *bootstrap.User, serializedItems []model.UserItemsSerialized) error {

	var failedNiins []string
	for _, val := range serializedItems {
		stmt := UserItemsSerialized.INSERT(UserItemsSerialized.UserID, UserItemsSerialized.Niin, UserItemsSerialized.ItemName,
			UserItemsSerialized.Serial,
			UserItemsSerialized.ImageLocation, UserItemsSerialized.SaveTime, UserItemsSerialized.ItemComment).
			MODEL(val).
			ON_CONFLICT(UserItemsSerialized.UserID, UserItemsSerialized.Niin, UserItemsSerialized.Serial).
			DO_UPDATE(
				SET(UserItemsSerialized.ImageLocation.
					SET(String(*val.ImageLocation)),
					UserItemsSerialized.ItemComment.
						SET(String(*val.ItemComment)),
					UserItemsSerialized.SaveTime.
						SET(helper.CurrentTimeDB())).
					WHERE(UserItemsSerialized.UserID.EQ(String(user.UserID)).
						AND(UserItemsSerialized.Niin.EQ(String(val.Niin)).
							AND(UserItemsSerialized.Serial.EQ(String(val.Serial))))))

		err := stmt.Query(repo.Db, &serializedItems)

		if err != nil {
			failedNiins = append(failedNiins, val.Niin)
		}

	}
	if len(failedNiins) > 0 {
		return errors.New(fmt.Sprintf("failed to save following items: %s", failedNiins))
	} else {
		slog.Info("serialized save item list inserted", "user_id", user.UserID)
		return nil
	}
}

func (repo *UserSavesRepositoryImpl) DeleteSerializedSaveItemByUser(user *bootstrap.User, serializedItem model.UserItemsSerialized) error {
	stmt := UserItemsSerialized.
		DELETE().
		WHERE(UserItemsSerialized.UserID.EQ(String(user.UserID)).
			AND(UserItemsSerialized.Niin.EQ(String(serializedItem.Niin)).
				AND(UserItemsSerialized.Serial.EQ(String(serializedItem.Serial)))))

	_, err := stmt.Exec(repo.Db)

	if err != nil {
		return errors.New("error deleting serialized item")
	}

	slog.Info("serialized save item deleted", "user_id", user.UserID, "niin", serializedItem.Niin)
	return nil
}

func (repo *UserSavesRepositoryImpl) GetUserItemCategories(user *bootstrap.User) ([]model.UserItemCategory, error) {
	var categories []model.UserItemCategory

	if user != nil {
		stmt := SELECT(UserItemCategory.AllColumns).
			FROM(UserItemCategory).
			WHERE(UserItemCategory.UserUID.EQ(String(user.UserID)))

		err := stmt.Query(repo.Db, &categories)
		if err != nil {
			return nil, errors.New(fmt.Sprintf("error retrieving item categories for user %s", user.UserID))
		} else {
			slog.Info("item categories retrieved for user", "user_id", user.UserID)
			return categories, nil
		}
	} else {
		return nil, errors.New("valid user not found")
	}
}

func (repo *UserSavesRepositoryImpl) UpsertUserItemCategory(user *bootstrap.User, itemCategory model.UserItemCategory) error {
	stmt := UserItemCategory.
		INSERT(UserItemCategory.ID, UserItemCategory.UserUID,
			UserItemCategory.Name, UserItemCategory.Comment,
			UserItemCategory.ImageLocation).
		MODEL(itemCategory).
		ON_CONFLICT(UserItemCategory.ID, UserItemCategory.UserUID).
		DO_UPDATE(
			SET(UserItemCategory.Name.
				SET(String(itemCategory.Name)),
				UserItemCategory.Comment.
					SET(String(*itemCategory.Comment)), //TODO Was *itemCategory.Comment
				UserItemCategory.ImageLocation.
					SET(String(*itemCategory.ImageLocation))).
				WHERE(UserItemCategory.UserUID.EQ(String(user.UserID)))).
		RETURNING(UserItemCategory.AllColumns)

	_, err := stmt.Exec(repo.Db)

	if err != nil {
		return errors.New("error saving item category")
	}

	slog.Info("item category saved", "user_id", user.UserID, "category_name", itemCategory.Name)
	return nil
}

func (repo *UserSavesRepositoryImpl) DeleteUserItemCategory(user *bootstrap.User, itemCategoryUuid string) error {
	stmt := UserItemCategory.DELETE().
		WHERE(UserItemCategory.UserUID.EQ(String(user.UserID)).
			AND(UserItemCategory.ID.EQ(String(itemCategoryUuid))))

	_, err := stmt.Exec(repo.Db)

	if err != nil {
		return errors.New("error deleting item category")
	}

	slog.Info("item category deleted", "user_id", user.UserID, "category_uuid", itemCategoryUuid)
	return nil
}

func (repo *UserSavesRepositoryImpl) GetCategorizedItemsByCategoryUuid(user *bootstrap.User, categoryUuid string) ([]model.UserItemsCategorized, error) {
	var items []model.UserItemsCategorized

	if user != nil {
		stmt :=
			SELECT(
				UserItemsCategorized.AllColumns,
			).WHERE(
				UserItemsCategorized.UserID.EQ(String(user.UserID)).
					AND(UserItemsCategorized.CategoryID.EQ(String(categoryUuid))))

		err := stmt.Query(repo.Db, &items)

		if err != nil {
			return nil, errors.New(fmt.Sprintf("error retrieving categorized items for user %s", user.UserID))
		} else {
			slog.Info("categorized items retrieved for user", "user_id", user.UserID)
			return items, nil
		}
	} else {
		return nil, errors.New("valid user not found")
	}

}
