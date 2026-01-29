package invites

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/shops/shared"
	"miltechserver/bootstrap"
	"strings"
	"time"

	"github.com/google/uuid"
)

type ServiceImpl struct {
	repo Repository
	auth shared.ShopAuthorization
}

func NewService(repo Repository, auth shared.ShopAuthorization) *ServiceImpl {
	return &ServiceImpl{
		repo: repo,
		auth: auth,
	}
}

func (service *ServiceImpl) GenerateInviteCode(user *bootstrap.User, shopID string) (*model.ShopInviteCodes, error) {
	if user == nil {
		return nil, errors.New("unauthorized user")
	}

	isMember, err := service.auth.IsUserMemberOfShop(user, shopID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify membership: %w", err)
	}

	if !isMember {
		return nil, errors.New("access denied: user is not a member of this shop")
	}

	code, err := generateShortCode()
	if err != nil {
		return nil, fmt.Errorf("failed to generate invite code: %w", err)
	}

	inviteCode := model.ShopInviteCodes{
		ID:        uuid.New().String(),
		ShopID:    shopID,
		Code:      code,
		CreatedBy: user.UserID,
		IsActive:  func() *bool { b := true; return &b }(),
	}

	now := time.Now()
	inviteCode.CreatedAt = &now

	createdCode, err := service.repo.CreateInviteCode(user, inviteCode)
	if err != nil {
		return nil, fmt.Errorf("failed to create invite code: %w", err)
	}

	slog.Info("Invite code generated", "user_id", user.UserID, "shop_id", shopID, "code", code)
	return createdCode, nil
}

func (service *ServiceImpl) GetInviteCodesByShop(user *bootstrap.User, shopID string) ([]model.ShopInviteCodes, error) {
	if user == nil {
		return nil, errors.New("unauthorized user")
	}

	isMember, err := service.auth.IsUserMemberOfShop(user, shopID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify membership: %w", err)
	}

	if !isMember {
		return nil, errors.New("access denied: user is not a member of this shop")
	}

	codes, err := service.repo.GetInviteCodesByShop(user, shopID)
	if err != nil {
		return nil, fmt.Errorf("failed to get invite codes: %w", err)
	}

	if codes == nil {
		return []model.ShopInviteCodes{}, nil
	}

	return codes, nil
}

func (service *ServiceImpl) DeactivateInviteCode(user *bootstrap.User, codeID string) error {
	if user == nil {
		return errors.New("unauthorized user")
	}

	inviteCode, err := service.repo.GetInviteCodeByID(codeID)
	if err != nil {
		return fmt.Errorf("failed to get invite code: %w", err)
	}

	isAdmin, err := service.auth.IsUserShopAdmin(user, inviteCode.ShopID)
	if err != nil {
		return fmt.Errorf("failed to verify admin status: %w", err)
	}

	if !isAdmin {
		return errors.New("only shop administrators can deactivate invite codes")
	}

	err = service.repo.DeactivateInviteCode(user, codeID)
	if err != nil {
		return fmt.Errorf("failed to deactivate invite code: %w", err)
	}

	slog.Info("Invite code deactivated", "user_id", user.UserID, "code_id", codeID)
	return nil
}

func (service *ServiceImpl) DeleteInviteCode(user *bootstrap.User, codeID string) error {
	if user == nil {
		return errors.New("unauthorized user")
	}

	inviteCode, err := service.repo.GetInviteCodeByID(codeID)
	if err != nil {
		return fmt.Errorf("failed to get invite code: %w", err)
	}

	isAdmin, err := service.auth.IsUserShopAdmin(user, inviteCode.ShopID)
	if err != nil {
		return fmt.Errorf("failed to verify admin status: %w", err)
	}

	if !isAdmin {
		return errors.New("only shop administrators can delete invite codes")
	}

	err = service.repo.DeleteInviteCode(user, codeID)
	if err != nil {
		return fmt.Errorf("failed to delete invite code: %w", err)
	}

	slog.Info("Invite code deleted", "user_id", user.UserID, "code_id", codeID)
	return nil
}

func generateShortCode() (string, error) {
	bytes := make([]byte, 4)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}

	code := strings.ToUpper(hex.EncodeToString(bytes))
	if len(code) > 8 {
		code = code[:8]
	}

	return code, nil
}
