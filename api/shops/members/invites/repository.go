package invites

import (
	"context"
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/bootstrap"
)

type Repository interface {
	CreateInviteCode(ctx context.Context, user *bootstrap.User, inviteCode model.ShopInviteCodes) (*model.ShopInviteCodes, error)
	GetInviteCodeByCode(ctx context.Context, code string) (*model.ShopInviteCodes, error)
	GetInviteCodeByID(ctx context.Context, codeID string) (*model.ShopInviteCodes, error)
	GetInviteCodesByShop(ctx context.Context, user *bootstrap.User, shopID string) ([]model.ShopInviteCodes, error)
	DeactivateInviteCode(ctx context.Context, user *bootstrap.User, codeID string) error
	DeleteInviteCode(ctx context.Context, user *bootstrap.User, codeID string) error
}
