package service

import (
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/bootstrap"
)

type UserSavesService interface {
	GetQuickSaveItemsByUser(user *bootstrap.User) ([]model.UserItemsQuick, error)
	UpsertQuickSaveItemByUser(user *bootstrap.User, quick model.UserItemsQuick) error
	UpsertQuickSaveItemListByUser(user *bootstrap.User, quickItems []model.UserItemsQuick) error
	DeleteQuickSaveItemByUser(user *bootstrap.User, quick model.UserItemsQuick) error
	DeleteAllQuickSaveItemsByUser(user *bootstrap.User) error

	GetSerializedItemsByUser(user *bootstrap.User) ([]model.UserItemsSerialized, error)
}
