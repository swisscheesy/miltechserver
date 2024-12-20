package repository

import (
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/bootstrap"
)

type UserSavesRepository interface {
	GetQuickSaveItemsByUserId(user *bootstrap.User) ([]model.UserItemsQuick, error)
}
