package facade

import (
	"context"
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

func (service *ServiceImpl) GetQuickSaveItemsByUser(ctx context.Context, user *bootstrap.User) ([]model.UserItemsQuick, error) {
	return service.quickService.GetByUser(ctx, user)
}

func (service *ServiceImpl) UpsertQuickSaveItemByUser(ctx context.Context, user *bootstrap.User, quickItem model.UserItemsQuick) error {
	return service.quickService.Upsert(ctx, user, quickItem)
}

func (service *ServiceImpl) UpsertQuickSaveItemListByUser(ctx context.Context, user *bootstrap.User, quickItems []model.UserItemsQuick) error {
	return service.quickService.UpsertBatch(ctx, user, quickItems)
}

func (service *ServiceImpl) DeleteQuickSaveItemByUser(ctx context.Context, user *bootstrap.User, quickItem model.UserItemsQuick) error {
	return service.quickService.Delete(ctx, user, quickItem)
}

func (service *ServiceImpl) DeleteAllQuickSaveItemsByUser(ctx context.Context, user *bootstrap.User) error {
	return service.quickService.DeleteAll(ctx, user)
}

func (service *ServiceImpl) GetSerializedItemsByUser(ctx context.Context, user *bootstrap.User) ([]model.UserItemsSerialized, error) {
	return service.serializedService.GetByUser(ctx, user)
}

func (service *ServiceImpl) UpsertSerializedSaveItemByUser(ctx context.Context, user *bootstrap.User, serializedItem model.UserItemsSerialized) error {
	return service.serializedService.Upsert(ctx, user, serializedItem)
}

func (service *ServiceImpl) UpsertSerializedSaveItemListByUser(ctx context.Context, user *bootstrap.User, serializedItems []model.UserItemsSerialized) error {
	return service.serializedService.UpsertBatch(ctx, user, serializedItems)
}

func (service *ServiceImpl) DeleteSerializedSaveItemByUser(ctx context.Context, user *bootstrap.User, serializedItem model.UserItemsSerialized) error {
	return service.serializedService.Delete(ctx, user, serializedItem)
}

func (service *ServiceImpl) DeleteAllSerializedItemsByUser(ctx context.Context, user *bootstrap.User) error {
	return service.serializedService.DeleteAll(ctx, user)
}

func (service *ServiceImpl) GetItemCategoriesByUser(ctx context.Context, user *bootstrap.User) ([]model.UserItemCategory, error) {
	return service.categoriesService.GetByUser(ctx, user)
}

func (service *ServiceImpl) UpsertItemCategoryByUser(ctx context.Context, user *bootstrap.User, itemCategory model.UserItemCategory) error {
	return service.categoriesService.Upsert(ctx, user, itemCategory)
}

func (service *ServiceImpl) DeleteItemCategory(ctx context.Context, user *bootstrap.User, itemCategory model.UserItemCategory) error {
	return service.categoriesService.Delete(ctx, user, itemCategory)
}

func (service *ServiceImpl) DeleteAllItemCategories(ctx context.Context, user *bootstrap.User) error {
	return service.categoriesService.DeleteAll(ctx, user)
}

func (service *ServiceImpl) GetCategorizedItemsByUser(ctx context.Context, user *bootstrap.User) ([]model.UserItemsCategorized, error) {
	return service.itemsService.GetByUser(ctx, user)
}

func (service *ServiceImpl) GetCategorizedItemsByCategory(ctx context.Context, user *bootstrap.User, itemCategory model.UserItemCategory) ([]model.UserItemsCategorized, error) {
	return service.itemsService.GetByCategory(ctx, user, itemCategory)
}

func (service *ServiceImpl) UpsertCategorizedItemByUser(ctx context.Context, user *bootstrap.User, categorizedItem model.UserItemsCategorized) error {
	return service.itemsService.Upsert(ctx, user, categorizedItem)
}

func (service *ServiceImpl) UpsertCategorizedItemListByUser(ctx context.Context, user *bootstrap.User, categorizedItems []model.UserItemsCategorized) error {
	return service.itemsService.UpsertBatch(ctx, user, categorizedItems)
}

func (service *ServiceImpl) DeleteCategorizedItemByCategoryId(ctx context.Context, user *bootstrap.User, categorizedItem model.UserItemsCategorized) error {
	return service.itemsService.Delete(ctx, user, categorizedItem)
}

func (service *ServiceImpl) DeleteAllCategorizedItems(ctx context.Context, user *bootstrap.User) error {
	return service.itemsService.DeleteAll(ctx, user)
}

func (service *ServiceImpl) UploadItemImage(ctx context.Context, user *bootstrap.User, itemID string, tableType string, imageData []byte) (string, error) {
	return service.imagesService.Upload(ctx, user, itemID, tableType, imageData)
}

func (service *ServiceImpl) DeleteItemImage(ctx context.Context, user *bootstrap.User, itemID string, tableType string) error {
	return service.imagesService.Delete(ctx, user, itemID, tableType)
}

func (service *ServiceImpl) GetItemImage(ctx context.Context, user *bootstrap.User, itemID string, tableType string) ([]byte, string, error) {
	return service.imagesService.Get(ctx, user, itemID, tableType)
}
