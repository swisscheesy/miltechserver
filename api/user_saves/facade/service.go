package facade

import (
	"context"
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/bootstrap"
)

// Service provides a unified interface for user saves domains.
type Service interface {
	// Quick Items
	GetQuickSaveItemsByUser(ctx context.Context, user *bootstrap.User) ([]model.UserItemsQuick, error)
	UpsertQuickSaveItemByUser(ctx context.Context, user *bootstrap.User, quick model.UserItemsQuick) error
	UpsertQuickSaveItemListByUser(ctx context.Context, user *bootstrap.User, quickItems []model.UserItemsQuick) error
	DeleteQuickSaveItemByUser(ctx context.Context, user *bootstrap.User, quick model.UserItemsQuick) error
	DeleteAllQuickSaveItemsByUser(ctx context.Context, user *bootstrap.User) error

	// Serialized Items
	GetSerializedItemsByUser(ctx context.Context, user *bootstrap.User) ([]model.UserItemsSerialized, error)
	UpsertSerializedSaveItemByUser(ctx context.Context, user *bootstrap.User, serializedItem model.UserItemsSerialized) error
	UpsertSerializedSaveItemListByUser(ctx context.Context, user *bootstrap.User, serializedItems []model.UserItemsSerialized) error
	DeleteSerializedSaveItemByUser(ctx context.Context, user *bootstrap.User, serializedItem model.UserItemsSerialized) error
	DeleteAllSerializedItemsByUser(ctx context.Context, user *bootstrap.User) error

	// Categories
	GetItemCategoriesByUser(ctx context.Context, user *bootstrap.User) ([]model.UserItemCategory, error)
	UpsertItemCategoryByUser(ctx context.Context, user *bootstrap.User, itemCategory model.UserItemCategory) error
	DeleteItemCategory(ctx context.Context, user *bootstrap.User, itemCategory model.UserItemCategory) error
	DeleteAllItemCategories(ctx context.Context, user *bootstrap.User) error

	// Categorized Items
	GetCategorizedItemsByUser(ctx context.Context, user *bootstrap.User) ([]model.UserItemsCategorized, error)
	GetCategorizedItemsByCategory(ctx context.Context, user *bootstrap.User, itemCategory model.UserItemCategory) ([]model.UserItemsCategorized, error)
	UpsertCategorizedItemByUser(ctx context.Context, user *bootstrap.User, categorizedItem model.UserItemsCategorized) error
	UpsertCategorizedItemListByUser(ctx context.Context, user *bootstrap.User, categorizedItems []model.UserItemsCategorized) error
	DeleteCategorizedItemByCategoryId(ctx context.Context, user *bootstrap.User, categorizedItem model.UserItemsCategorized) error
	DeleteAllCategorizedItems(ctx context.Context, user *bootstrap.User) error

	// Images
	UploadItemImage(ctx context.Context, user *bootstrap.User, itemID string, tableType string, imageData []byte) (string, error)
	DeleteItemImage(ctx context.Context, user *bootstrap.User, itemID string, tableType string) error
	GetItemImage(ctx context.Context, user *bootstrap.User, itemID string, tableType string) ([]byte, string, error)
}
