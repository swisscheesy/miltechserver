package core

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"miltechserver/.gen/miltech_ng/public/model"
	. "miltechserver/.gen/miltech_ng/public/table"
	"miltechserver/api/response"
	dbutil "miltechserver/api/shared/db"
	"miltechserver/api/shops/messages"
	"miltechserver/api/shops/shared"
	"miltechserver/bootstrap"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	. "github.com/go-jet/jet/v2/postgres"
)

type RepositoryImpl struct {
	db         *sql.DB
	blobClient *azblob.Client
	env        *bootstrap.Env
	assets     messages.AssetRepository
}

func NewRepository(db *sql.DB, blobClient *azblob.Client, env *bootstrap.Env) *RepositoryImpl {
	storage := messages.AssetStorage{Container: "shop-message-images"}
	if env != nil {
		storage.Account = env.BlobAccountName
	}
	return &RepositoryImpl{
		db:         db,
		blobClient: blobClient,
		env:        env,
		assets:     messages.NewAssetRepository(db, storage),
	}
}

func (repo *RepositoryImpl) CreateShop(ctx context.Context, user *bootstrap.User, shop model.Shops) (*model.Shops, error) {
	var createdShop model.Shops
	err := dbutil.WithTxContext(ctx, repo.db, func(tx *sql.Tx) error {
		stmt := Shops.INSERT(
			Shops.ID, Shops.Name, Shops.Details, Shops.CreatedBy,
			Shops.CreatedAt, Shops.UpdatedAt, Shops.AdminOnlyLists,
		).MODEL(shop).RETURNING(Shops.AllColumns)
		if err := stmt.QueryContext(ctx, tx, &createdShop); err != nil {
			return fmt.Errorf("failed to create shop: %w", err)
		}
		now := time.Now().UTC()
		member := model.ShopMembers{
			ID:     fmt.Sprintf("%s_%s", shop.ID, user.UserID),
			ShopID: shop.ID, UserID: user.UserID, Role: "admin", JoinedAt: &now,
		}
		memberStmt := ShopMembers.INSERT(
			ShopMembers.ID, ShopMembers.ShopID, ShopMembers.UserID,
			ShopMembers.Role, ShopMembers.JoinedAt,
		).MODEL(member)
		if _, err := memberStmt.ExecContext(ctx, tx); err != nil {
			return fmt.Errorf("failed to add creator as admin to shop: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	slog.Info("Shop created in database", "shop_id", shop.ID, "created_by", user.UserID)
	return &createdShop, nil
}

func (repo *RepositoryImpl) UpdateShop(ctx context.Context, user *bootstrap.User, shop model.Shops) (*model.Shops, error) {
	var updatedShop model.Shops
	err := dbutil.WithTxContext(ctx, repo.db, func(tx *sql.Tx) error {
		admin, _, err := shared.LockShopMutation(ctx, tx, shop.ID, user.UserID)
		if err != nil {
			return err
		}
		if !admin {
			return shared.ErrShopAdminRequired
		}
		now := time.Now()
		updateStmt := Shops.UPDATE(Shops.Name, Shops.UpdatedAt).SET(
			Shops.Name.SET(String(shop.Name)), Shops.UpdatedAt.SET(TimestampzT(now)),
		)
		// Omitted/null details preserve the existing value, as in the released contract.
		if shop.Details != nil {
			updateStmt = Shops.UPDATE(Shops.Name, Shops.Details, Shops.UpdatedAt).SET(
				Shops.Name.SET(String(shop.Name)), Shops.Details.SET(String(*shop.Details)), Shops.UpdatedAt.SET(TimestampzT(now)),
			)
		}
		stmt := updateStmt.WHERE(Shops.ID.EQ(String(shop.ID))).RETURNING(Shops.AllColumns)
		if err := stmt.QueryContext(ctx, tx, &updatedShop); err != nil {
			return fmt.Errorf("failed to update shop: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	slog.Info("Shop updated in database", "shop_id", shop.ID, "updated_by", user.UserID)
	return &updatedShop, nil
}

func (repo *RepositoryImpl) DeleteShop(ctx context.Context, user *bootstrap.User, shopID string) error {
	if user == nil {
		return errors.New("unauthorized user")
	}
	err := dbutil.WithTxContext(ctx, repo.db, func(tx *sql.Tx) error {
		if _, _, err := shared.LockShopMutation(ctx, tx, shopID, user.UserID); err != nil {
			return err
		}
		// Current membership and original creation, not current admin role, grant deletion.
		if err := repo.assets.EnqueueShopCleanup(ctx, tx, shopID); err != nil {
			return err
		}
		stmt := Shops.DELETE().WHERE(Shops.ID.EQ(String(shopID)).AND(Shops.CreatedBy.EQ(String(user.UserID))))
		result, err := stmt.ExecContext(ctx, tx)
		if err != nil {
			return fmt.Errorf("failed to delete shop: %w", err)
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("failed to get rows affected: %w", err)
		}
		if affected != 1 {
			return shared.ErrShopCreatorOnly
		}
		return nil
	})
	if err != nil {
		return err
	}
	slog.Info("Shop deleted from database", "shop_id", shopID, "deleted_by", user.UserID)
	return nil
}

func (repo *RepositoryImpl) GetShopsByUser(ctx context.Context, user *bootstrap.User) ([]model.Shops, error) {
	stmt := SELECT(Shops.AllColumns).
		FROM(
			Shops.
				INNER_JOIN(ShopMembers, ShopMembers.ShopID.EQ(Shops.ID)),
		).
		WHERE(ShopMembers.UserID.EQ(String(user.UserID))).
		ORDER_BY(Shops.CreatedAt.DESC())

	var shops []model.Shops
	err := stmt.QueryContext(ctx, repo.db, &shops)
	if err != nil {
		return nil, fmt.Errorf("failed to get shops for user: %w", err)
	}

	slog.Info("Shops retrieved for user", "user_id", user.UserID, "count", len(shops))
	return shops, nil
}

func (repo *RepositoryImpl) GetShopEquipmentOverview(
	ctx context.Context,
	user *bootstrap.User,
) ([]response.ShopEquipmentOverview, error) {
	queryStarted := time.Now()
	stmt := SELECT(
		Shops.ID, Shops.Name, Shops.Details, ShopMembers.Role,
		ShopVehicle.ID, ShopVehicle.Admin, ShopVehicle.Model,
		ShopVehicle.Serial, ShopVehicle.Niin,
	).FROM(
		Shops.
			INNER_JOIN(ShopMembers, ShopMembers.ShopID.EQ(Shops.ID)).
			LEFT_JOIN(ShopVehicle, ShopVehicle.ShopID.EQ(Shops.ID)),
	).WHERE(
		ShopMembers.UserID.EQ(String(user.UserID)),
	).ORDER_BY(
		Shops.CreatedAt.DESC(),
		ShopVehicle.SaveTime.DESC(),
		ShopVehicle.ID.DESC(),
	)

	shops := make([]response.ShopEquipmentOverview, 0)
	if err := stmt.QueryContext(ctx, repo.db, &shops); err != nil {
		return nil, fmt.Errorf("failed to query shop equipment overview: %w", err)
	}

	equipmentCount := 0
	for i := range shops {
		equipmentCount += len(shops[i].Equipment)
	}
	slog.Info("Shop equipment overview query completed",
		"user_id", user.UserID,
		"shop_count", len(shops),
		"equipment_count", equipmentCount,
		"duration_ms", time.Since(queryStarted).Milliseconds(),
	)
	return shops, nil
}

func (repo *RepositoryImpl) GetShopByID(ctx context.Context, user *bootstrap.User, shopID string) (*response.ShopDetailResponse, error) {
	rawSQL := `
		SELECT
			s.id,
			s.name,
			s.details,
			s.created_by,
			s.created_at,
			s.updated_at,
			s.admin_only_lists,
			COALESCE(message_stats.message_count, 0) as total_messages,
			COALESCE(member_stats.member_count, 0) as member_count,
			COALESCE(vehicle_stats.vehicle_count, 0) as vehicle_count,
			CASE WHEN admin_check.user_id IS NOT NULL THEN true ELSE false END as is_admin
		FROM shops s
		LEFT JOIN (
			SELECT shop_id, COUNT(*) as message_count
			FROM shop_messages
			WHERE shop_id = $1
			GROUP BY shop_id
		) message_stats ON s.id = message_stats.shop_id
		LEFT JOIN (
			SELECT shop_id, COUNT(*) as member_count
			FROM shop_members
			WHERE shop_id = $1
			GROUP BY shop_id
		) member_stats ON s.id = member_stats.shop_id
		LEFT JOIN (
			SELECT shop_id, COUNT(*) as vehicle_count
			FROM shop_vehicle
			WHERE shop_id = $1
			GROUP BY shop_id
		) vehicle_stats ON s.id = vehicle_stats.shop_id
		LEFT JOIN (
			SELECT shop_id, user_id
			FROM shop_members
			WHERE shop_id = $1 AND user_id = $2 AND role = 'admin'
		) admin_check ON s.id = admin_check.shop_id
		WHERE s.id = $1
	`

	var result response.ShopDetailResponse
	err := repo.db.QueryRowContext(ctx, rawSQL, shopID, user.UserID).Scan(
		&result.ID,
		&result.Name,
		&result.Details,
		&result.CreatedBy,
		&result.CreatedAt,
		&result.UpdatedAt,
		&result.AdminOnlyLists,
		&result.TotalMessages,
		&result.MemberCount,
		&result.VehicleCount,
		&result.IsAdmin,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get shop with stats: %w", err)
	}

	return &result, nil
}

func (repo *RepositoryImpl) GetShopsWithStatsForUser(ctx context.Context, user *bootstrap.User) ([]response.ShopWithStats, error) {
	rawSQL := `
		WITH user_shops AS (
			SELECT
				s.id,
				s.name,
				s.details,
				s.created_by,
				s.created_at,
				s.updated_at,
				s.admin_only_lists,
				sm.role
			FROM shops s
			INNER JOIN shop_members sm ON s.id = sm.shop_id
			WHERE sm.user_id = $1
		),
		member_stats AS (
			SELECT sm.shop_id, COUNT(*) AS member_count
			FROM shop_members sm
			INNER JOIN user_shops us ON us.id = sm.shop_id
			GROUP BY sm.shop_id
		),
		vehicle_stats AS (
			SELECT sv.shop_id, COUNT(*) AS vehicle_count
			FROM shop_vehicle sv
			INNER JOIN user_shops us ON us.id = sv.shop_id
			GROUP BY sv.shop_id
		)
		SELECT
			us.id,
			us.name,
			us.details,
			us.created_by,
			us.created_at,
			us.updated_at,
			us.admin_only_lists,
			COALESCE(member_stats.member_count, 0) AS member_count,
			COALESCE(vehicle_stats.vehicle_count, 0) AS vehicle_count,
			(us.role = 'admin') AS is_admin
		FROM user_shops us
		LEFT JOIN member_stats ON us.id = member_stats.shop_id
		LEFT JOIN vehicle_stats ON us.id = vehicle_stats.shop_id
		ORDER BY us.created_at DESC
	`

	rows, err := repo.db.QueryContext(ctx, rawSQL, user.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get shops with stats: %w", err)
	}
	defer rows.Close()

	results := make([]response.ShopWithStats, 0, 16)
	for rows.Next() {
		var shop model.Shops
		var memberCount, vehicleCount int64
		var isAdmin bool

		err := rows.Scan(
			&shop.ID,
			&shop.Name,
			&shop.Details,
			&shop.CreatedBy,
			&shop.CreatedAt,
			&shop.UpdatedAt,
			&shop.AdminOnlyLists,
			&memberCount,
			&vehicleCount,
			&isAdmin,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan shop row: %w", err)
		}

		results = append(results, response.ShopWithStats{
			Shop:             shop,
			MemberCount:      memberCount,
			VehicleCount:     vehicleCount,
			IsAdmin:          isAdmin,
			IsListsAdminOnly: shop.AdminOnlyLists,
		})
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	slog.Info("Shops with stats retrieved for user", "user_id", user.UserID, "count", len(results))
	return results, nil
}
