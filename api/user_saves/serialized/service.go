package serialized

import (
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/bootstrap"
)

type Service interface {
	GetByUser(user *bootstrap.User) ([]model.UserItemsSerialized, error)
	Upsert(user *bootstrap.User, item model.UserItemsSerialized) error
	UpsertBatch(user *bootstrap.User, items []model.UserItemsSerialized) error
	Delete(user *bootstrap.User, item model.UserItemsSerialized) error
	DeleteAll(user *bootstrap.User) error
}
