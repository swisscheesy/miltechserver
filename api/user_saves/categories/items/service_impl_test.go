package items

import (
	"errors"
	"testing"

	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/user_saves/images"
	"miltechserver/api/user_saves/shared"
	"miltechserver/bootstrap"

	"github.com/stretchr/testify/require"
)

type itemsRepoStub struct {
	items     []model.UserItemsCategorized
	getErr    error
	deleteAll bool
}

func (repo *itemsRepoStub) GetByCategory(user *bootstrap.User, category model.UserItemCategory) ([]model.UserItemsCategorized, error) {
	return repo.items, repo.getErr
}

func (repo *itemsRepoStub) GetByUser(user *bootstrap.User) ([]model.UserItemsCategorized, error) {
	if repo.getErr != nil {
		return nil, repo.getErr
	}
	return repo.items, nil
}

func (repo *itemsRepoStub) Upsert(user *bootstrap.User, item model.UserItemsCategorized) error {
	return nil
}

func (repo *itemsRepoStub) UpsertBatch(user *bootstrap.User, items []model.UserItemsCategorized) error {
	return nil
}

func (repo *itemsRepoStub) Delete(user *bootstrap.User, item model.UserItemsCategorized) error {
	return nil
}

func (repo *itemsRepoStub) DeleteAll(user *bootstrap.User) error {
	repo.deleteAll = true
	return nil
}

type itemsImagesStub struct {
	deletedIDs []string
}

func (repo *itemsImagesStub) Upload(user *bootstrap.User, itemID string, tableType string, imageData []byte) (string, error) {
	return "", nil
}

func (repo *itemsImagesStub) Delete(user *bootstrap.User, itemID string, tableType string) error {
	repo.deletedIDs = append(repo.deletedIDs, itemID)
	return nil
}

func (repo *itemsImagesStub) Get(user *bootstrap.User, itemID string, tableType string) ([]byte, string, error) {
	return nil, "", errors.New("not implemented")
}

var _ images.Repository = (*itemsImagesStub)(nil)

func TestServiceImplGetByUserRequiresUser(t *testing.T) {
	service := NewService(&itemsRepoStub{}, &itemsImagesStub{})

	_, err := service.GetByUser(nil)
	require.ErrorIs(t, err, shared.ErrUserNotFound)
}

func TestServiceImplDeleteAllDeletesImages(t *testing.T) {
	imageURL := "https://example.com/test.jpg"
	repo := &itemsRepoStub{items: []model.UserItemsCategorized{{ID: "one", Image: &imageURL}, {ID: "two"}}}
	imagesRepo := &itemsImagesStub{}
	service := NewService(repo, imagesRepo)

	err := service.DeleteAll(&bootstrap.User{UserID: "user"})
	require.NoError(t, err)
	require.True(t, repo.deleteAll)
	require.Equal(t, []string{"one"}, imagesRepo.deletedIDs)
}
