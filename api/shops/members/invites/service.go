package invites

import (
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/bootstrap"
)

type Service interface {
	GenerateInviteCode(user *bootstrap.User, shopID string) (*model.ShopInviteCodes, error)
	GetInviteCodesByShop(user *bootstrap.User, shopID string) ([]model.ShopInviteCodes, error)
	DeactivateInviteCode(user *bootstrap.User, codeID string) error
	DeleteInviteCode(user *bootstrap.User, codeID string) error
}
