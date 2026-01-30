package categories

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

func (repo *RepositoryImpl) GetByUser(user *bootstrap.User) ([]model.UserItemCategory, error) {
	var categories []model.UserItemCategory

	stmt := SELECT(UserItemCategory.AllColumns).
		FROM(UserItemCategory).
		WHERE(UserItemCategory.UserUID.EQ(String(user.UserID)))

	err := stmt.Query(repo.db, &categories)
	if err != nil {
		return nil, fmt.Errorf("error retrieving item categories for user %s", user.UserID)
	}

	slog.Info("item categories retrieved for user", "user_id", user.UserID)
	return categories, nil
}

func (repo *RepositoryImpl) Upsert(user *bootstrap.User, itemCategory model.UserItemCategory) error {
	stmt := UserItemCategory.
		INSERT(UserItemCategory.ID, UserItemCategory.UserUID,
			UserItemCategory.Name, UserItemCategory.Comment,
			UserItemCategory.Image, UserItemCategory.LastUpdated).
		MODEL(itemCategory).
		ON_CONFLICT(UserItemCategory.ID, UserItemCategory.UserUID).
		DO_UPDATE(
			SET(UserItemCategory.Name.
				SET(String(itemCategory.Name)),
				UserItemCategory.Comment.
					SET(String(*itemCategory.Comment)),
				UserItemCategory.Image.
					SET(String(*itemCategory.Image)),
				UserItemCategory.LastUpdated.
					SET(TimestampT(*itemCategory.LastUpdated))).
				WHERE(UserItemCategory.UserUID.EQ(String(user.UserID)))).
		RETURNING(UserItemCategory.AllColumns)

	_, err := stmt.Exec(repo.db)
	if err != nil {
		return errors.New("error saving item category")
	}

	slog.Info("item category saved", "user_id", user.UserID, "category_name", itemCategory.Name)
	return nil
}

func (repo *RepositoryImpl) Delete(user *bootstrap.User, itemCategory model.UserItemCategory) error {
	tx, err := repo.db.Begin()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	itemsStmt := UserItemsCategorized.DELETE().
		WHERE(UserItemsCategorized.CategoryID.EQ(String(itemCategory.ID)))

	_, err = itemsStmt.Exec(tx)
	if err != nil {
		return errors.New("error deleting categorized items")
	}

	catStmt := UserItemCategory.DELETE().
		WHERE(UserItemCategory.UserUID.EQ(String(user.UserID)).
			AND(UserItemCategory.ID.EQ(String(itemCategory.ID))))

	_, err = catStmt.Exec(tx)
	if err != nil {
		return errors.New("error deleting item category")
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	slog.Info("item category and categorized items deleted", "user_id", user.UserID, "category_uuid", itemCategory.ID)
	return nil
}

func (repo *RepositoryImpl) DeleteAll(user *bootstrap.User) error {
	tx, err := repo.db.Begin()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	itemsStmt := UserItemsCategorized.DELETE().
		WHERE(UserItemsCategorized.UserID.EQ(String(user.UserID)))

	_, err = itemsStmt.Exec(tx)
	if err != nil {
		return errors.New("error deleting all categorized items: " + err.Error())
	}

	catStmt := UserItemCategory.DELETE().
		WHERE(UserItemCategory.UserUID.EQ(String(user.UserID)))

	_, err = catStmt.Exec(tx)
	if err != nil {
		return errors.New("error deleting all item categories: " + err.Error())
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	slog.Info("all item categories and categorized items deleted", "user_id", user.UserID)
	return nil
}
