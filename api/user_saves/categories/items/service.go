package items

import (
	"context"
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/bootstrap"
)

type Service interface {
	GetByCategory(ctx context.Context, user *bootstrap.User, category model.UserItemCategory) ([]model.UserItemsCategorized, error)
	GetByUser(ctx context.Context, user *bootstrap.User) ([]model.UserItemsCategorized, error)
	Upsert(ctx context.Context, user *bootstrap.User, item model.UserItemsCategorized) error
	UpsertBatch(ctx context.Context, user *bootstrap.User, items []model.UserItemsCategorized) error
	Delete(ctx context.Context, user *bootstrap.User, item model.UserItemsCategorized) error
	DeleteAll(ctx context.Context, user *bootstrap.User) error
}
