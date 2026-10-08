package invites

import (
	"context"
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/bootstrap"
)

type Service interface {
	GenerateInviteCode(ctx context.Context, user *bootstrap.User, shopID string) (*model.ShopInviteCodes, error)
	GetInviteCodesByShop(ctx context.Context, user *bootstrap.User, shopID string) ([]model.ShopInviteCodes, error)
	DeactivateInviteCode(ctx context.Context, user *bootstrap.User, codeID string) error
	DeleteInviteCode(ctx context.Context, user *bootstrap.User, codeID string) error
}
