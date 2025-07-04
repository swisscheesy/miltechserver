package repository

import (
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/bootstrap"
)

type UserSavesRepository interface {
	// Quick Save Items
	GetQuickSaveItemsByUserId(user *bootstrap.User) ([]model.UserItemsQuick, error)
	UpsertQuickSaveItemByUser(user *bootstrap.User, quickItem model.UserItemsQuick) error
	UpsertQuickSaveItemListByUser(user *bootstrap.User, quickItems []model.UserItemsQuick) error
	DeleteQuickSaveItemByUser(user *bootstrap.User, quickItem model.UserItemsQuick) error
	DeleteAllQuickSaveItemsByUser(user *bootstrap.User) error

	// Serialized Save Items
	GetSerializedItemsByUserId(user *bootstrap.User) ([]model.UserItemsSerialized, error)
	UpsertSerializedSaveItemByUser(user *bootstrap.User, serializedItem model.UserItemsSerialized) error
	UpsertSerializedSaveItemListByUser(user *bootstrap.User, serializedItems []model.UserItemsSerialized) error
	DeleteSerializedSaveItemByUser(user *bootstrap.User, serializedItem model.UserItemsSerialized) error
	DeleteAllSerializedItemsByUser(user *bootstrap.User) error

	// Item Categories
	GetUserItemCategories(user *bootstrap.User) ([]model.UserItemCategory, error)
	UpsertUserItemCategory(user *bootstrap.User, itemCategory model.UserItemCategory) error
	DeleteUserItemCategory(user *bootstrap.User, itemCategory model.UserItemCategory) error
	DeleteAllUserItemCategories(user *bootstrap.User) error

	// Categorized Items
	GetCategorizedItemsByCategory(user *bootstrap.User, itemCategory model.UserItemCategory) ([]model.UserItemsCategorized, error)
	GetCategorizedItemsByUser(user *bootstrap.User) ([]model.UserItemsCategorized, error)
	UpsertUserItemsCategorized(user *bootstrap.User, categorizedItem model.UserItemsCategorized) error
	UpsertUserItemsCategorizedList(user *bootstrap.User, categorizedItems []model.UserItemsCategorized) error
	DeleteUserItemsCategorized(user *bootstrap.User, categorizedItem model.UserItemsCategorized) error
	DeleteAllUserItemsCategorized(user *bootstrap.User) error

	// Image Management
	UploadItemImage(user *bootstrap.User, itemID string, tableType string, imageData []byte) (string, error)
	DeleteItemImage(user *bootstrap.User, itemID string, tableType string) error
	GetItemImage(user *bootstrap.User, itemID string, tableType string) ([]byte, string, error)
}
