package quick

import (
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/bootstrap"
)

type Repository interface {
	GetByUser(user *bootstrap.User) ([]model.UserItemsQuick, error)
	Upsert(user *bootstrap.User, item model.UserItemsQuick) error
	UpsertBatch(user *bootstrap.User, items []model.UserItemsQuick) error
	Delete(user *bootstrap.User, item model.UserItemsQuick) error
	DeleteAll(user *bootstrap.User) error
}
