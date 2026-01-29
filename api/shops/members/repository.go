package members

import (
	"miltechserver/api/response"
	"miltechserver/bootstrap"
)

type Repository interface {
	IsUserShopAdmin(user *bootstrap.User, shopID string) (bool, error)
	IsUserMemberOfShop(user *bootstrap.User, shopID string) (bool, error)
	AddMemberToShop(user *bootstrap.User, shopID string, role string) error
	RemoveMemberFromShop(user *bootstrap.User, shopID string, targetUserID string) error
	UpdateMemberRole(user *bootstrap.User, shopID string, targetUserID string, role string) error
	GetShopMembers(user *bootstrap.User, shopID string) ([]response.ShopMemberWithUsername, error)
	GetShopMemberCount(user *bootstrap.User, shopID string) (int64, error)
	DeleteShop(user *bootstrap.User, shopID string) error
	DeleteShopMessageBlobs(shopID string) error
}
