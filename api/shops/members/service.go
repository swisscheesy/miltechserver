package members

import (
	"miltechserver/api/response"
	"miltechserver/bootstrap"
)

type Service interface {
	JoinShopViaInviteCode(user *bootstrap.User, inviteCode string) error
	LeaveShop(user *bootstrap.User, shopID string) error
	RemoveMemberFromShop(user *bootstrap.User, shopID string, targetUserID string) error
	GetShopMembers(user *bootstrap.User, shopID string) ([]response.ShopMemberWithUsername, error)
	PromoteMemberToAdmin(user *bootstrap.User, shopID string, targetUserID string) error
}
