package service

import (
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/bootstrap"
)

type UserSavesService interface {
	// Quick Save Items
	GetQuickSaveItemsByUser(user *bootstrap.User) ([]model.UserItemsQuick, error)
	UpsertQuickSaveItemByUser(user *bootstrap.User, quick model.UserItemsQuick) error
	UpsertQuickSaveItemListByUser(user *bootstrap.User, quickItems []model.UserItemsQuick) error
	DeleteQuickSaveItemByUser(user *bootstrap.User, quick model.UserItemsQuick) error
	DeleteAllQuickSaveItemsByUser(user *bootstrap.User) error

	// Serialized Save Items
	GetSerializedItemsByUser(user *bootstrap.User) ([]model.UserItemsSerialized, error)
	UpsertSerializedSaveItemByUser(user *bootstrap.User, serializedItem model.UserItemsSerialized) error
	UpsertSerializedSaveItemListByUser(user *bootstrap.User, serializedItems []model.UserItemsSerialized) error
	DeleteSerializedSaveItemByUser(user *bootstrap.User, serializedItem model.UserItemsSerialized) error
	DeleteAllSerializedItemsByUser(user *bootstrap.User) error

	// Item Categories
	GetItemCategoriesByUser(user *bootstrap.User) ([]model.UserItemCategory, error)
	UpsertItemCategoryByUser(user *bootstrap.User, itemCategory model.UserItemCategory) error
	DeleteItemCategory(user *bootstrap.User, itemCategory model.UserItemCategory) error
	DeleteAllItemCategories(user *bootstrap.User) error

	// Categorized Items
	GetCategorizedItemsByUser(user *bootstrap.User) ([]model.UserItemsCategorized, error)
	GetCategorizedItemsByCategory(user *bootstrap.User, itemCategory model.UserItemCategory) ([]model.UserItemsCategorized, error)
	UpsertCategorizedItemByUser(user *bootstrap.User, categorizedItem model.UserItemsCategorized) error
	UpsertCategorizedItemListByUser(user *bootstrap.User, categorizedItems []model.UserItemsCategorized) error
	DeleteCategorizedItemByCategoryId(user *bootstrap.User, categorizedItem model.UserItemsCategorized) error
	DeleteAllCategorizedItems(user *bootstrap.User) error

	// Image Management
	UploadItemImage(user *bootstrap.User, itemID string, tableType string, imageData []byte) (string, error)
	DeleteItemImage(user *bootstrap.User, itemID string, tableType string) error
	GetItemImage(user *bootstrap.User, itemID string, tableType string) ([]byte, string, error)
}
