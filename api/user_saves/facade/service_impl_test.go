package facade

import (
	"testing"

	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/bootstrap"

	"github.com/stretchr/testify/require"
)

type quickStub struct{ called bool }

type serializedStub struct{ called bool }

type categoriesStub struct{ called bool }

type itemsStub struct{ called bool }

type imagesStub struct{ called bool }

func (stub *quickStub) GetByUser(user *bootstrap.User) ([]model.UserItemsQuick, error) {
	stub.called = true
	return nil, nil
}

func (stub *quickStub) Upsert(user *bootstrap.User, item model.UserItemsQuick) error {
	stub.called = true
	return nil
}

func (stub *quickStub) UpsertBatch(user *bootstrap.User, items []model.UserItemsQuick) error {
	stub.called = true
	return nil
}

func (stub *quickStub) Delete(user *bootstrap.User, item model.UserItemsQuick) error {
	stub.called = true
	return nil
}

func (stub *quickStub) DeleteAll(user *bootstrap.User) error {
	stub.called = true
	return nil
}

func (stub *serializedStub) GetByUser(user *bootstrap.User) ([]model.UserItemsSerialized, error) {
	stub.called = true
	return nil, nil
}

func (stub *serializedStub) Upsert(user *bootstrap.User, item model.UserItemsSerialized) error {
	stub.called = true
	return nil
}

func (stub *serializedStub) UpsertBatch(user *bootstrap.User, items []model.UserItemsSerialized) error {
	stub.called = true
	return nil
}

func (stub *serializedStub) Delete(user *bootstrap.User, item model.UserItemsSerialized) error {
	stub.called = true
	return nil
}

func (stub *serializedStub) DeleteAll(user *bootstrap.User) error {
	stub.called = true
	return nil
}

func (stub *categoriesStub) GetByUser(user *bootstrap.User) ([]model.UserItemCategory, error) {
	stub.called = true
	return nil, nil
}

func (stub *categoriesStub) Upsert(user *bootstrap.User, category model.UserItemCategory) error {
	stub.called = true
	return nil
}

func (stub *categoriesStub) Delete(user *bootstrap.User, category model.UserItemCategory) error {
	stub.called = true
	return nil
}

func (stub *categoriesStub) DeleteAll(user *bootstrap.User) error {
	stub.called = true
	return nil
}

func (stub *itemsStub) GetByCategory(user *bootstrap.User, category model.UserItemCategory) ([]model.UserItemsCategorized, error) {
	stub.called = true
	return nil, nil
}

func (stub *itemsStub) GetByUser(user *bootstrap.User) ([]model.UserItemsCategorized, error) {
	stub.called = true
	return nil, nil
}

func (stub *itemsStub) Upsert(user *bootstrap.User, item model.UserItemsCategorized) error {
	stub.called = true
	return nil
}

func (stub *itemsStub) UpsertBatch(user *bootstrap.User, items []model.UserItemsCategorized) error {
	stub.called = true
	return nil
}

func (stub *itemsStub) Delete(user *bootstrap.User, item model.UserItemsCategorized) error {
	stub.called = true
	return nil
}

func (stub *itemsStub) DeleteAll(user *bootstrap.User) error {
	stub.called = true
	return nil
}

func (stub *imagesStub) Upload(user *bootstrap.User, itemID string, tableType string, imageData []byte) (string, error) {
	stub.called = true
	return "", nil
}

func (stub *imagesStub) Delete(user *bootstrap.User, itemID string, tableType string) error {
	stub.called = true
	return nil
}

func (stub *imagesStub) Get(user *bootstrap.User, itemID string, tableType string) ([]byte, string, error) {
	stub.called = true
	return nil, "", nil
}

func TestFacadeDelegates(t *testing.T) {
	quick := &quickStub{}
	serialized := &serializedStub{}
	categories := &categoriesStub{}
	items := &itemsStub{}
	images := &imagesStub{}

	service := NewService(quick, serialized, categories, items, images)

	require.NoError(t, service.UpsertQuickSaveItemByUser(&bootstrap.User{}, model.UserItemsQuick{}))
	require.NoError(t, service.UpsertSerializedSaveItemByUser(&bootstrap.User{}, model.UserItemsSerialized{}))
	require.NoError(t, service.UpsertItemCategoryByUser(&bootstrap.User{}, model.UserItemCategory{}))
	require.NoError(t, service.UpsertCategorizedItemByUser(&bootstrap.User{}, model.UserItemsCategorized{}))
	require.NoError(t, service.DeleteItemImage(&bootstrap.User{}, "id", "quick"))

	require.True(t, quick.called)
	require.True(t, serialized.called)
	require.True(t, categories.called)
	require.True(t, items.called)
	require.True(t, images.called)
}
