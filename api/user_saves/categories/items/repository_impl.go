package items

import (
	"database/sql"
	"errors"
	"fmt"
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

func (repo *RepositoryImpl) GetByCategory(user *bootstrap.User, category model.UserItemCategory) ([]model.UserItemsCategorized, error) {
	var items []model.UserItemsCategorized

	stmt := SELECT(UserItemsCategorized.AllColumns).
		FROM(UserItemsCategorized).
		WHERE(UserItemsCategorized.UserID.EQ(String(user.UserID)).
			AND(UserItemsCategorized.CategoryID.EQ(String(category.ID))))

	err := stmt.Query(repo.db, &items)
	if err != nil {
		return nil, fmt.Errorf("error retrieving categorized items for user %s", user.UserID)
	}

	return items, nil
}

func (repo *RepositoryImpl) GetByUser(user *bootstrap.User) ([]model.UserItemsCategorized, error) {
	var items []model.UserItemsCategorized

	stmt := SELECT(UserItemsCategorized.AllColumns).
		FROM(UserItemsCategorized).
		WHERE(UserItemsCategorized.UserID.EQ(String(user.UserID)))

	err := stmt.Query(repo.db, &items)
	if err != nil {
		return nil, fmt.Errorf("error retrieving categorized items for user %s", user.UserID)
	}

	slog.Info("categorized items retrieved for user", "user_id", user.UserID)
	return items, nil
}

func (repo *RepositoryImpl) Upsert(user *bootstrap.User, categorizedItem model.UserItemsCategorized) error {
	stmt := UserItemsCategorized.
		INSERT(
			UserItemsCategorized.UserID,
			UserItemsCategorized.Niin,
			UserItemsCategorized.ItemName,
			UserItemsCategorized.Quantity,
			UserItemsCategorized.EquipModel,
			UserItemsCategorized.Uoc,
			UserItemsCategorized.CategoryID,
			UserItemsCategorized.SaveTime,
			UserItemsCategorized.Image,
			UserItemsCategorized.LastUpdated,
			UserItemsCategorized.ID,
			UserItemsCategorized.Nickname,
		).
		MODEL(categorizedItem).
		ON_CONFLICT(
			UserItemsCategorized.Niin,
			UserItemsCategorized.CategoryID,
			UserItemsCategorized.ID,
		).
		DO_UPDATE(
			SET(
				UserItemsCategorized.ItemName.SET(String(*categorizedItem.ItemName)),
				UserItemsCategorized.Quantity.SET(Int32(*categorizedItem.Quantity)),
				UserItemsCategorized.EquipModel.SET(String(*categorizedItem.EquipModel)),
				UserItemsCategorized.Uoc.SET(String(*categorizedItem.Uoc)),
				UserItemsCategorized.SaveTime.SET(TimestampT(*categorizedItem.SaveTime)),
				UserItemsCategorized.Image.SET(String(*categorizedItem.Image)),
				UserItemsCategorized.LastUpdated.SET(TimestampT(*categorizedItem.LastUpdated)),
				UserItemsCategorized.Nickname.SET(String(*categorizedItem.Nickname)),
			).
				WHERE(
					UserItemsCategorized.UserID.EQ(String(user.UserID)).
						AND(UserItemsCategorized.Niin.EQ(String(categorizedItem.Niin))).
						AND(UserItemsCategorized.CategoryID.EQ(String(categorizedItem.CategoryID))).
						AND(UserItemsCategorized.ID.EQ(String(categorizedItem.ID))),
				),
		).
		RETURNING(UserItemsCategorized.AllColumns)

	err := stmt.Query(repo.db, &categorizedItem)
	if err != nil {
		return errors.New("error saving categorized item: " + err.Error())
	}

	slog.Info("categorized item saved", "user_id", user.UserID, "niin", categorizedItem.Niin, "category_id", categorizedItem.CategoryID)
	return nil
}

func (repo *RepositoryImpl) UpsertBatch(user *bootstrap.User, categorizedItems []model.UserItemsCategorized) error {
	var failedNiins []string
	for _, val := range categorizedItems {
		stmt := UserItemsCategorized.
			INSERT(UserItemsCategorized.UserID, UserItemsCategorized.Niin, UserItemsCategorized.ItemName,
				UserItemsCategorized.Quantity, UserItemsCategorized.EquipModel, UserItemsCategorized.Uoc,
				UserItemsCategorized.CategoryID, UserItemsCategorized.SaveTime, UserItemsCategorized.Image,
				UserItemsCategorized.LastUpdated, UserItemsCategorized.ID, UserItemsCategorized.Nickname).
			MODEL(val).
			ON_CONFLICT(UserItemsCategorized.UserID, UserItemsCategorized.Niin, UserItemsCategorized.CategoryID, UserItemsCategorized.ID).
			DO_UPDATE(
				SET(
					UserItemsCategorized.ItemName.SET(String(*val.ItemName)),
					UserItemsCategorized.Quantity.SET(Int32(*val.Quantity)),
					UserItemsCategorized.EquipModel.SET(String(*val.EquipModel)),
					UserItemsCategorized.Uoc.SET(String(*val.Uoc)),
					UserItemsCategorized.SaveTime.SET(TimestampT(*val.SaveTime)),
					UserItemsCategorized.Image.SET(String(*val.Image)),
					UserItemsCategorized.LastUpdated.SET(TimestampT(*val.LastUpdated)),
					UserItemsCategorized.Nickname.SET(String(*val.Nickname)),
				).
					WHERE(
						UserItemsCategorized.UserID.EQ(String(user.UserID)).
							AND(UserItemsCategorized.Niin.EQ(String(val.Niin))).
							AND(UserItemsCategorized.CategoryID.EQ(String(val.CategoryID))).
							AND(UserItemsCategorized.ID.EQ(String(val.ID))),
					),
			)

		err := stmt.Query(repo.db, &categorizedItems)
		if err != nil {
			failedNiins = append(failedNiins, val.Niin)
		}
	}

	if len(failedNiins) > 0 {
		return fmt.Errorf("failed to save following items: %s", failedNiins)
	}

	slog.Info("categorized item list inserted", "user_id", user.UserID)
	return nil
}

func (repo *RepositoryImpl) Delete(user *bootstrap.User, categorizedItem model.UserItemsCategorized) error {
	stmt := UserItemsCategorized.
		DELETE().
		WHERE(
			UserItemsCategorized.UserID.EQ(String(user.UserID)).
				AND(UserItemsCategorized.Niin.EQ(String(categorizedItem.Niin))).
				AND(UserItemsCategorized.CategoryID.EQ(String(categorizedItem.CategoryID))).
				AND(UserItemsCategorized.ID.EQ(String(categorizedItem.ID))),
		)

	_, err := stmt.Exec(repo.db)
	if err != nil {
		return errors.New("error deleting categorized item: " + err.Error())
	}

	slog.Info("categorized item and associated image deleted", "user_id", user.UserID, "niin", categorizedItem.Niin, "category_id", categorizedItem.CategoryID)
	return nil
}

func (repo *RepositoryImpl) DeleteAll(user *bootstrap.User) error {
	stmt := UserItemsCategorized.DELETE().
		WHERE(UserItemsCategorized.UserID.EQ(String(user.UserID)))

	_, err := stmt.Exec(repo.db)
	if err != nil {
		return errors.New("error deleting all categorized items: " + err.Error())
	}

	slog.Info("all categorized items deleted", "user_id", user.UserID)
	return nil
}
