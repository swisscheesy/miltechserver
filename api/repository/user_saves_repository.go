package repository

import (
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/bootstrap"
)

type UserSavesRepository interface {
	GetQuickSaveItemsByUserId(user *bootstrap.User) ([]model.UserItemsQuick, error)
	UpsertQuickSaveItemByUser(user *bootstrap.User, quickItem model.UserItemsQuick) error
	UpsertQuickSaveItemListByUser(user *bootstrap.User, quickItems []model.UserItemsQuick) error
	DeleteQuickSaveItemByUser(user *bootstrap.User, quickItem model.UserItemsQuick) error
	DeleteAllQuickSaveItemsByUser(user *bootstrap.User) error

	GetSerializedItemsByUserId(user *bootstrap.User) ([]model.UserItemsSerialized, error)
	UpsertSerializedSaveItemByUser(user *bootstrap.User, serializedItem model.UserItemsSerialized) error
	UpsertSerializedSaveItemListByUser(user *bootstrap.User, serializedItems []model.UserItemsSerialized) error
	DeleteSerializedSaveItemByUser(user *bootstrap.User, serializedItem model.UserItemsSerialized) error
	DeleteAllSerializedItemsByUser(user *bootstrap.User) error
}
