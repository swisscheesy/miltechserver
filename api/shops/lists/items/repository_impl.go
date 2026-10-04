package items

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

	"github.com/go-jet/jet/v2/postgres"
	. "github.com/go-jet/jet/v2/postgres"
)

type RepositoryImpl struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *RepositoryImpl {
	return &RepositoryImpl{db: db}
}

func (repo *RepositoryImpl) AddListItem(ctx context.Context, user *bootstrap.User, item model.ShopListItems) (*response.ShopListItemWithUsername, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := shared.ValidateItemFields(item.Niin, item.Nomenclature, item.Quantity); err != nil {
		return nil, err
	}
	if err := shared.ValidateItemEnrichment(item.Nickname, item.UnitOfMeasure); err != nil {
		return nil, err
	}

	tx, err := repo.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if err := shared.AuthorizeListMutation(ctx, tx, user.UserID, "", []string{item.ListID}, nil, false); err != nil {
		return nil, err
	}

	stmt := ShopListItems.INSERT(
		ShopListItems.ID,
		ShopListItems.ListID,
		ShopListItems.Niin,
		ShopListItems.Nomenclature,
		ShopListItems.Quantity,
		ShopListItems.AddedBy,
		ShopListItems.CreatedAt,
		ShopListItems.UpdatedAt,
		ShopListItems.Nickname,
		ShopListItems.UnitOfMeasure,
	).MODEL(item)

	_, err = stmt.ExecContext(ctx, tx)
	if err != nil {
		return nil, fmt.Errorf("failed to add list item: %w", err)
	}

	selectStmt := SELECT(
		ShopListItems.ID,
		ShopListItems.ListID,
		ShopListItems.Niin,
		ShopListItems.Nomenclature,
		ShopListItems.Quantity,
		ShopListItems.AddedBy,
		ShopListItems.CreatedAt,
		ShopListItems.UpdatedAt,
		ShopListItems.Nickname,
		ShopListItems.UnitOfMeasure,
		Users.Username.AS("added_by_username"),
	).FROM(
		ShopListItems.
			LEFT_JOIN(Users, Users.UID.EQ(ShopListItems.AddedBy)),
	).WHERE(
		ShopListItems.ID.EQ(String(item.ID)),
	)

	var result struct {
		model.ShopListItems
		AddedByUsername *string `sql:"added_by_username"`
	}

	err = selectStmt.QueryContext(ctx, tx, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to get created list item with username: %w", err)
	}

	createdItemWithUsername := &response.ShopListItemWithUsername{
		ID:              result.ID,
		ListID:          result.ListID,
		Niin:            result.Niin,
		Nomenclature:    result.Nomenclature,
		Quantity:        result.Quantity,
		AddedBy:         result.AddedBy,
		AddedByUsername: result.AddedByUsername,
		CreatedAt:       &result.CreatedAt,
		UpdatedAt:       &result.UpdatedAt,
		Nickname:        result.Nickname,
		UnitOfMeasure:   result.UnitOfMeasure,
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return createdItemWithUsername, nil
}

func (repo *RepositoryImpl) GetListItems(ctx context.Context, user *bootstrap.User, listID string) ([]response.ShopListItemWithUsername, error) {
	stmt := SELECT(
		ShopListItems.ID,
		ShopListItems.ListID,
		ShopListItems.Niin,
		ShopListItems.Nomenclature,
		ShopListItems.Quantity,
		ShopListItems.AddedBy,
		ShopListItems.CreatedAt,
		ShopListItems.UpdatedAt,
		ShopListItems.Nickname,
		ShopListItems.UnitOfMeasure,
		Users.Username.AS("added_by_username"),
	).FROM(
		ShopListItems.
			LEFT_JOIN(Users, Users.UID.EQ(ShopListItems.AddedBy)),
	).WHERE(
		ShopListItems.ListID.EQ(String(listID)),
	).ORDER_BY(ShopListItems.CreatedAt.ASC())

	var results []struct {
		model.ShopListItems
		AddedByUsername *string `sql:"added_by_username"`
	}

	err := stmt.QueryContext(ctx, repo.db, &results)
	if err != nil {
		return nil, fmt.Errorf("failed to get list items with usernames: %w", err)
	}

	items := make([]response.ShopListItemWithUsername, len(results))
	for i, r := range results {
		items[i] = response.ShopListItemWithUsername{
			ID:              r.ID,
			ListID:          r.ListID,
			Niin:            r.Niin,
			Nomenclature:    r.Nomenclature,
			Quantity:        r.Quantity,
			AddedBy:         r.AddedBy,
			AddedByUsername: r.AddedByUsername,
			CreatedAt:       &r.CreatedAt,
			UpdatedAt:       &r.UpdatedAt,
			Nickname:        r.Nickname,
			UnitOfMeasure:   r.UnitOfMeasure,
		}
	}

	return items, nil
}

func (repo *RepositoryImpl) GetListItemByID(ctx context.Context, user *bootstrap.User, itemID string) (*model.ShopListItems, error) {
	stmt := SELECT(ShopListItems.AllColumns).
		FROM(ShopListItems).
		WHERE(ShopListItems.ID.EQ(String(itemID)))

	var item model.ShopListItems
	err := stmt.QueryContext(ctx, repo.db, &item)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("list item not found")
		}
		return nil, fmt.Errorf("failed to get list item: %w", err)
	}

	return &item, nil
}

func (repo *RepositoryImpl) UpdateListItem(ctx context.Context, user *bootstrap.User, item model.ShopListItems) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := shared.ValidateItemFields(item.Niin, item.Nomenclature, item.Quantity); err != nil {
		return err
	}
	if err := shared.ValidateItemEnrichment(item.Nickname, item.UnitOfMeasure); err != nil {
		return err
	}

	tx, err := repo.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := shared.AuthorizeListMutation(ctx, tx, user.UserID, "", nil, []string{item.ID}, false); err != nil {
		return err
	}

	stmt := ShopListItems.UPDATE(
		ShopListItems.Niin,
		ShopListItems.Nomenclature,
		ShopListItems.Quantity,
		ShopListItems.UpdatedAt,
		ShopListItems.Nickname,
		ShopListItems.UnitOfMeasure,
	).MODEL(item).
		WHERE(ShopListItems.ID.EQ(String(item.ID)))

	result, err := stmt.ExecContext(ctx, tx)
	if err != nil {
		return fmt.Errorf("failed to update list item: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return errors.New("list item not found")
	}

	if err := ctx.Err(); err != nil {
		return err
	}
	return tx.Commit()
}

func (repo *RepositoryImpl) RemoveListItem(ctx context.Context, user *bootstrap.User, itemID string) error {
	tx, err := repo.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := shared.AuthorizeListMutation(ctx, tx, user.UserID, "", nil, []string{itemID}, false); err != nil {
		return err
	}

	stmt := ShopListItems.DELETE().
		WHERE(ShopListItems.ID.EQ(String(itemID)))

	result, err := stmt.ExecContext(ctx, tx)
	if err != nil {
		return fmt.Errorf("failed to remove list item: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return errors.New("list item not found")
	}

	if err := ctx.Err(); err != nil {
		return err
	}
	return tx.Commit()
}

func (repo *RepositoryImpl) AddListItemBatch(ctx context.Context, user *bootstrap.User, items []model.ShopListItems) ([]response.ShopListItemWithUsername, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	for _, item := range items {
		if err := shared.ValidateItemFields(item.Niin, item.Nomenclature, item.Quantity); err != nil {
			return nil, err
		}
		if err := shared.ValidateItemEnrichment(item.Nickname, item.UnitOfMeasure); err != nil {
			return nil, err
		}
	}

	if len(items) == 0 {
		return []response.ShopListItemWithUsername{}, nil
	}
	listIDs := make([]string, len(items))
	for i, item := range items {
		listIDs[i] = item.ListID
	}
	tx, err := repo.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if err := shared.AuthorizeListMutation(ctx, tx, user.UserID, "", listIDs, nil, false); err != nil {
		return nil, err
	}

	stmt := ShopListItems.INSERT(
		ShopListItems.ID,
		ShopListItems.ListID,
		ShopListItems.Niin,
		ShopListItems.Nomenclature,
		ShopListItems.Quantity,
		ShopListItems.AddedBy,
		ShopListItems.CreatedAt,
		ShopListItems.UpdatedAt,
		ShopListItems.Nickname,
		ShopListItems.UnitOfMeasure,
	).MODELS(items)

	_, err = stmt.ExecContext(ctx, tx)
	if err != nil {
		return nil, fmt.Errorf("failed to add list items: %w", err)
	}

	itemIDs := make([]postgres.Expression, len(items))
	for i, item := range items {
		itemIDs[i] = String(item.ID)
	}

	selectStmt := SELECT(
		ShopListItems.ID,
		ShopListItems.ListID,
		ShopListItems.Niin,
		ShopListItems.Nomenclature,
		ShopListItems.Quantity,
		ShopListItems.AddedBy,
		ShopListItems.CreatedAt,
		ShopListItems.UpdatedAt,
		ShopListItems.Nickname,
		ShopListItems.UnitOfMeasure,
		Users.Username.AS("added_by_username"),
	).FROM(
		ShopListItems.
			LEFT_JOIN(Users, Users.UID.EQ(ShopListItems.AddedBy)),
	).WHERE(
		ShopListItems.ID.IN(itemIDs...),
	).ORDER_BY(ShopListItems.CreatedAt.ASC())

	var results []struct {
		model.ShopListItems
		AddedByUsername *string `sql:"added_by_username"`
	}

	err = selectStmt.QueryContext(ctx, tx, &results)
	if err != nil {
		return nil, fmt.Errorf("failed to get created list items with usernames: %w", err)
	}

	createdItemsWithUsername := make([]response.ShopListItemWithUsername, len(results))
	for i, r := range results {
		createdItemsWithUsername[i] = response.ShopListItemWithUsername{
			ID:              r.ID,
			ListID:          r.ListID,
			Niin:            r.Niin,
			Nomenclature:    r.Nomenclature,
			Quantity:        r.Quantity,
			AddedBy:         r.AddedBy,
			AddedByUsername: r.AddedByUsername,
			CreatedAt:       &r.CreatedAt,
			UpdatedAt:       &r.UpdatedAt,
			Nickname:        r.Nickname,
			UnitOfMeasure:   r.UnitOfMeasure,
		}
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return createdItemsWithUsername, nil
}

func (repo *RepositoryImpl) RemoveListItemBatch(ctx context.Context, user *bootstrap.User, itemIDs []string) (int64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	if user == nil || user.UserID == "" {
		return 0, errors.New("unauthorized user")
	}
	if len(itemIDs) == 0 {
		return 0, nil
	}
	tx, err := repo.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	survivors, err := shared.AuthorizeExistingListItems(ctx, tx, user.UserID, itemIDs)
	if err != nil {
		return 0, err
	}
	var count int64
	if len(survivors) > 0 {
		expressions := make([]Expression, len(survivors))
		for i, id := range survivors {
			expressions[i] = String(id)
		}
		result, err := ShopListItems.DELETE().WHERE(ShopListItems.ID.IN(expressions...)).ExecContext(ctx, tx)
		if err != nil {
			return 0, fmt.Errorf("failed to remove list items: %w", err)
		}
		count, err = result.RowsAffected()
		if err != nil {
			return 0, err
		}
	}
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return count, nil
}
