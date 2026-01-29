package members

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

func (repo *RepositoryImpl) IsUserShopAdmin(user *bootstrap.User, shopID string) (bool, error) {
	stmt := SELECT(COUNT(ShopMembers.ID).AS("count")).
		FROM(ShopMembers).
		WHERE(
			ShopMembers.ShopID.EQ(String(shopID)).
				AND(ShopMembers.UserID.EQ(String(user.UserID))).
				AND(ShopMembers.Role.EQ(String("admin"))),
		)

	var result struct {
		Count int64 `sql:"primary_key"`
	}
	err := stmt.Query(repo.db, &result)
	if err != nil {
		return false, fmt.Errorf("failed to check admin status: %w", err)
	}

	return result.Count > 0, nil
}

func (repo *RepositoryImpl) IsUserMemberOfShop(user *bootstrap.User, shopID string) (bool, error) {
	stmt := SELECT(COUNT(ShopMembers.ID).AS("count")).
		FROM(ShopMembers).
		WHERE(
			ShopMembers.ShopID.EQ(String(shopID)).
				AND(ShopMembers.UserID.EQ(String(user.UserID))),
		)

	var result struct {
		Count int64 `sql:"primary_key"`
	}
	err := stmt.Query(repo.db, &result)
	if err != nil {
		return false, fmt.Errorf("failed to check membership: %w", err)
	}

	return result.Count > 0, nil
}

// Shop Member Operations
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

func (repo *RepositoryImpl) RemoveMemberFromShop(user *bootstrap.User, shopID string, targetUserID string) error {
	stmt := ShopMembers.DELETE().
		WHERE(
			ShopMembers.ShopID.EQ(String(shopID)).
				AND(ShopMembers.UserID.EQ(String(targetUserID))),
		)

	result, err := stmt.Exec(repo.db)
	if err != nil {
		return fmt.Errorf("failed to remove member from shop: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return errors.New("member not found in shop")
	}

	slog.Info("Member removed from shop", "shop_id", shopID, "removed_user_id", targetUserID, "removed_by", user.UserID)
	return nil
}

func (repo *RepositoryImpl) UpdateMemberRole(user *bootstrap.User, shopID string, targetUserID string, newRole string) error {
	stmt := ShopMembers.UPDATE(
		ShopMembers.Role,
	).SET(
		newRole,
	).WHERE(
		ShopMembers.ShopID.EQ(String(shopID)).
			AND(ShopMembers.UserID.EQ(String(targetUserID))),
	)

	result, err := stmt.Exec(repo.db)
	if err != nil {
		return fmt.Errorf("failed to update member role: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return errors.New("member not found in shop")
	}

	slog.Info("Member role updated", "shop_id", shopID, "target_user_id", targetUserID, "new_role", newRole, "updated_by", user.UserID)
	return nil
}

func (repo *RepositoryImpl) GetShopMembers(user *bootstrap.User, shopID string) ([]response.ShopMemberWithUsername, error) {
	rawSQL := `
		SELECT 
			sm.id,
			sm.shop_id,
			sm.user_id,
			sm.role,
			sm.joined_at,
			u.username
		FROM shop_members sm
		LEFT JOIN users u ON sm.user_id = u.uid
		WHERE sm.shop_id = $1
		ORDER BY sm.joined_at ASC
	`

	rows, err := repo.db.Query(rawSQL, shopID)
	if err != nil {
		return nil, fmt.Errorf("failed to get shop members: %w", err)
	}
	defer rows.Close()

	var members []response.ShopMemberWithUsername
	for rows.Next() {
		var member response.ShopMemberWithUsername
		err := rows.Scan(&member.ID, &member.ShopID, &member.UserID, &member.Role, &member.JoinedAt, &member.Username)
		if err != nil {
			return nil, fmt.Errorf("failed to scan member row: %w", err)
		}
		members = append(members, member)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return members, nil
}

func (repo *RepositoryImpl) GetShopMemberCount(user *bootstrap.User, shopID string) (int64, error) {
	stmt := SELECT(COUNT(ShopMembers.ID).AS("count")).
		FROM(ShopMembers).
		WHERE(ShopMembers.ShopID.EQ(String(shopID)))

	var result struct {
		Count int64 `sql:"primary_key"`
	}
	err := stmt.Query(repo.db, &result)
	if err != nil {
		return 0, fmt.Errorf("failed to get member count: %w", err)
	}

	return result.Count, nil
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
