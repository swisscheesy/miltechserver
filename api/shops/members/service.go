package members

import (
	"context"
	"miltechserver/api/response"
	"miltechserver/bootstrap"
)

type Service interface {
	JoinShopViaInviteCode(ctx context.Context, user *bootstrap.User, inviteCode string) error
	LeaveShop(ctx context.Context, user *bootstrap.User, shopID string) error
	RemoveMemberFromShop(ctx context.Context, user *bootstrap.User, shopID string, targetUserID string) error
	GetShopMembers(ctx context.Context, user *bootstrap.User, shopID string) ([]response.ShopMemberWithUsername, error)
	PromoteMemberToAdmin(ctx context.Context, user *bootstrap.User, shopID string, targetUserID string) error
}
