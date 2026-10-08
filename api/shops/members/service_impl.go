package members

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"miltechserver/api/response"
	"miltechserver/api/shops/members/invites"
	"miltechserver/api/shops/shared"
	"miltechserver/bootstrap"
)

type ServiceImpl struct {
	repo       Repository
	inviteRepo invites.Repository
	auth       shared.ShopAuthorization
}

func NewService(repo Repository, inviteRepo invites.Repository, auth shared.ShopAuthorization) *ServiceImpl {
	return &ServiceImpl{
		repo:       repo,
		inviteRepo: inviteRepo,
		auth:       auth,
	}
}

func (service *ServiceImpl) WithAuthorization(auth shared.ShopAuthorization) shared.AuthorizationAware {
	return &ServiceImpl{
		repo:       service.repo,
		inviteRepo: service.inviteRepo,
		auth:       auth,
	}
}

func (service *ServiceImpl) JoinShopViaInviteCode(ctx context.Context, user *bootstrap.User, inviteCode string) error {
	if user == nil {
		return errors.New("unauthorized user")
	}

	if err := service.repo.JoinViaInvite(ctx, user, inviteCode); err != nil {
		return err
	}
	return nil
}

func (service *ServiceImpl) LeaveShop(ctx context.Context, user *bootstrap.User, shopID string) error {
	if user == nil {
		return errors.New("unauthorized user")
	}

	if err := service.repo.LeaveShop(ctx, user, shopID); err != nil {
		return fmt.Errorf("failed to leave shop: %w", err)
	}
	return nil
}

func (service *ServiceImpl) RemoveMemberFromShop(ctx context.Context, user *bootstrap.User, shopID string, targetUserID string) error {
	if user == nil {
		return errors.New("unauthorized user")
	}

	if user.UserID == targetUserID {
		return errors.New("use leave shop endpoint to remove yourself")
	}

	err := service.repo.RemoveMemberFromShop(ctx, user, shopID, targetUserID)
	if err != nil {
		return fmt.Errorf("failed to remove member: %w", err)
	}

	slog.Info("Member removed from shop", "admin_user_id", user.UserID, "removed_user_id", targetUserID, "shop_id", shopID)
	return nil
}

func (service *ServiceImpl) GetShopMembers(ctx context.Context, user *bootstrap.User, shopID string) ([]response.ShopMemberWithUsername, error) {
	if user == nil {
		return nil, errors.New("unauthorized user")
	}

	isMember, err := service.auth.IsUserMemberOfShop(ctx, user, shopID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify membership: %w", err)
	}

	if !isMember {
		return nil, errors.New("access denied: user is not a member of this shop")
	}

	members, err := service.repo.GetShopMembers(ctx, user, shopID)
	if err != nil {
		return nil, fmt.Errorf("failed to get shop members: %w", err)
	}

	if members == nil {
		return []response.ShopMemberWithUsername{}, nil
	}

	return members, nil
}

func (service *ServiceImpl) PromoteMemberToAdmin(ctx context.Context, user *bootstrap.User, shopID string, targetUserID string) error {
	if user == nil {
		return errors.New("unauthorized user")
	}

	isAdmin, err := service.auth.IsUserShopAdmin(ctx, user, shopID)
	if err != nil {
		return fmt.Errorf("failed to verify admin status: %w", err)
	}

	if !isAdmin {
		return errors.New("only shop administrators can promote members")
	}

	isMember, err := service.auth.IsUserMemberOfShop(ctx, &bootstrap.User{UserID: targetUserID}, shopID)
	if err != nil {
		return fmt.Errorf("failed to verify target user membership: %w", err)
	}

	if !isMember {
		return errors.New("target user is not a member of this shop")
	}

	err = service.repo.UpdateMemberRole(ctx, user, shopID, targetUserID, "admin")
	if err != nil {
		return fmt.Errorf("failed to promote member to admin: %w", err)
	}

	slog.Info("Member promoted to admin", "admin_user_id", user.UserID, "promoted_user_id", targetUserID, "shop_id", shopID)
	return nil
}
