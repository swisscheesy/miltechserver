package user_saves_test

import (
	"net/http"
	"testing"
	"time"

	"miltechserver/.gen/miltech_ng/public/model"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestQuickHandlers(t *testing.T) {
	clearUserSavesTables(t, testDB)
	router := newTestRouter(t)
	userID := "user-quick"
	ensureUser(t, testDB, userID)

	itemID := uuid.New().String()
	now := time.Now().UTC()
	image := ""
	comment := ""
	nickname := ""
	itemName := "Widget"

	quick := model.UserItemsQuick{
		ID:          itemID,
		UserID:      userID,
		Niin:        "N-1",
		ItemName:    &itemName,
		Image:       &image,
		ItemComment: &comment,
		SaveTime:    &now,
		LastUpdated: &now,
		Nickname:    &nickname,
	}

	resp := doJSONRequest(t, router, http.MethodPut, "/api/v1/auth/user/saves/quick_items/add", quick, userID)
	require.Equal(t, http.StatusOK, resp.Code)

	getResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/user/saves/quick_items", nil, userID)
	require.Equal(t, http.StatusOK, getResp.Code)

	payload := decodeStandardResponse(t, getResp.Body)
	items := decodeSlice(t, payload.Data)
	require.Len(t, items, 1)
}

func TestSerializedHandlers(t *testing.T) {
	clearUserSavesTables(t, testDB)
	router := newTestRouter(t)
	userID := "user-serialized"
	ensureUser(t, testDB, userID)

	itemID := uuid.New().String()
	now := time.Now().UTC()
	image := ""
	comment := ""
	nickname := ""
	itemName := "Serialized"
	serial := "SER-1"

	_, err := testDB.Exec(
		`INSERT INTO user_items_serialized (id, user_id, niin, item_name, serial, image, item_comment, save_time, last_updated, nickname)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		itemID,
		userID,
		"S-1",
		itemName,
		serial,
		image,
		comment,
		now,
		now,
		nickname,
	)
	require.NoError(t, err)

	getResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/user/saves/serialized_items", nil, userID)
	require.Equal(t, http.StatusOK, getResp.Code)

	payload := decodeStandardResponse(t, getResp.Body)
	items := decodeSlice(t, payload.Data)
	require.Len(t, items, 1)
}

func TestCategoriesAndItemsHandlers(t *testing.T) {
	clearUserSavesTables(t, testDB)
	router := newTestRouter(t)
	userID := "user-category"
	ensureUser(t, testDB, userID)

	categoryID := uuid.New().String()
	now := time.Now().UTC()
	comment := ""
	image := ""

	category := model.UserItemCategory{
		ID:          categoryID,
		UserUID:     userID,
		Name:        "Cat",
		Comment:     &comment,
		Image:       &image,
		LastUpdated: &now,
	}

	catResp := doJSONRequest(t, router, http.MethodPut, "/api/v1/auth/user/saves/item_category", category, userID)
	require.Equal(t, http.StatusOK, catResp.Code)

	catGetResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/user/saves/item_category", nil, userID)
	require.Equal(t, http.StatusOK, catGetResp.Code)
	catPayload := decodeStandardResponse(t, catGetResp.Body)
	catItems := decodeSlice(t, catPayload.Data)
	require.Len(t, catItems, 1)

	itemID := uuid.New().String()
	itemName := "Item"
	quantity := int32(1)
	equipModel := "Model"
	uoc := "UOC"
	nickname := ""
	categorized := model.UserItemsCategorized{
		ID:          itemID,
		UserID:      userID,
		Niin:        "C-1",
		CategoryID:  categoryID,
		ItemName:    &itemName,
		Quantity:    &quantity,
		EquipModel:  &equipModel,
		Uoc:         &uoc,
		SaveTime:    &now,
		Image:       &image,
		LastUpdated: &now,
		Nickname:    &nickname,
	}

	itemResp := doJSONRequest(t, router, http.MethodPut, "/api/v1/auth/user/saves/categorized_items/add", categorized, userID)
	require.Equal(t, http.StatusOK, itemResp.Code)

	itemsResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/user/saves/categorized_items", nil, userID)
	require.Equal(t, http.StatusOK, itemsResp.Code)

	itemsPayload := decodeStandardResponse(t, itemsResp.Body)
	items := decodeSlice(t, itemsPayload.Data)
	require.Len(t, items, 1)

	byCategoryResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/user/saves/categorized_items/category", category, userID)
	require.Equal(t, http.StatusOK, byCategoryResp.Code)
}
