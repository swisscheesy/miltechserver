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
	"miltechserver/bootstrap"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/container"
	. "github.com/go-jet/jet/v2/postgres"
)

const shopMessageImagesContainer = "shop-message-images"

type RepositoryImpl struct {
	db         *sql.DB
	blobClient *azblob.Client
	env        *bootstrap.Env
}

func NewRepository(db *sql.DB, blobClient *azblob.Client, env *bootstrap.Env) *RepositoryImpl {
	return &RepositoryImpl{
		db:         db,
		blobClient: blobClient,
		env:        env,
	}
}

func (repo *RepositoryImpl) CreateShop(user *bootstrap.User, shop model.Shops) (*model.Shops, error) {
	stmt := Shops.INSERT(
		Shops.ID,
		Shops.Name,
		Shops.Details,
		Shops.CreatedBy,
		Shops.CreatedAt,
		Shops.UpdatedAt,
	).MODEL(shop).RETURNING(Shops.AllColumns)

	var createdShop model.Shops
	err := stmt.Query(repo.db, &createdShop)
	if err != nil {
		return nil, fmt.Errorf("failed to create shop: %w", err)
	}

	slog.Info("Shop created in database", "shop_id", shop.ID, "created_by", user.UserID)
	return &createdShop, nil
}

func (repo *RepositoryImpl) UpdateShop(user *bootstrap.User, shop model.Shops) (*model.Shops, error) {
	now := time.Now()
	shop.UpdatedAt = &now

	updateStmt := Shops.UPDATE(
		Shops.Name,
		Shops.UpdatedAt,
	).SET(
		Shops.Name.SET(String(shop.Name)),
		Shops.UpdatedAt.SET(TimestampzT(*shop.UpdatedAt)),
		Shops.AdminOnlyLists.SET(Bool(shop.AdminOnlyLists)),
	)

	if shop.Details != nil {
		updateStmt = Shops.UPDATE(
			Shops.Name,
			Shops.Details,
			Shops.UpdatedAt,
		).SET(
			Shops.Name.SET(String(shop.Name)),
			Shops.Details.SET(String(*shop.Details)),
			Shops.UpdatedAt.SET(TimestampzT(*shop.UpdatedAt)),
			Shops.AdminOnlyLists.SET(Bool(shop.AdminOnlyLists)),
		)
	}

	stmt := updateStmt.WHERE(
		Shops.ID.EQ(String(shop.ID)).
			AND(Shops.CreatedBy.EQ(String(user.UserID))),
	).RETURNING(Shops.AllColumns)

	var updatedShop model.Shops
	err := stmt.Query(repo.db, &updatedShop)
	if err != nil {
		return nil, fmt.Errorf("failed to update shop: %w", err)
	}

	slog.Info("Shop updated in database", "shop_id", shop.ID, "updated_by", user.UserID)
	return &updatedShop, nil
}

func (repo *RepositoryImpl) DeleteShop(user *bootstrap.User, shopID string) error {
	stmt := Shops.DELETE().WHERE(
		Shops.ID.EQ(String(shopID)).
			AND(Shops.CreatedBy.EQ(String(user.UserID))),
	)

	result, err := stmt.Exec(repo.db)
	if err != nil {
		return fmt.Errorf("failed to delete shop: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return errors.New("shop not found or user not authorized to delete")
	}

	slog.Info("Shop deleted from database", "shop_id", shopID, "deleted_by", user.UserID)
	return nil
}

func (repo *RepositoryImpl) GetShopsByUser(user *bootstrap.User) ([]model.Shops, error) {
	stmt := SELECT(Shops.AllColumns).
		FROM(
			Shops.
				INNER_JOIN(ShopMembers, ShopMembers.ShopID.EQ(Shops.ID)),
		).
		WHERE(ShopMembers.UserID.EQ(String(user.UserID))).
		ORDER_BY(Shops.CreatedAt.DESC())

	var shops []model.Shops
	err := stmt.Query(repo.db, &shops)
	if err != nil {
		return nil, fmt.Errorf("failed to get shops for user: %w", err)
	}

	slog.Info("Shops retrieved for user", "user_id", user.UserID, "count", len(shops))
	return shops, nil
}

func (repo *RepositoryImpl) GetShopByID(user *bootstrap.User, shopID string) (*response.ShopDetailResponse, error) {
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
	err := repo.db.QueryRow(rawSQL, shopID, user.UserID).Scan(
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

func (repo *RepositoryImpl) GetShopsWithStatsForUser(user *bootstrap.User) ([]response.ShopWithStats, error) {
	rawSQL := `
		SELECT
			s.id,
			s.name,
			s.details,
			s.created_by,
			s.created_at,
			s.updated_at,
			s.admin_only_lists,
			COALESCE(member_stats.member_count, 0) as member_count,
			COALESCE(vehicle_stats.vehicle_count, 0) as vehicle_count,
			CASE WHEN admin_check.user_id IS NOT NULL THEN true ELSE false END as is_admin
		FROM shops s
		INNER JOIN shop_members sm ON s.id = sm.shop_id
		LEFT JOIN (
			SELECT shop_id, COUNT(*) as member_count
			FROM shop_members
			GROUP BY shop_id
		) member_stats ON s.id = member_stats.shop_id
		LEFT JOIN (
			SELECT shop_id, COUNT(*) as vehicle_count
			FROM shop_vehicle
			GROUP BY shop_id
		) vehicle_stats ON s.id = vehicle_stats.shop_id
		LEFT JOIN (
			SELECT shop_id, user_id
			FROM shop_members
			WHERE role = 'admin'
		) admin_check ON s.id = admin_check.shop_id AND admin_check.user_id = $1
		WHERE sm.user_id = $1
		ORDER BY s.created_at DESC
	`

	rows, err := repo.db.Query(rawSQL, user.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get shops with stats: %w", err)
	}
	defer rows.Close()

	var results []response.ShopWithStats
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

func (repo *RepositoryImpl) AddMemberToShop(user *bootstrap.User, shopID string, role string) error {
	curTime := time.Now().UTC()
	member := model.ShopMembers{
		ID:       fmt.Sprintf("%s_%s", shopID, user.UserID),
		ShopID:   shopID,
		UserID:   user.UserID,
		Role:     role,
		JoinedAt: &curTime,
	}

	stmt := ShopMembers.INSERT(
		ShopMembers.ID,
		ShopMembers.ShopID,
		ShopMembers.UserID,
		ShopMembers.Role,
		ShopMembers.JoinedAt,
	).MODEL(member).
		ON_CONFLICT(ShopMembers.ShopID, ShopMembers.UserID).
		DO_UPDATE(SET(ShopMembers.Role.SET(String(role))))

	_, err := stmt.Exec(repo.db)
	if err != nil {
		return fmt.Errorf("failed to add member to shop: %w", err)
	}

	slog.Info("Member added to shop", "shop_id", shopID, "user_id", user.UserID, "role", role)
	return nil
}

func (repo *RepositoryImpl) DeleteShopMessageBlobs(shopID string) error {
	ctx := context.Background()

	containerClient := repo.blobClient.ServiceClient().NewContainerClient(shopMessageImagesContainer)

	prefix := fmt.Sprintf("%s/", shopID)
	pager := containerClient.NewListBlobsFlatPager(&container.ListBlobsFlatOptions{
		Prefix: &prefix,
	})

	deletedCount := 0
	errorCount := 0

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			slog.Warn("Failed to list blobs for shop deletion", "shop_id", shopID, "error", err)
			continue
		}

		for _, blob := range page.Segment.BlobItems {
			if blob.Name == nil {
				continue
			}

			_, err := repo.blobClient.DeleteBlob(ctx, shopMessageImagesContainer, *blob.Name, nil)
			if err != nil {
				slog.Warn("Failed to delete shop message blob",
					"shop_id", shopID,
					"blob_name", *blob.Name,
					"error", err)
				errorCount++
			} else {
				deletedCount++
			}
		}
	}

	slog.Info("Shop message blobs cleanup completed",
		"shop_id", shopID,
		"deleted_count", deletedCount,
		"error_count", errorCount)

	return nil
}
