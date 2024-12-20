package service

import (
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/bootstrap"
)

type UserSavesService interface {
	GetQuickSaveItemsByUser(user *bootstrap.User) ([]model.UserItemsQuick, error)
}
