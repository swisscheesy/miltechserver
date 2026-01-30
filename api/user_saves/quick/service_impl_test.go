package quick

import (
	"errors"
	"testing"

	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/user_saves/images"
	"miltechserver/api/user_saves/shared"
	"miltechserver/bootstrap"

	"github.com/stretchr/testify/require"
)

type quickRepoStub struct {
	items     []model.UserItemsQuick
	getErr    error
	deleteAll bool
}

func (repo *quickRepoStub) GetByUser(user *bootstrap.User) ([]model.UserItemsQuick, error) {
	if repo.getErr != nil {
		return nil, repo.getErr
	}
	return repo.items, nil
}

func (repo *quickRepoStub) Upsert(user *bootstrap.User, item model.UserItemsQuick) error {
	return nil
}

func (repo *quickRepoStub) UpsertBatch(user *bootstrap.User, items []model.UserItemsQuick) error {
	return nil
}

func (repo *quickRepoStub) Delete(user *bootstrap.User, item model.UserItemsQuick) error {
	return nil
}

func (repo *quickRepoStub) DeleteAll(user *bootstrap.User) error {
	repo.deleteAll = true
	return nil
}

type imagesRepoStub struct {
	deletedIDs []string
}

func (repo *imagesRepoStub) Upload(user *bootstrap.User, itemID string, tableType string, imageData []byte) (string, error) {
	return "", nil
}

func (repo *imagesRepoStub) Delete(user *bootstrap.User, itemID string, tableType string) error {
	repo.deletedIDs = append(repo.deletedIDs, itemID)
	return nil
}

func (repo *imagesRepoStub) Get(user *bootstrap.User, itemID string, tableType string) ([]byte, string, error) {
	return nil, "", errors.New("not implemented")
}

var _ images.Repository = (*imagesRepoStub)(nil)

func TestServiceImplDeleteAllRequiresUser(t *testing.T) {
	service := NewService(&quickRepoStub{}, &imagesRepoStub{})

	_, err := service.GetByUser(nil)
	require.ErrorIs(t, err, shared.ErrUserNotFound)
}

func TestServiceImplDeleteAllDeletesImages(t *testing.T) {
	imageURL := "https://example.com/test.jpg"
	items := []model.UserItemsQuick{
		{ID: "one", Image: &imageURL},
		{ID: "two", Image: nil},
	}

	repo := &quickRepoStub{items: items}
	imagesRepo := &imagesRepoStub{}
	service := NewService(repo, imagesRepo)

	err := service.DeleteAll(&bootstrap.User{UserID: "user"})
	require.NoError(t, err)
	require.True(t, repo.deleteAll)
	require.Equal(t, []string{"one"}, imagesRepo.deletedIDs)
}

func TestServiceImplDeleteAllContinuesOnGetError(t *testing.T) {
	repo := &quickRepoStub{getErr: errors.New("failed")}
	imagesRepo := &imagesRepoStub{}
	service := NewService(repo, imagesRepo)

	err := service.DeleteAll(&bootstrap.User{UserID: "user"})
	require.NoError(t, err)
	require.True(t, repo.deleteAll)
}
