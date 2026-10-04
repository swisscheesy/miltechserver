package lists

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"miltechserver/.gen/miltech_ng/public/model"
	. "miltechserver/.gen/miltech_ng/public/table"
	"miltechserver/api/response"
	"miltechserver/api/shops/shared"
	"miltechserver/bootstrap"

	. "github.com/go-jet/jet/v2/postgres"
)

type RepositoryImpl struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *RepositoryImpl {
	return &RepositoryImpl{db: db}
}

func (repo *RepositoryImpl) CreateShopList(ctx context.Context, user *bootstrap.User, list model.ShopLists) (*response.ShopListWithUsername, error) {
	tx, err := repo.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if err := shared.AuthorizeListMutation(ctx, tx, user.UserID, list.ShopID, nil, nil, false); err != nil {
		return nil, err
	}

	stmt := ShopLists.INSERT(
		ShopLists.ID,
		ShopLists.ShopID,
		ShopLists.CreatedBy,
		ShopLists.Description,
		ShopLists.CreatedAt,
		ShopLists.UpdatedAt,
	).MODEL(list)

	_, err = stmt.ExecContext(ctx, tx)
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

	err = selectStmt.QueryContext(ctx, tx, &result)
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

	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return createdListWithUsername, nil
}

func (repo *RepositoryImpl) GetShopLists(ctx context.Context, user *bootstrap.User, shopID string) ([]response.ShopListWithUsername, error) {
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

	err := stmt.QueryContext(ctx, repo.db, &results)
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

func (repo *RepositoryImpl) GetShopListByID(ctx context.Context, user *bootstrap.User, listID string) (*response.ShopListWithUsername, error) {
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

	err := stmt.QueryContext(ctx, repo.db, &result)
	if err != nil {
		if shared.ErrorsIsNoRows(err) {
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

func (repo *RepositoryImpl) UpdateShopList(ctx context.Context, user *bootstrap.User, list model.ShopLists) error {
	tx, err := repo.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := shared.AuthorizeListMutation(ctx, tx, user.UserID, "", []string{list.ID}, nil, false); err != nil {
		return err
	}

	stmt := ShopLists.UPDATE(
		ShopLists.Description,
		ShopLists.UpdatedAt,
	).MODEL(list).
		WHERE(ShopLists.ID.EQ(String(list.ID)))

	result, err := stmt.ExecContext(ctx, tx)
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

	if err := ctx.Err(); err != nil {
		return err
	}
	return tx.Commit()
}

func (repo *RepositoryImpl) DeleteShopList(ctx context.Context, user *bootstrap.User, listID string) error {
	tx, err := repo.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := shared.AuthorizeListMutation(ctx, tx, user.UserID, "", []string{listID}, nil, true); err != nil {
		return err
	}

	var inUse bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM equipment_services WHERE list_id=$1)`, listID).Scan(&inUse); err != nil {
		return fmt.Errorf("failed to check list dependencies: %w", err)
	}
	if inUse {
		return errors.New("list is in use")
	}
	// Preserve notifications even on existing schemas. The FK migration remains
	// required for older binaries and other writers before rollout.
	if _, err := ShopVehicleNotifications.UPDATE(ShopVehicleNotifications.AttachedShopList).
		SET(ShopVehicleNotifications.AttachedShopList.SET(StringExp(NULL))).
		WHERE(ShopVehicleNotifications.AttachedShopList.EQ(String(listID))).ExecContext(ctx, tx); err != nil {
		return fmt.Errorf("failed to detach list notifications: %w", err)
	}

	stmt := ShopLists.DELETE().
		WHERE(ShopLists.ID.EQ(String(listID)))

	result, err := stmt.ExecContext(ctx, tx)
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

	if err := ctx.Err(); err != nil {
		return err
	}
	return tx.Commit()
}
