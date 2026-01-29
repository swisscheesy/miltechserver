package lists

import (
	"database/sql"
	"errors"
	"fmt"
	"miltechserver/.gen/miltech_ng/public/model"
	. "miltechserver/.gen/miltech_ng/public/table"
	"miltechserver/api/response"
	"miltechserver/bootstrap"

	. "github.com/go-jet/jet/v2/postgres"
)

type RepositoryImpl struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *RepositoryImpl {
	return &RepositoryImpl{db: db}
}

func (repo *RepositoryImpl) CreateShopList(user *bootstrap.User, list model.ShopLists) (*response.ShopListWithUsername, error) {
	stmt := ShopLists.INSERT(
		ShopLists.ID,
		ShopLists.ShopID,
		ShopLists.CreatedBy,
		ShopLists.Description,
		ShopLists.CreatedAt,
		ShopLists.UpdatedAt,
	).MODEL(list)

	_, err := stmt.Exec(repo.db)
	if err != nil {
		return nil, fmt.Errorf("failed to create shop list: %w", err)
	}

	selectStmt := SELECT(
		ShopLists.ID,
		ShopLists.ShopID,
		ShopLists.CreatedBy,
		ShopLists.Description,
		ShopLists.CreatedAt,
		ShopLists.UpdatedAt,
		Users.Username.AS("created_by_username"),
	).FROM(
		ShopLists.
			LEFT_JOIN(Users, Users.UID.EQ(ShopLists.CreatedBy)),
	).WHERE(
		ShopLists.ID.EQ(String(list.ID)),
	)

	var result struct {
		model.ShopLists
		CreatedByUsername *string `sql:"created_by_username"`
	}

	err = selectStmt.Query(repo.db, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to get created shop list with username: %w", err)
	}

	createdListWithUsername := &response.ShopListWithUsername{
		ID:                result.ID,
		ShopID:            result.ShopID,
		CreatedBy:         result.CreatedBy,
		CreatedByUsername: result.CreatedByUsername,
		Description:       result.Description,
		CreatedAt:         &result.CreatedAt,
		UpdatedAt:         &result.UpdatedAt,
	}

	return createdListWithUsername, nil
}

func (repo *RepositoryImpl) GetShopLists(user *bootstrap.User, shopID string) ([]response.ShopListWithUsername, error) {
	stmt := SELECT(
		ShopLists.ID,
		ShopLists.ShopID,
		ShopLists.CreatedBy,
		ShopLists.Description,
		ShopLists.CreatedAt,
		ShopLists.UpdatedAt,
		Users.Username.AS("created_by_username"),
	).FROM(
		ShopLists.
			LEFT_JOIN(Users, Users.UID.EQ(ShopLists.CreatedBy)),
	).WHERE(
		ShopLists.ShopID.EQ(String(shopID)),
	).ORDER_BY(ShopLists.CreatedAt.DESC())

	var results []struct {
		model.ShopLists
		CreatedByUsername *string `sql:"created_by_username"`
	}

	err := stmt.Query(repo.db, &results)
	if err != nil {
		return nil, fmt.Errorf("failed to get shop lists with usernames: %w", err)
	}

	lists := make([]response.ShopListWithUsername, len(results))
	for i, r := range results {
		lists[i] = response.ShopListWithUsername{
			ID:                r.ID,
			ShopID:            r.ShopID,
			CreatedBy:         r.CreatedBy,
			CreatedByUsername: r.CreatedByUsername,
			Description:       r.Description,
			CreatedAt:         &r.CreatedAt,
			UpdatedAt:         &r.UpdatedAt,
		}
	}

	return lists, nil
}

func (repo *RepositoryImpl) GetShopListByID(user *bootstrap.User, listID string) (*response.ShopListWithUsername, error) {
	stmt := SELECT(
		ShopLists.ID,
		ShopLists.ShopID,
		ShopLists.CreatedBy,
		ShopLists.Description,
		ShopLists.CreatedAt,
		ShopLists.UpdatedAt,
		Users.Username.AS("created_by_username"),
	).FROM(
		ShopLists.
			LEFT_JOIN(Users, Users.UID.EQ(ShopLists.CreatedBy)),
	).WHERE(
		ShopLists.ID.EQ(String(listID)),
	)

	var result struct {
		model.ShopLists
		CreatedByUsername *string `sql:"created_by_username"`
	}

	err := stmt.Query(repo.db, &result)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("shop list not found")
		}
		return nil, fmt.Errorf("failed to get shop list: %w", err)
	}

	listWithUsername := &response.ShopListWithUsername{
		ID:                result.ID,
		ShopID:            result.ShopID,
		CreatedBy:         result.CreatedBy,
		CreatedByUsername: result.CreatedByUsername,
		Description:       result.Description,
		CreatedAt:         &result.CreatedAt,
		UpdatedAt:         &result.UpdatedAt,
	}

	return listWithUsername, nil
}

func (repo *RepositoryImpl) UpdateShopList(user *bootstrap.User, list model.ShopLists) error {
	stmt := ShopLists.UPDATE(
		ShopLists.Description,
		ShopLists.UpdatedAt,
	).MODEL(list).
		WHERE(ShopLists.ID.EQ(String(list.ID)))

	result, err := stmt.Exec(repo.db)
	if err != nil {
		return fmt.Errorf("failed to update shop list: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return errors.New("shop list not found")
	}

	return nil
}

func (repo *RepositoryImpl) DeleteShopList(user *bootstrap.User, listID string) error {
	stmt := ShopLists.DELETE().
		WHERE(ShopLists.ID.EQ(String(listID)))

	result, err := stmt.Exec(repo.db)
	if err != nil {
		return fmt.Errorf("failed to delete shop list: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return errors.New("shop list not found")
	}

	return nil
}
