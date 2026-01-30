package facade

import (
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/user_saves/categories"
	categoryitems "miltechserver/api/user_saves/categories/items"
	"miltechserver/api/user_saves/images"
	"miltechserver/api/user_saves/quick"
	"miltechserver/api/user_saves/serialized"
	"miltechserver/bootstrap"
)

type ServiceImpl struct {
	quickService      quick.Service
	serializedService serialized.Service
	categoriesService categories.Service
	itemsService      categoryitems.Service
	imagesService     images.Service
}

func NewService(
	quickService quick.Service,
	serializedService serialized.Service,
	categoriesService categories.Service,
	itemsService categoryitems.Service,
	imagesService images.Service,
) *ServiceImpl {
	return &ServiceImpl{
		quickService:      quickService,
		serializedService: serializedService,
		categoriesService: categoriesService,
		itemsService:      itemsService,
		imagesService:     imagesService,
	}
}

func (service *ServiceImpl) GetQuickSaveItemsByUser(user *bootstrap.User) ([]model.UserItemsQuick, error) {
	return service.quickService.GetByUser(user)
}

func (service *ServiceImpl) UpsertQuickSaveItemByUser(user *bootstrap.User, quickItem model.UserItemsQuick) error {
	return service.quickService.Upsert(user, quickItem)
}

func (service *ServiceImpl) UpsertQuickSaveItemListByUser(user *bootstrap.User, quickItems []model.UserItemsQuick) error {
	return service.quickService.UpsertBatch(user, quickItems)
}

func (service *ServiceImpl) DeleteQuickSaveItemByUser(user *bootstrap.User, quickItem model.UserItemsQuick) error {
	return service.quickService.Delete(user, quickItem)
}

func (service *ServiceImpl) DeleteAllQuickSaveItemsByUser(user *bootstrap.User) error {
	return service.quickService.DeleteAll(user)
}

func (service *ServiceImpl) GetSerializedItemsByUser(user *bootstrap.User) ([]model.UserItemsSerialized, error) {
	return service.serializedService.GetByUser(user)
}

func (service *ServiceImpl) UpsertSerializedSaveItemByUser(user *bootstrap.User, serializedItem model.UserItemsSerialized) error {
	return service.serializedService.Upsert(user, serializedItem)
}

func (service *ServiceImpl) UpsertSerializedSaveItemListByUser(user *bootstrap.User, serializedItems []model.UserItemsSerialized) error {
	return service.serializedService.UpsertBatch(user, serializedItems)
}

func (service *ServiceImpl) DeleteSerializedSaveItemByUser(user *bootstrap.User, serializedItem model.UserItemsSerialized) error {
	return service.serializedService.Delete(user, serializedItem)
}

func (service *ServiceImpl) DeleteAllSerializedItemsByUser(user *bootstrap.User) error {
	return service.serializedService.DeleteAll(user)
}

func (service *ServiceImpl) GetItemCategoriesByUser(user *bootstrap.User) ([]model.UserItemCategory, error) {
	return service.categoriesService.GetByUser(user)
}

func (service *ServiceImpl) UpsertItemCategoryByUser(user *bootstrap.User, itemCategory model.UserItemCategory) error {
	return service.categoriesService.Upsert(user, itemCategory)
}

func (service *ServiceImpl) DeleteItemCategory(user *bootstrap.User, itemCategory model.UserItemCategory) error {
	return service.categoriesService.Delete(user, itemCategory)
}

func (service *ServiceImpl) DeleteAllItemCategories(user *bootstrap.User) error {
	return service.categoriesService.DeleteAll(user)
}

func (service *ServiceImpl) GetCategorizedItemsByUser(user *bootstrap.User) ([]model.UserItemsCategorized, error) {
	return service.itemsService.GetByUser(user)
}

func (service *ServiceImpl) GetCategorizedItemsByCategory(user *bootstrap.User, itemCategory model.UserItemCategory) ([]model.UserItemsCategorized, error) {
	return service.itemsService.GetByCategory(user, itemCategory)
}

func (service *ServiceImpl) UpsertCategorizedItemByUser(user *bootstrap.User, categorizedItem model.UserItemsCategorized) error {
	return service.itemsService.Upsert(user, categorizedItem)
}

func (service *ServiceImpl) UpsertCategorizedItemListByUser(user *bootstrap.User, categorizedItems []model.UserItemsCategorized) error {
	return service.itemsService.UpsertBatch(user, categorizedItems)
}

func (service *ServiceImpl) DeleteCategorizedItemByCategoryId(user *bootstrap.User, categorizedItem model.UserItemsCategorized) error {
	return service.itemsService.Delete(user, categorizedItem)
}

func (service *ServiceImpl) DeleteAllCategorizedItems(user *bootstrap.User) error {
	return service.itemsService.DeleteAll(user)
}

func (service *ServiceImpl) UploadItemImage(user *bootstrap.User, itemID string, tableType string, imageData []byte) (string, error) {
	return service.imagesService.Upload(user, itemID, tableType, imageData)
}

func (service *ServiceImpl) DeleteItemImage(user *bootstrap.User, itemID string, tableType string) error {
	return service.imagesService.Delete(user, itemID, tableType)
}

func (service *ServiceImpl) GetItemImage(user *bootstrap.User, itemID string, tableType string) ([]byte, string, error) {
	return service.imagesService.Get(user, itemID, tableType)
}
