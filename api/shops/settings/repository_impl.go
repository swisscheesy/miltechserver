package settings

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"miltechserver/.gen/miltech_ng/public/model"
	. "miltechserver/.gen/miltech_ng/public/table"
	"miltechserver/api/request"
	dbutil "miltechserver/api/shared/db"
	"miltechserver/api/shops/shared"
	"miltechserver/bootstrap"
	"strings"
	"time"

	. "github.com/go-jet/jet/v2/postgres"
)

type RepositoryImpl struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *RepositoryImpl {
	return &RepositoryImpl{db: db}
}

// GetShopAdminOnlyListsSetting retrieves the admin_only_lists setting for a shop
func (repo *RepositoryImpl) GetShopAdminOnlyListsSetting(ctx context.Context, shopID string) (bool, error) {
	stmt := SELECT(Shops.AllColumns).
		FROM(Shops).
		WHERE(Shops.ID.EQ(String(shopID)))

	var shop model.Shops
	err := stmt.QueryContext(ctx, repo.db, &shop)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, errors.New("shop not found")
		}
		return false, fmt.Errorf("failed to get admin_only_lists setting: %w", err)
	}

	return shop.AdminOnlyLists, nil
}

// UpdateShopAdminOnlyListsSetting updates the admin_only_lists setting for a shop
func (repo *RepositoryImpl) UpdateShopAdminOnlyListsSetting(ctx context.Context, user *bootstrap.User, shopID string, adminOnlyLists bool) error {
	err := dbutil.WithTxContext(ctx, repo.db, func(tx *sql.Tx) error {
		admin, _, err := shared.LockShopMutation(ctx, tx, shopID, user.UserID)
		if err != nil {
			return err
		}
		if !admin {
			return shared.ErrShopAdminRequired
		}

		now := time.Now()

		stmt := Shops.UPDATE(
			Shops.AdminOnlyLists,
			Shops.UpdatedAt,
		).SET(
			Shops.AdminOnlyLists.SET(Bool(adminOnlyLists)),
			Shops.UpdatedAt.SET(TimestampzT(now)),
		).WHERE(
			Shops.ID.EQ(String(shopID)),
		)

		result, err := stmt.ExecContext(ctx, tx)
		if err != nil {
			return fmt.Errorf("failed to update admin_only_lists setting: %w", err)
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("failed to get rows affected: %w", err)
		}

		if rowsAffected == 0 {
			return errors.New("shop not found")
		}

		return nil
	})
	if err != nil {
		return err
	}
	slog.Info("Shop admin_only_lists setting updated", "shop_id", shopID, "admin_only_lists", adminOnlyLists)
	return nil
}

// GetShopSettings retrieves all settings for a shop
func (repo *RepositoryImpl) GetShopSettings(ctx context.Context, shopID string) (*request.ShopSettings, error) {
	stmt := SELECT(Shops.AllColumns).
		FROM(Shops).
		WHERE(Shops.ID.EQ(String(shopID)))

	var shop model.Shops
	err := stmt.QueryContext(ctx, repo.db, &shop)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("shop not found")
		}
		return nil, fmt.Errorf("failed to get shop settings: %w", err)
	}

	settings := &request.ShopSettings{
		AdminOnlyLists: shop.AdminOnlyLists,
	}

	return settings, nil
}

// UpdateShopSettings updates shop settings with support for partial updates
func (repo *RepositoryImpl) UpdateShopSettings(ctx context.Context, user *bootstrap.User, shopID string, updates request.UpdateShopSettingsRequest) error {
	err := dbutil.WithTxContext(ctx, repo.db, func(tx *sql.Tx) error {
		admin, _, err := shared.LockShopMutation(ctx, tx, shopID, user.UserID)
		if err != nil {
			return err
		}
		if !admin {
			return shared.ErrShopAdminRequired
		}

		now := time.Now()

		if updates.AdminOnlyLists == nil {
			return errors.New("no settings to update")
		}

		stmt := Shops.UPDATE(Shops.UpdatedAt, Shops.AdminOnlyLists).SET(
			Shops.UpdatedAt.SET(TimestampzT(now)),
			Shops.AdminOnlyLists.SET(Bool(*updates.AdminOnlyLists)),
		).WHERE(Shops.ID.EQ(String(shopID)))

		result, err := stmt.ExecContext(ctx, tx)
		if err != nil {
			return fmt.Errorf("failed to update shop settings: %w", err)
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("failed to get rows affected: %w", err)
		}

		if rowsAffected == 0 {
			return errors.New("shop not found")
		}

		return nil
	})
	if err != nil {
		return err
	}
	slog.Info("Shop settings updated", "shop_id", shopID, "updates", formatSettingsUpdate(updates))
	return nil
}

func formatSettingsUpdate(updates request.UpdateShopSettingsRequest) string {
	var parts []string
	if updates.AdminOnlyLists != nil {
		parts = append(parts, fmt.Sprintf("admin_only_lists=%v", *updates.AdminOnlyLists))
	}
	return fmt.Sprintf("{%s}", strings.Join(parts, ", "))
}
