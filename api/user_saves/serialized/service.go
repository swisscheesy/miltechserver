package serialized

import (
	"context"
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/bootstrap"
)

type Service interface {
	GetByUser(ctx context.Context, user *bootstrap.User) ([]model.UserItemsSerialized, error)
	Upsert(ctx context.Context, user *bootstrap.User, item model.UserItemsSerialized) error
	UpsertBatch(ctx context.Context, user *bootstrap.User, items []model.UserItemsSerialized) error
	Delete(ctx context.Context, user *bootstrap.User, item model.UserItemsSerialized) error
	DeleteAll(ctx context.Context, user *bootstrap.User) error
}
