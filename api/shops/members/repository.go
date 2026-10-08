package members

import (
	"context"
	"miltechserver/api/response"
	"miltechserver/bootstrap"
)

type Repository interface {
	IsUserShopAdmin(ctx context.Context, user *bootstrap.User, shopID string) (bool, error)
	IsUserMemberOfShop(ctx context.Context, user *bootstrap.User, shopID string) (bool, error)
	JoinViaInvite(ctx context.Context, user *bootstrap.User, code string) error
	LeaveShop(ctx context.Context, user *bootstrap.User, shopID string) error
	RemoveMemberFromShop(ctx context.Context, user *bootstrap.User, shopID string, targetUserID string) error
	UpdateMemberRole(ctx context.Context, user *bootstrap.User, shopID string, targetUserID string, role string) error
	GetShopMembers(ctx context.Context, user *bootstrap.User, shopID string) ([]response.ShopMemberWithUsername, error)
}
