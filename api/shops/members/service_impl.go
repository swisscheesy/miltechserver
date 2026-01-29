package members

import (
	"errors"
	"fmt"
	"log/slog"
	"miltechserver/api/response"
	"miltechserver/api/shops/members/invites"
	"miltechserver/api/shops/shared"
	"miltechserver/bootstrap"
)

type ServiceImpl struct {
	repo         Repository
	inviteRepo   invites.Repository
	auth         shared.ShopAuthorization
}

func NewService(repo Repository, inviteRepo invites.Repository, auth shared.ShopAuthorization) *ServiceImpl {
	return &ServiceImpl{
		repo:       repo,
		inviteRepo: inviteRepo,
		auth:       auth,
	}
}

func (service *ServiceImpl) JoinShopViaInviteCode(user *bootstrap.User, inviteCode string) error {
	if user == nil {
		return errors.New("unauthorized user")
	}

	code, err := service.inviteRepo.GetInviteCodeByCode(inviteCode)
	if err != nil {
		return fmt.Errorf("invalid invite code: %w", err)
	}

	if code.IsActive != nil && !*code.IsActive {
		return errors.New("invite code is inactive")
	}

	isMember, err := service.auth.IsUserMemberOfShop(user, code.ShopID)
	if err != nil {
		return fmt.Errorf("failed to check membership: %w", err)
	}

	if isMember {
		return errors.New("user is already a member of this shop")
	}

	err = service.repo.AddMemberToShop(user, code.ShopID, "member")
	if err != nil {
		return fmt.Errorf("failed to add member to shop: %w", err)
	}

	slog.Info("User joined shop via invite code", "user_id", user.UserID, "shop_id", code.ShopID, "invite_code", inviteCode)
	return nil
}

func (service *ServiceImpl) LeaveShop(user *bootstrap.User, shopID string) error {
	if user == nil {
		return errors.New("unauthorized user")
	}

	isMember, err := service.auth.IsUserMemberOfShop(user, shopID)
	if err != nil {
		return fmt.Errorf("failed to verify membership: %w", err)
	}

	if !isMember {
		return errors.New("user is not a member of this shop")
	}

	memberCount, err := service.repo.GetShopMemberCount(user, shopID)
	if err != nil {
		return fmt.Errorf("failed to get member count: %w", err)
	}

	if memberCount == 1 {
		err = service.deleteShopWithBlobCleanup(user, shopID)
		if err != nil {
			return fmt.Errorf("failed to delete shop: %w", err)
		}
		slog.Info("Shop deleted as last member left", "user_id", user.UserID, "shop_id", shopID)
	} else {
		err = service.repo.RemoveMemberFromShop(user, shopID, user.UserID)
		if err != nil {
			return fmt.Errorf("failed to leave shop: %w", err)
		}
		slog.Info("User left shop", "user_id", user.UserID, "shop_id", shopID)
	}

	return nil
}

func (service *ServiceImpl) RemoveMemberFromShop(user *bootstrap.User, shopID string, targetUserID string) error {
	if user == nil {
		return errors.New("unauthorized user")
	}

	isAdmin, err := service.auth.IsUserShopAdmin(user, shopID)
	if err != nil {
		return fmt.Errorf("failed to verify admin status: %w", err)
	}

	if !isAdmin {
		return errors.New("only shop administrators can remove members")
	}

	if user.UserID == targetUserID {
		return errors.New("use leave shop endpoint to remove yourself")
	}

	err = service.repo.RemoveMemberFromShop(user, shopID, targetUserID)
	if err != nil {
		return fmt.Errorf("failed to remove member: %w", err)
	}

	slog.Info("Member removed from shop", "admin_user_id", user.UserID, "removed_user_id", targetUserID, "shop_id", shopID)
	return nil
}

func (service *ServiceImpl) GetShopMembers(user *bootstrap.User, shopID string) ([]response.ShopMemberWithUsername, error) {
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

	members, err := service.repo.GetShopMembers(user, shopID)
	if err != nil {
		return nil, fmt.Errorf("failed to get shop members: %w", err)
	}

	if members == nil {
		return []response.ShopMemberWithUsername{}, nil
	}

	return members, nil
}

func (service *ServiceImpl) PromoteMemberToAdmin(user *bootstrap.User, shopID string, targetUserID string) error {
	if user == nil {
		return errors.New("unauthorized user")
	}

	isAdmin, err := service.auth.IsUserShopAdmin(user, shopID)
	if err != nil {
		return fmt.Errorf("failed to verify admin status: %w", err)
	}

	if !isAdmin {
		return errors.New("only shop administrators can promote members")
	}

	isMember, err := service.auth.IsUserMemberOfShop(&bootstrap.User{UserID: targetUserID}, shopID)
	if err != nil {
		return fmt.Errorf("failed to verify target user membership: %w", err)
	}

	if !isMember {
		return errors.New("target user is not a member of this shop")
	}

	err = service.repo.UpdateMemberRole(user, shopID, targetUserID, "admin")
	if err != nil {
		return fmt.Errorf("failed to promote member to admin: %w", err)
	}

	slog.Info("Member promoted to admin", "admin_user_id", user.UserID, "promoted_user_id", targetUserID, "shop_id", shopID)
	return nil
}

// deleteShopWithBlobCleanup is a private helper that deletes a shop and cleans up associated blobs
// This is used by both DeleteShop (admin deletion) and LeaveShop (last member deletion)
func (service *ServiceImpl) deleteShopWithBlobCleanup(user *bootstrap.User, shopID string) error {
	err := service.repo.DeleteShop(user, shopID)
	if err != nil {
		slog.Error("Failed to delete shop", "error", err, "user_id", user.UserID, "shop_id", shopID)
		return fmt.Errorf("failed to delete shop: %w", err)
	}

	err = service.repo.DeleteShopMessageBlobs(shopID)
	if err != nil {
		slog.Warn("Failed to delete shop message blobs during shop deletion",
			"shop_id", shopID,
			"user_id", user.UserID,
			"error", err)
	}

	slog.Info("Shop deleted successfully", "user_id", user.UserID, "shop_id", shopID)
	return nil
}
