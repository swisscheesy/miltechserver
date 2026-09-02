package categories

import (
	"context"
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/bootstrap"
)

type Service interface {
	GetByUser(ctx context.Context, user *bootstrap.User) ([]model.UserItemCategory, error)
	Upsert(ctx context.Context, user *bootstrap.User, category model.UserItemCategory) error
	Delete(ctx context.Context, user *bootstrap.User, category model.UserItemCategory) error
	DeleteAll(ctx context.Context, user *bootstrap.User) error
}
