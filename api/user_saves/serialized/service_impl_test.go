package serialized

import (
	"errors"
	"testing"

	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/user_saves/images"
	"miltechserver/api/user_saves/shared"
	"miltechserver/bootstrap"

	"github.com/stretchr/testify/require"
)

type serializedRepoStub struct {
	items     []model.UserItemsSerialized
	getErr    error
	deleteAll bool
}

func (repo *serializedRepoStub) GetByUser(user *bootstrap.User) ([]model.UserItemsSerialized, error) {
	if repo.getErr != nil {
		return nil, repo.getErr
	}
	return repo.items, nil
}

func (repo *serializedRepoStub) Upsert(user *bootstrap.User, item model.UserItemsSerialized) error {
	return nil
}

func (repo *serializedRepoStub) UpsertBatch(user *bootstrap.User, items []model.UserItemsSerialized) error {
	return nil
}

func (repo *serializedRepoStub) Delete(user *bootstrap.User, item model.UserItemsSerialized) error {
	return nil
}

func (repo *serializedRepoStub) DeleteAll(user *bootstrap.User) error {
	repo.deleteAll = true
	return nil
}

type serializedImagesStub struct {
	deletedIDs []string
}

func (repo *serializedImagesStub) Upload(user *bootstrap.User, itemID string, tableType string, imageData []byte) (string, error) {
	return "", nil
}

func (repo *serializedImagesStub) Delete(user *bootstrap.User, itemID string, tableType string) error {
	repo.deletedIDs = append(repo.deletedIDs, itemID)
	return nil
}

func (repo *serializedImagesStub) Get(user *bootstrap.User, itemID string, tableType string) ([]byte, string, error) {
	return nil, "", errors.New("not implemented")
}

var _ images.Repository = (*serializedImagesStub)(nil)

func TestServiceImplGetByUserRequiresUser(t *testing.T) {
	service := NewService(&serializedRepoStub{}, &serializedImagesStub{})

	_, err := service.GetByUser(nil)
	require.ErrorIs(t, err, shared.ErrUserNotFound)
}

func TestServiceImplDeleteAllDeletesImages(t *testing.T) {
	imageURL := "https://example.com/test.jpg"
	items := []model.UserItemsSerialized{
		{ID: "one", Image: &imageURL},
		{ID: "two", Image: nil},
	}

	repo := &serializedRepoStub{items: items}
	imagesRepo := &serializedImagesStub{}
	service := NewService(repo, imagesRepo)

	err := service.DeleteAll(&bootstrap.User{UserID: "user"})
	require.NoError(t, err)
	require.True(t, repo.deleteAll)
	require.Equal(t, []string{"one"}, imagesRepo.deletedIDs)
}

func TestServiceImplDeleteAllContinuesOnGetError(t *testing.T) {
	repo := &serializedRepoStub{getErr: errors.New("failed")}
	imagesRepo := &serializedImagesStub{}
	service := NewService(repo, imagesRepo)

	err := service.DeleteAll(&bootstrap.User{UserID: "user"})
	require.NoError(t, err)
	require.True(t, repo.deleteAll)
}
