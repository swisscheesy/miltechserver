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

func (repo *RepositoryImpl) IsUserShopAdmin(ctx context.Context, user *bootstrap.User, shopID string) (bool, error) {
	stmt := SELECT(Int(1).AS("exists")).
		FROM(ShopMembers).
		WHERE(
			ShopMembers.ShopID.EQ(String(shopID)).
				AND(ShopMembers.UserID.EQ(String(user.UserID))).
				AND(ShopMembers.Role.EQ(String("admin"))),
		).
		LIMIT(1)

	var result []struct {
		Exists int `sql:"exists"`
	}
	err := stmt.QueryContext(ctx, repo.db, &result)
	if err != nil {
		return false, fmt.Errorf("failed to check admin status: %w", err)
	}

	return len(result) > 0, nil
}

func (repo *RepositoryImpl) IsUserMemberOfShop(ctx context.Context, user *bootstrap.User, shopID string) (bool, error) {
	stmt := SELECT(Int(1).AS("exists")).
		FROM(ShopMembers).
		WHERE(
			ShopMembers.ShopID.EQ(String(shopID)).
				AND(ShopMembers.UserID.EQ(String(user.UserID))),
		).
		LIMIT(1)

	var result []struct {
		Exists int `sql:"exists"`
	}
	err := stmt.QueryContext(ctx, repo.db, &result)
	if err != nil {
		return false, fmt.Errorf("failed to check membership: %w", err)
	}

	return len(result) > 0, nil
}

// Shop Member Operations
func (repo *RepositoryImpl) JoinViaInvite(ctx context.Context, user *bootstrap.User, code string) error {
	if user == nil {
		return errors.New("unauthorized user")
	}
	var admittedShopID string
	err := dbutil.WithTxContext(ctx, repo.db, func(tx *sql.Tx) error {
		var invite model.ShopInviteCodes
		resolve := SELECT(ShopInviteCodes.AllColumns).FROM(ShopInviteCodes).
			WHERE(ShopInviteCodes.Code.EQ(String(code)))
		if err := resolve.QueryContext(ctx, tx, &invite); err != nil {
			return fmt.Errorf("invalid invite code: %w", err)
		}
		// Admission has no member yet. Lock the persisted Shop before checking
		// the invite again so revocation and membership changes share one order.
		var shop model.Shops
		lockShop := SELECT(Shops.ID).FROM(Shops).WHERE(Shops.ID.EQ(String(invite.ShopID))).FOR(UPDATE())
		if err := lockShop.QueryContext(ctx, tx, &shop); err != nil {
			return fmt.Errorf("invalid invite code: %w", err)
		}
		recheck := SELECT(ShopInviteCodes.AllColumns).FROM(ShopInviteCodes).
			WHERE(ShopInviteCodes.ID.EQ(String(invite.ID)).AND(ShopInviteCodes.ShopID.EQ(String(shop.ID))).AND(ShopInviteCodes.Code.EQ(String(code))))
		if err := recheck.QueryContext(ctx, tx, &invite); err != nil {
			return fmt.Errorf("invalid invite code: %w", err)
		}
		if invite.IsActive != nil && !*invite.IsActive {
			return errors.New("invite code is inactive")
		}
		var existing []model.ShopMembers
		memberQuery := SELECT(ShopMembers.ID).FROM(ShopMembers).
			WHERE(ShopMembers.ShopID.EQ(String(shop.ID)).AND(ShopMembers.UserID.EQ(String(user.UserID))))
		if err := memberQuery.QueryContext(ctx, tx, &existing); err != nil {
			return fmt.Errorf("failed to check membership: %w", err)
		}
		if len(existing) != 0 {
			return errors.New("user is already a member of this shop")
		}
		now := time.Now().UTC()
		member := model.ShopMembers{ID: fmt.Sprintf("%s_%s", shop.ID, user.UserID), ShopID: shop.ID, UserID: user.UserID, Role: "member", JoinedAt: &now}
		stmt := ShopMembers.INSERT(ShopMembers.ID, ShopMembers.ShopID, ShopMembers.UserID, ShopMembers.Role, ShopMembers.JoinedAt).
			MODEL(member).ON_CONFLICT(ShopMembers.ShopID, ShopMembers.UserID).DO_NOTHING()
		result, err := stmt.ExecContext(ctx, tx)
		if err != nil {
			return fmt.Errorf("failed to add member to shop: %w", err)
		}
		rows, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("failed to get rows affected: %w", err)
		}
		if rows == 0 {
			return errors.New("user is already a member of this shop")
		}
		admittedShopID = shop.ID
		return nil
	})
	if err != nil {
		return err
	}
	slog.Info("shop_joined", "shop_id", admittedShopID, "outcome", "success")
	return nil
}

func (repo *RepositoryImpl) LeaveShop(ctx context.Context, user *bootstrap.User, shopID string) error {
	if user == nil {
		return errors.New("unauthorized user")
	}
	return repo.departMember(ctx, user, shopID, user.UserID, false)
}

func (repo *RepositoryImpl) RemoveMemberFromShop(ctx context.Context, user *bootstrap.User, shopID string, targetUserID string) error {
	if user == nil {
		return errors.New("unauthorized user")
	}
	if user.UserID == targetUserID {
		return errors.New("use leave shop endpoint to remove yourself")
	}
	return repo.departMember(ctx, user, shopID, targetUserID, true)
}

func (repo *RepositoryImpl) departMember(ctx context.Context, user *bootstrap.User, shopID, targetUserID string, requiresAdmin bool) error {
	err := dbutil.WithTxContext(ctx, repo.db, func(tx *sql.Tx) error {
		admin, _, err := shared.LockShopMutation(ctx, tx, shopID, user.UserID)
		if err != nil {
			return err
		}
		if requiresAdmin && !admin {
			return shared.ErrShopAdminRequired
		}
		targetAdmin, err := shared.RequireShopMember(ctx, tx, shopID, targetUserID)
		if err != nil {
			return err
		}
		count, err := shopMemberCount(ctx, tx, shopID, false)
		if err != nil {
			return err
		}
		if count == 1 {
			return repo.deleteFinalMemberShop(ctx, tx, shopID, targetUserID)
		}
		if targetAdmin {
			admins, err := shopMemberCount(ctx, tx, shopID, true)
			if err != nil {
				return err
			}
			if admins == 1 {
				return &shared.Failure{Code: "conflict", Status: 409, PublicMessage: "promote another member to admin before leaving the shop"}
			}
		}
		stmt := ShopMembers.DELETE().WHERE(ShopMembers.ShopID.EQ(String(shopID)).AND(ShopMembers.UserID.EQ(String(targetUserID))))
		result, err := stmt.ExecContext(ctx, tx)
		if err != nil {
			return fmt.Errorf("failed to remove member from shop: %w", err)
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("failed to get rows affected: %w", err)
		}
		if affected != 1 {
			return shared.ErrMemberNotFound
		}
		return nil
	})
	if err != nil {
		return err
	}
	slog.Info("Member departed shop", "shop_id", shopID, "removed_user_id", targetUserID, "removed_by", user.UserID)
	return nil
}

// The Shop lock serializes admission, role changes and departure before these counts.
func shopMemberCount(ctx context.Context, tx *sql.Tx, shopID string, adminsOnly bool) (int64, error) {
	condition := ShopMembers.ShopID.EQ(String(shopID))
	if adminsOnly {
		condition = condition.AND(ShopMembers.Role.EQ(String("admin")))
	}
	stmt := SELECT(COUNT(ShopMembers.ID).AS("count")).FROM(ShopMembers).WHERE(condition)
	var result struct{ Count int64 }
	if err := stmt.QueryContext(ctx, tx, &result); err != nil {
		return 0, fmt.Errorf("failed to get member count: %w", err)
	}
	return result.Count, nil
}

// Final-member cleanup is independent of original ownership. Recheck the exact
// current member under the caller's Shop lock before deleting the aggregate.
func (repo *RepositoryImpl) deleteFinalMemberShop(ctx context.Context, tx *sql.Tx, shopID, userID string) error {
	if _, err := shared.RequireShopMember(ctx, tx, shopID, userID); err != nil {
		return err
	}
	count, err := shopMemberCount(ctx, tx, shopID, false)
	if err != nil {
		return err
	}
	if count != 1 {
		return &shared.Failure{Code: "conflict", Status: 409, PublicMessage: "shop membership changed; retry leaving the shop"}
	}
	if err := repo.assets.EnqueueShopCleanup(ctx, tx, shopID); err != nil {
		return err
	}
	result, err := Shops.DELETE().WHERE(Shops.ID.EQ(String(shopID))).ExecContext(ctx, tx)
	if err != nil {
		return fmt.Errorf("failed to delete shop: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if affected != 1 {
		return shared.ErrShopNotFound
	}
	return nil
}

func (repo *RepositoryImpl) UpdateMemberRole(ctx context.Context, user *bootstrap.User, shopID string, targetUserID string, newRole string) error {
	tx, err := repo.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return err
	}
	defer tx.Rollback()
	admin, _, err := shared.LockShopMutation(ctx, tx, shopID, user.UserID)
	if err != nil {
		return err
	}
	if !admin {
		return shared.ErrShopAdminRequired
	}
	if _, err := shared.RequireShopMember(ctx, tx, shopID, targetUserID); err != nil {
		return err
	}

	stmt := ShopMembers.UPDATE(
		ShopMembers.Role,
	).SET(
		newRole,
	).WHERE(
		ShopMembers.ShopID.EQ(String(shopID)).
			AND(ShopMembers.UserID.EQ(String(targetUserID))),
	)

	result, err := stmt.ExecContext(ctx, tx)
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
	return tx.Commit()
}

func (repo *RepositoryImpl) GetShopMembers(ctx context.Context, user *bootstrap.User, shopID string) ([]response.ShopMemberWithUsername, error) {
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

	rows, err := repo.db.QueryContext(ctx, rawSQL, shopID)
	if err != nil {
		return nil, fmt.Errorf("failed to get shop members: %w", err)
	}
	defer rows.Close()

	members := make([]response.ShopMemberWithUsername, 0, 16)
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
