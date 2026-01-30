package items

import (
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/bootstrap"
)

type Service interface {
	GetByCategory(user *bootstrap.User, category model.UserItemCategory) ([]model.UserItemsCategorized, error)
	GetByUser(user *bootstrap.User) ([]model.UserItemsCategorized, error)
	Upsert(user *bootstrap.User, item model.UserItemsCategorized) error
	UpsertBatch(user *bootstrap.User, items []model.UserItemsCategorized) error
	Delete(user *bootstrap.User, item model.UserItemsCategorized) error
	DeleteAll(user *bootstrap.User) error
}
