package categories

import (
	"errors"
	"testing"

	"miltechserver/.gen/miltech_ng/public/model"
	categoryitems "miltechserver/api/user_saves/categories/items"
	"miltechserver/api/user_saves/images"
	"miltechserver/api/user_saves/shared"
	"miltechserver/bootstrap"

	"github.com/stretchr/testify/require"
)

type categoriesRepoStub struct {
	deleteCalled bool
	deleteAll    bool
	getErr       error
	categories   []model.UserItemCategory
}

func (repo *categoriesRepoStub) GetByUser(user *bootstrap.User) ([]model.UserItemCategory, error) {
	if repo.getErr != nil {
		return nil, repo.getErr
	}
	return repo.categories, nil
}

func (repo *categoriesRepoStub) Upsert(user *bootstrap.User, category model.UserItemCategory) error {
	return nil
}

func (repo *categoriesRepoStub) Delete(user *bootstrap.User, category model.UserItemCategory) error {
	repo.deleteCalled = true
	return nil
}

func (repo *categoriesRepoStub) DeleteAll(user *bootstrap.User) error {
	repo.deleteAll = true
	return nil
}

type categoryItemsStub struct {
	itemsByCategory  []model.UserItemsCategorized
	itemsByUser      []model.UserItemsCategorized
	getByCategoryErr error
	getByUserErr     error
}

func (repo *categoryItemsStub) GetByCategory(user *bootstrap.User, category model.UserItemCategory) ([]model.UserItemsCategorized, error) {
	if repo.getByCategoryErr != nil {
		return nil, repo.getByCategoryErr
	}
	return repo.itemsByCategory, nil
}

func (repo *categoryItemsStub) GetByUser(user *bootstrap.User) ([]model.UserItemsCategorized, error) {
	if repo.getByUserErr != nil {
		return nil, repo.getByUserErr
	}
	return repo.itemsByUser, nil
}

func (repo *categoryItemsStub) Upsert(user *bootstrap.User, item model.UserItemsCategorized) error {
	return nil
}

func (repo *categoryItemsStub) UpsertBatch(user *bootstrap.User, items []model.UserItemsCategorized) error {
	return nil
}

func (repo *categoryItemsStub) Delete(user *bootstrap.User, item model.UserItemsCategorized) error {
	return nil
}

func (repo *categoryItemsStub) DeleteAll(user *bootstrap.User) error {
	return nil
}

var _ categoryitems.Repository = (*categoryItemsStub)(nil)

type categoriesImagesStub struct {
	deletedIDs []string
}

func (repo *categoriesImagesStub) Upload(user *bootstrap.User, itemID string, tableType string, imageData []byte) (string, error) {
	return "", nil
}

func (repo *categoriesImagesStub) Delete(user *bootstrap.User, itemID string, tableType string) error {
	repo.deletedIDs = append(repo.deletedIDs, itemID)
	return nil
}

func (repo *categoriesImagesStub) Get(user *bootstrap.User, itemID string, tableType string) ([]byte, string, error) {
	return nil, "", errors.New("not implemented")
}

var _ images.Repository = (*categoriesImagesStub)(nil)

func TestServiceImplGetByUserRequiresUser(t *testing.T) {
	service := NewService(&categoriesRepoStub{}, &categoryItemsStub{}, &categoriesImagesStub{})

	_, err := service.GetByUser(nil)
	require.ErrorIs(t, err, shared.ErrUserNotFound)
}

func TestServiceImplDeleteCallsRepoOnItemsError(t *testing.T) {
	repo := &categoriesRepoStub{}
	itemsRepo := &categoryItemsStub{getByCategoryErr: errors.New("failed")}
	service := NewService(repo, itemsRepo, &categoriesImagesStub{})

	category := model.UserItemCategory{ID: "cat"}
	err := service.Delete(&bootstrap.User{UserID: "user"}, category)
	require.NoError(t, err)
	require.True(t, repo.deleteCalled)
}

func TestServiceImplDeleteAllDeletesImages(t *testing.T) {
	imageURL := "https://example.com/test.jpg"
	itemsRepo := &categoryItemsStub{itemsByUser: []model.UserItemsCategorized{{ID: "item", Image: &imageURL}}}
	repo := &categoriesRepoStub{categories: []model.UserItemCategory{{ID: "cat", Image: &imageURL}}}
	imagesRepo := &categoriesImagesStub{}
	service := NewService(repo, itemsRepo, imagesRepo)

	err := service.DeleteAll(&bootstrap.User{UserID: "user"})
	require.NoError(t, err)
	require.True(t, repo.deleteAll)
	require.ElementsMatch(t, []string{"item", "cat"}, imagesRepo.deletedIDs)
}
