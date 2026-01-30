package categories

import (
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/bootstrap"
)

type Service interface {
	GetByUser(user *bootstrap.User) ([]model.UserItemCategory, error)
	Upsert(user *bootstrap.User, category model.UserItemCategory) error
	Delete(user *bootstrap.User, category model.UserItemCategory) error
	DeleteAll(user *bootstrap.User) error
}
