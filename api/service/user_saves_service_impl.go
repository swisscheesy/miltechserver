package service

import (
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/repository"
	"miltechserver/bootstrap"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
)

type UserSavesServiceImpl struct {
	UserSavesRepository repository.UserSavesRepository
	BlobClient          *azblob.Client
}

func NewUserSavesServiceImpl(userSavesRepository repository.UserSavesRepository, blobClient *azblob.Client) *UserSavesServiceImpl {
	return &UserSavesServiceImpl{UserSavesRepository: userSavesRepository, BlobClient: blobClient}
}

// GetQuickSaveItemsByUser is a function that returns the quick save items of a user
func (service *UserSavesServiceImpl) GetQuickSaveItemsByUser(user *bootstrap.User) ([]model.UserItemsQuick, error) {
	var items, err = service.UserSavesRepository.GetQuickSaveItemsByUserId(user)
	// if test is nil, return an empty array
	if items == nil {
		return []model.UserItemsQuick{}, nil
	}

	return items, err
}

// UpsertQuickSaveItemByUser is a function that upserts a quick save item for a user
func (service *UserSavesServiceImpl) UpsertQuickSaveItemByUser(user *bootstrap.User, quick model.UserItemsQuick) error {
	return service.UserSavesRepository.UpsertQuickSaveItemByUser(user, quick)
}

// DeleteQuickSaveItemByUser is a function that deletes a quick save item for a user
func (service *UserSavesServiceImpl) DeleteQuickSaveItemByUser(user *bootstrap.User, quick model.UserItemsQuick) error {
	return service.UserSavesRepository.DeleteQuickSaveItemByUser(user, quick)
}

// DeleteAllQuickSaveItemsByUser is a function that deletes all quick save items for a user
func (service *UserSavesServiceImpl) DeleteAllQuickSaveItemsByUser(user *bootstrap.User) error {
	return service.UserSavesRepository.DeleteAllQuickSaveItemsByUser(user)
}

// UpsertQuickSaveItemListByUser is a function that upserts a list of quick save items for a user
func (service *UserSavesServiceImpl) UpsertQuickSaveItemListByUser(user *bootstrap.User, quickItems []model.UserItemsQuick) error {
	return service.UserSavesRepository.UpsertQuickSaveItemListByUser(user, quickItems)
}

// GetSerializedItemsByUser is a function that returns the serialized items of a user
func (service *UserSavesServiceImpl) GetSerializedItemsByUser(user *bootstrap.User) ([]model.UserItemsSerialized, error) {
	var items, err = service.UserSavesRepository.GetSerializedItemsByUserId(user)
	if items == nil {
		return []model.UserItemsSerialized{}, nil
	}
	return items, err
}

// UpsertSerializedSaveItemByUser is a function that upserts a serialized item for a user
func (service *UserSavesServiceImpl) UpsertSerializedSaveItemByUser(user *bootstrap.User, serializedItem model.UserItemsSerialized) error {
	return service.UserSavesRepository.UpsertSerializedSaveItemByUser(user, serializedItem)
}

// DeleteSerializedSaveItemByUser is a function that deletes a serialized item for a user
func (service *UserSavesServiceImpl) DeleteSerializedSaveItemByUser(user *bootstrap.User, serializedItem model.UserItemsSerialized) error {
	return service.UserSavesRepository.DeleteSerializedSaveItemByUser(user, serializedItem)
}

// DeleteAllSerializedItemsByUser is a function that deletes all serialized items for a user
func (service *UserSavesServiceImpl) DeleteAllSerializedItemsByUser(user *bootstrap.User) error {
	return service.UserSavesRepository.DeleteAllSerializedItemsByUser(user)
}

// UpsertSerializedSaveItemListByUser is a function that upserts a list of serialized items for a user
func (service *UserSavesServiceImpl) UpsertSerializedSaveItemListByUser(user *bootstrap.User, serializedItems []model.UserItemsSerialized) error {
	return service.UserSavesRepository.UpsertSerializedSaveItemListByUser(user, serializedItems)
}

// GetItemCategoriesByUser is a function that returns the item categories of a user
func (service *UserSavesServiceImpl) GetItemCategoriesByUser(user *bootstrap.User) ([]model.UserItemCategory, error) {
	var items, err = service.UserSavesRepository.GetUserItemCategories(user)
	if items == nil {
		return []model.UserItemCategory{}, nil
	}
	return items, err
}

// UpsertItemCategoryByUser is a function that upserts an item category for a user
func (service *UserSavesServiceImpl) UpsertItemCategoryByUser(user *bootstrap.User, itemCategory model.UserItemCategory) error {
	return service.UserSavesRepository.UpsertUserItemCategory(user, itemCategory)
}

// DeleteItemCategory is a function that deletes an item category for a user
func (service *UserSavesServiceImpl) DeleteItemCategory(user *bootstrap.User, itemCategory model.UserItemCategory) error {
	return service.UserSavesRepository.DeleteUserItemCategory(user, itemCategory)
}

// DeleteAllItemCategories is a function that deletes all item categories for a user
func (service *UserSavesServiceImpl) DeleteAllItemCategories(user *bootstrap.User) error {
	return service.UserSavesRepository.DeleteAllUserItemCategories(user)
}

func (service *UserSavesServiceImpl) GetCategorizedItemsByUser(user *bootstrap.User) ([]model.UserItemsCategorized, error) {
	var items, err = service.UserSavesRepository.GetCategorizedItemsByUser(user)
	if items == nil {
		return []model.UserItemsCategorized{}, nil
	}
	return items, err
}

func (service *UserSavesServiceImpl) GetCategorizedItemsByCategory(user *bootstrap.User, itemCategory model.UserItemCategory) ([]model.UserItemsCategorized, error) {
	var items, err = service.UserSavesRepository.GetCategorizedItemsByCategory(user, itemCategory)
	if items == nil {
		return []model.UserItemsCategorized{}, nil
	}
	return items, err
}

func (service *UserSavesServiceImpl) UpsertCategorizedItemByUser(user *bootstrap.User, categorizedItem model.UserItemsCategorized) error {
	return service.UserSavesRepository.UpsertUserItemsCategorized(user, categorizedItem)
}

func (service *UserSavesServiceImpl) UpsertCategorizedItemListByUser(user *bootstrap.User, categorizedItems []model.UserItemsCategorized) error {
	return service.UserSavesRepository.UpsertUserItemsCategorizedList(user, categorizedItems)
}

func (service *UserSavesServiceImpl) DeleteCategorizedItemByCategoryId(user *bootstrap.User, categorizedItem model.UserItemsCategorized) error {
	return service.UserSavesRepository.DeleteUserItemsCategorized(user, categorizedItem)
}

// DeleteAllCategorizedItems is a function that deletes all categorized items for a user
func (service *UserSavesServiceImpl) DeleteAllCategorizedItems(user *bootstrap.User) error {
	return service.UserSavesRepository.DeleteAllUserItemsCategorized(user)
}

// UploadItemImage uploads an item image to Azure Blob Storage
func (service *UserSavesServiceImpl) UploadItemImage(user *bootstrap.User, itemID string, tableType string, imageData []byte) (string, error) {
	return service.UserSavesRepository.UploadItemImage(user, itemID, tableType, imageData)
}

// DeleteItemImage deletes an item image from Azure Blob Storage
func (service *UserSavesServiceImpl) DeleteItemImage(user *bootstrap.User, itemID string, tableType string) error {
	return service.UserSavesRepository.DeleteItemImage(user, itemID, tableType)
}

// GetItemImage retrieves an item image from Azure Blob Storage
func (service *UserSavesServiceImpl) GetItemImage(user *bootstrap.User, itemID string, tableType string) ([]byte, string, error) {
	return service.UserSavesRepository.GetItemImage(user, itemID, tableType)
}
