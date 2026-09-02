package quick

import (
	"context"
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/bootstrap"
)

type Service interface {
	GetByUser(ctx context.Context, user *bootstrap.User) ([]model.UserItemsQuick, error)
	Upsert(ctx context.Context, user *bootstrap.User, item model.UserItemsQuick) error
	UpsertBatch(ctx context.Context, user *bootstrap.User, items []model.UserItemsQuick) error
	Delete(ctx context.Context, user *bootstrap.User, item model.UserItemsQuick) error
	DeleteAll(ctx context.Context, user *bootstrap.User) error
}
