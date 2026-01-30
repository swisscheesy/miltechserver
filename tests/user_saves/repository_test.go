package user_saves_test

import (
	"testing"
	"time"

	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/user_saves/categories"
	categoryitems "miltechserver/api/user_saves/categories/items"
	"miltechserver/api/user_saves/quick"
	"miltechserver/api/user_saves/serialized"
	"miltechserver/bootstrap"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestQuickRepositoryCRUD(t *testing.T) {
	clearUserSavesTables(t, testDB)
	user := &bootstrap.User{UserID: "repo-quick"}
	ensureUser(t, testDB, user.UserID)

	now := time.Now().UTC()
	image := ""
	comment := ""
	nickname := ""
	itemName := "Quick"
	item := model.UserItemsQuick{
		ID:          uuid.New().String(),
		UserID:      user.UserID,
		Niin:        "Q-1",
		ItemName:    &itemName,
		Image:       &image,
		ItemComment: &comment,
		SaveTime:    &now,
		LastUpdated: &now,
		Nickname:    &nickname,
	}

	repo := quick.NewRepository(testDB)
	require.NoError(t, repo.Upsert(user, item))

	items, err := repo.GetByUser(user)
	require.NoError(t, err)
	require.Len(t, items, 1)

	require.NoError(t, repo.Delete(user, item))
	items, err = repo.GetByUser(user)
	require.NoError(t, err)
	require.Len(t, items, 0)
}

func TestSerializedRepositoryCRUD(t *testing.T) {
	clearUserSavesTables(t, testDB)
	user := &bootstrap.User{UserID: "repo-serialized"}
	ensureUser(t, testDB, user.UserID)

	now := time.Now().UTC()
	image := ""
	comment := ""
	nickname := ""
	itemName := "Serialized"
	serial := "SER-1"
	item := model.UserItemsSerialized{
		ID:          uuid.New().String(),
		UserID:      user.UserID,
		Niin:        "S-1",
		ItemName:    &itemName,
		Serial:      serial,
		Image:       &image,
		ItemComment: &comment,
		SaveTime:    &now,
		LastUpdated: &now,
		Nickname:    &nickname,
	}

	_, err := testDB.Exec(
		`INSERT INTO user_items_serialized (id, user_id, niin, item_name, serial, image, item_comment, save_time, last_updated, nickname)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		item.ID,
		item.UserID,
		item.Niin,
		itemName,
		serial,
		image,
		comment,
		now,
		now,
		nickname,
	)
	require.NoError(t, err)

	repo := serialized.NewRepository(testDB)

	items, err := repo.GetByUser(user)
	require.NoError(t, err)
	require.Len(t, items, 1)

	require.NoError(t, repo.Delete(user, item))
	items, err = repo.GetByUser(user)
	require.NoError(t, err)
	require.Len(t, items, 0)
}

func TestCategoriesRepositoryDeleteCascade(t *testing.T) {
	clearUserSavesTables(t, testDB)
	user := &bootstrap.User{UserID: "repo-cat"}
	ensureUser(t, testDB, user.UserID)

	now := time.Now().UTC()
	comment := ""
	image := ""
	category := model.UserItemCategory{
		ID:          uuid.New().String(),
		UserUID:     user.UserID,
		Name:        "Cat",
		Comment:     &comment,
		Image:       &image,
		LastUpdated: &now,
	}

	categoryRepo := categories.NewRepository(testDB)
	itemsRepo := categoryitems.NewRepository(testDB)
	require.NoError(t, categoryRepo.Upsert(user, category))

	itemName := "Item"
	quantity := int32(2)
	equipModel := "Model"
	uoc := "UOC"
	nickname := ""
	item := model.UserItemsCategorized{
		ID:          uuid.New().String(),
		UserID:      user.UserID,
		Niin:        "C-1",
		CategoryID:  category.ID,
		ItemName:    &itemName,
		Quantity:    &quantity,
		EquipModel:  &equipModel,
		Uoc:         &uoc,
		SaveTime:    &now,
		Image:       &image,
		LastUpdated: &now,
		Nickname:    &nickname,
	}
	require.NoError(t, itemsRepo.Upsert(user, item))

	require.NoError(t, categoryRepo.Delete(user, category))

	cats, err := categoryRepo.GetByUser(user)
	require.NoError(t, err)
	require.Len(t, cats, 0)

	items, err := itemsRepo.GetByUser(user)
	require.NoError(t, err)
	require.Len(t, items, 0)
}
