package invites

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"miltechserver/.gen/miltech_ng/public/model"
	. "miltechserver/.gen/miltech_ng/public/table"
	dbutil "miltechserver/api/shared/db"
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

func (repo *RepositoryImpl) CreateInviteCode(ctx context.Context, user *bootstrap.User, inviteCode model.ShopInviteCodes) (*model.ShopInviteCodes, error) {
	if user == nil {
		return nil, errors.New("unauthorized user")
	}
	var createdCode model.ShopInviteCodes
	err := dbutil.WithTxContext(ctx, repo.db, func(tx *sql.Tx) error {
		if _, _, err := shared.LockShopMutation(ctx, tx, inviteCode.ShopID, user.UserID); err != nil {
			return err
		}
		inviteCode.CreatedBy = user.UserID
		stmt := ShopInviteCodes.INSERT(
			ShopInviteCodes.ID, ShopInviteCodes.ShopID, ShopInviteCodes.Code,
			ShopInviteCodes.CreatedBy, ShopInviteCodes.IsActive, ShopInviteCodes.CreatedAt,
		).MODEL(inviteCode).RETURNING(ShopInviteCodes.AllColumns)
		if err := stmt.QueryContext(ctx, tx, &createdCode); err != nil {
			return fmt.Errorf("failed to create invite code: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &createdCode, nil
}

func (repo *RepositoryImpl) GetInviteCodeByCode(ctx context.Context, code string) (*model.ShopInviteCodes, error) {
	stmt := SELECT(ShopInviteCodes.AllColumns).
		FROM(ShopInviteCodes).
		WHERE(ShopInviteCodes.Code.EQ(String(code)))

	var inviteCode model.ShopInviteCodes
	err := stmt.QueryContext(ctx, repo.db, &inviteCode)
	if err != nil {
		return nil, fmt.Errorf("invite code not found: %w", err)
	}

	return &inviteCode, nil
}

func (repo *RepositoryImpl) GetInviteCodeByID(ctx context.Context, codeID string) (*model.ShopInviteCodes, error) {
	stmt := SELECT(ShopInviteCodes.AllColumns).
		FROM(ShopInviteCodes).
		WHERE(ShopInviteCodes.ID.EQ(String(codeID)))

	var inviteCode model.ShopInviteCodes
	err := stmt.QueryContext(ctx, repo.db, &inviteCode)
	if err != nil {
		return nil, fmt.Errorf("invite code not found: %w", err)
	}

	return &inviteCode, nil
}

func (repo *RepositoryImpl) GetInviteCodesByShop(ctx context.Context, user *bootstrap.User, shopID string) ([]model.ShopInviteCodes, error) {
	stmt := SELECT(ShopInviteCodes.AllColumns).
		FROM(ShopInviteCodes).
		WHERE(ShopInviteCodes.ShopID.EQ(String(shopID))).
		ORDER_BY(ShopInviteCodes.CreatedAt.DESC())

	var codes []model.ShopInviteCodes
	err := stmt.QueryContext(ctx, repo.db, &codes)
	if err != nil {
		return nil, fmt.Errorf("failed to get invite codes: %w", err)
	}

	return codes, nil
}

// lockInviteMutation derives authority from the stored invite, then follows the
// same Shop/member order as admission, promotion, and removal.
func lockInviteMutation(ctx context.Context, tx *sql.Tx, user *bootstrap.User, codeID, denied string) error {
	if user == nil {
		return errors.New("unauthorized user")
	}
	var invite model.ShopInviteCodes
	resolve := SELECT(ShopInviteCodes.AllColumns).FROM(ShopInviteCodes).WHERE(ShopInviteCodes.ID.EQ(String(codeID)))
	if err := resolve.QueryContext(ctx, tx, &invite); err != nil {
		return fmt.Errorf("invite code not found: %w", err)
	}
	actualInviteShop := invite.ShopID
	admin, _, err := shared.LockShopMutation(ctx, tx, actualInviteShop, user.UserID)
	if err != nil {
		return err
	}
	if !admin {
		return errors.New(denied)
	}
	// A concurrent delete may have committed while the Shop lock was awaited.
	recheck := SELECT(ShopInviteCodes.AllColumns).FROM(ShopInviteCodes).
		WHERE(ShopInviteCodes.ID.EQ(String(codeID)).AND(ShopInviteCodes.ShopID.EQ(String(actualInviteShop))))
	if err := recheck.QueryContext(ctx, tx, &invite); err != nil {
		return fmt.Errorf("invite code not found: %w", err)
	}
	return nil
}

func (repo *RepositoryImpl) DeactivateInviteCode(ctx context.Context, user *bootstrap.User, codeID string) error {
	return dbutil.WithTxContext(ctx, repo.db, func(tx *sql.Tx) error {
		if err := lockInviteMutation(ctx, tx, user, codeID, "only shop administrators can deactivate invite codes"); err != nil {
			return err
		}
		stmt := ShopInviteCodes.UPDATE(ShopInviteCodes.IsActive).
			SET(ShopInviteCodes.IsActive.SET(Bool(false))).WHERE(ShopInviteCodes.ID.EQ(String(codeID)))
		if _, err := stmt.ExecContext(ctx, tx); err != nil {
			return fmt.Errorf("failed to deactivate invite code: %w", err)
		}
		return nil
	})
}

func (repo *RepositoryImpl) DeleteInviteCode(ctx context.Context, user *bootstrap.User, codeID string) error {
	err := dbutil.WithTxContext(ctx, repo.db, func(tx *sql.Tx) error {
		if err := lockInviteMutation(ctx, tx, user, codeID, "only shop administrators can delete invite codes"); err != nil {
			return err
		}
		stmt := ShopInviteCodes.DELETE().WHERE(ShopInviteCodes.ID.EQ(String(codeID)))
		if _, err := stmt.ExecContext(ctx, tx); err != nil {
			return fmt.Errorf("failed to delete invite code: %w", err)
		}
		return nil
	})
	if err != nil {
		return err
	}
	slog.Info("Invite code deleted from database", "code_id", codeID, "deleted_by", user.UserID)
	return nil
}
