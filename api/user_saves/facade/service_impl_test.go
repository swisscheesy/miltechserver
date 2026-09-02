package facade

import (
	"context"
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

func (stub *quickStub) GetByUser(ctx context.Context, user *bootstrap.User) ([]model.UserItemsQuick, error) {
	stub.called = true
	return nil, nil
}

func (stub *quickStub) Upsert(ctx context.Context, user *bootstrap.User, item model.UserItemsQuick) error {
	stub.called = true
	return nil
}

func (stub *quickStub) UpsertBatch(ctx context.Context, user *bootstrap.User, items []model.UserItemsQuick) error {
	stub.called = true
	return nil
}

func (stub *quickStub) Delete(ctx context.Context, user *bootstrap.User, item model.UserItemsQuick) error {
	stub.called = true
	return nil
}

func (stub *quickStub) DeleteAll(ctx context.Context, user *bootstrap.User) error {
	stub.called = true
	return nil
}

func (stub *serializedStub) GetByUser(ctx context.Context, user *bootstrap.User) ([]model.UserItemsSerialized, error) {
	stub.called = true
	return nil, nil
}

func (stub *serializedStub) Upsert(ctx context.Context, user *bootstrap.User, item model.UserItemsSerialized) error {
	stub.called = true
	return nil
}

func (stub *serializedStub) UpsertBatch(ctx context.Context, user *bootstrap.User, items []model.UserItemsSerialized) error {
	stub.called = true
	return nil
}

func (stub *serializedStub) Delete(ctx context.Context, user *bootstrap.User, item model.UserItemsSerialized) error {
	stub.called = true
	return nil
}

func (stub *serializedStub) DeleteAll(ctx context.Context, user *bootstrap.User) error {
	stub.called = true
	return nil
}

func (stub *categoriesStub) GetByUser(ctx context.Context, user *bootstrap.User) ([]model.UserItemCategory, error) {
	stub.called = true
	return nil, nil
}

func (stub *categoriesStub) Upsert(ctx context.Context, user *bootstrap.User, category model.UserItemCategory) error {
	stub.called = true
	return nil
}

func (stub *categoriesStub) Delete(ctx context.Context, user *bootstrap.User, category model.UserItemCategory) error {
	stub.called = true
	return nil
}

func (stub *categoriesStub) DeleteAll(ctx context.Context, user *bootstrap.User) error {
	stub.called = true
	return nil
}

func (stub *itemsStub) GetByCategory(ctx context.Context, user *bootstrap.User, category model.UserItemCategory) ([]model.UserItemsCategorized, error) {
	stub.called = true
	return nil, nil
}

func (stub *itemsStub) GetByUser(ctx context.Context, user *bootstrap.User) ([]model.UserItemsCategorized, error) {
	stub.called = true
	return nil, nil
}

func (stub *itemsStub) Upsert(ctx context.Context, user *bootstrap.User, item model.UserItemsCategorized) error {
	stub.called = true
	return nil
}

func (stub *itemsStub) UpsertBatch(ctx context.Context, user *bootstrap.User, items []model.UserItemsCategorized) error {
	stub.called = true
	return nil
}

func (stub *itemsStub) Delete(ctx context.Context, user *bootstrap.User, item model.UserItemsCategorized) error {
	stub.called = true
	return nil
}

func (stub *itemsStub) DeleteAll(ctx context.Context, user *bootstrap.User) error {
	stub.called = true
	return nil
}

func (stub *imagesStub) Upload(ctx context.Context, user *bootstrap.User, itemID string, tableType string, imageData []byte) (string, error) {
	stub.called = true
	return "", nil
}

func (stub *imagesStub) Delete(ctx context.Context, user *bootstrap.User, itemID string, tableType string) error {
	stub.called = true
	return nil
}

func (stub *imagesStub) Get(ctx context.Context, user *bootstrap.User, itemID string, tableType string) ([]byte, string, error) {
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

	require.NoError(t, service.UpsertQuickSaveItemByUser(context.Background(), &bootstrap.User{}, model.UserItemsQuick{}))
	require.NoError(t, service.UpsertSerializedSaveItemByUser(context.Background(), &bootstrap.User{}, model.UserItemsSerialized{}))
	require.NoError(t, service.UpsertItemCategoryByUser(context.Background(), &bootstrap.User{}, model.UserItemCategory{}))
	require.NoError(t, service.UpsertCategorizedItemByUser(context.Background(), &bootstrap.User{}, model.UserItemsCategorized{}))
	require.NoError(t, service.DeleteItemImage(context.Background(), &bootstrap.User{}, "id", "quick"))

	require.True(t, quick.called)
	require.True(t, serialized.called)
	require.True(t, categories.called)
	require.True(t, items.called)
	require.True(t, images.called)
}
