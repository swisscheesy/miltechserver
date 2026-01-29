package invites

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

func (repo *RepositoryImpl) CreateInviteCode(user *bootstrap.User, inviteCode model.ShopInviteCodes) (*model.ShopInviteCodes, error) {
	stmt := ShopInviteCodes.INSERT(
		ShopInviteCodes.ID,
		ShopInviteCodes.ShopID,
		ShopInviteCodes.Code,
		ShopInviteCodes.CreatedBy,
		ShopInviteCodes.IsActive,
		ShopInviteCodes.CreatedAt,
	).MODEL(inviteCode).RETURNING(ShopInviteCodes.AllColumns)

	var createdCode model.ShopInviteCodes
	err := stmt.Query(repo.db, &createdCode)
	if err != nil {
		return nil, fmt.Errorf("failed to create invite code: %w", err)
	}

	return &createdCode, nil
}

func (repo *RepositoryImpl) GetInviteCodeByCode(code string) (*model.ShopInviteCodes, error) {
	stmt := SELECT(ShopInviteCodes.AllColumns).
		FROM(ShopInviteCodes).
		WHERE(ShopInviteCodes.Code.EQ(String(code)))

	var inviteCode model.ShopInviteCodes
	err := stmt.Query(repo.db, &inviteCode)
	if err != nil {
		return nil, fmt.Errorf("invite code not found: %w", err)
	}

	return &inviteCode, nil
}

func (repo *RepositoryImpl) GetInviteCodeByID(codeID string) (*model.ShopInviteCodes, error) {
	stmt := SELECT(ShopInviteCodes.AllColumns).
		FROM(ShopInviteCodes).
		WHERE(ShopInviteCodes.ID.EQ(String(codeID)))

	var inviteCode model.ShopInviteCodes
	err := stmt.Query(repo.db, &inviteCode)
	if err != nil {
		return nil, fmt.Errorf("invite code not found: %w", err)
	}

	return &inviteCode, nil
}

func (repo *RepositoryImpl) GetInviteCodesByShop(user *bootstrap.User, shopID string) ([]model.ShopInviteCodes, error) {
	stmt := SELECT(ShopInviteCodes.AllColumns).
		FROM(ShopInviteCodes).
		WHERE(ShopInviteCodes.ShopID.EQ(String(shopID))).
		ORDER_BY(ShopInviteCodes.CreatedAt.DESC())

	var codes []model.ShopInviteCodes
	err := stmt.Query(repo.db, &codes)
	if err != nil {
		return nil, fmt.Errorf("failed to get invite codes: %w", err)
	}

	return codes, nil
}

func (repo *RepositoryImpl) DeactivateInviteCode(user *bootstrap.User, codeID string) error {
	stmt := ShopInviteCodes.UPDATE(
		ShopInviteCodes.IsActive,
	).SET(
		ShopInviteCodes.IsActive.SET(Bool(false)),
	).WHERE(ShopInviteCodes.ID.EQ(String(codeID)))

	result, err := stmt.Exec(repo.db)
	if err != nil {
		return fmt.Errorf("failed to deactivate invite code: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return errors.New("invite code not found")
	}

	return nil
}

func (repo *RepositoryImpl) DeleteInviteCode(user *bootstrap.User, codeID string) error {
	stmt := ShopInviteCodes.DELETE().WHERE(ShopInviteCodes.ID.EQ(String(codeID)))

	result, err := stmt.Exec(repo.db)
	if err != nil {
		return fmt.Errorf("failed to delete invite code: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return errors.New("invite code not found")
	}

	slog.Info("Invite code deleted from database", "code_id", codeID, "deleted_by", user.UserID)
	return nil
}
