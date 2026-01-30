package user_saves_test

import (
	"net/http"
	"testing"
	"time"

	"miltechserver/.gen/miltech_ng/public/model"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestUserSavesLifecycle(t *testing.T) {
	clearUserSavesTables(t, testDB)
	router := newTestRouter(t)
	userID := "user-lifecycle"
	ensureUser(t, testDB, userID)

	now := time.Now().UTC()
	comment := ""
	image := "https://example.com/img.jpg"

	category := model.UserItemCategory{
		ID:          uuid.New().String(),
		UserUID:     userID,
		Name:        "Cat",
		Comment:     &comment,
		Image:       &image,
		LastUpdated: &now,
	}

	catResp := doJSONRequest(t, router, http.MethodPut, "/api/v1/auth/user/saves/item_category", category, userID)
	require.Equal(t, http.StatusOK, catResp.Code)

	itemName := "Item"
	quantity := int32(1)
	equipModel := "Model"
	uoc := "UOC"
	nickname := ""
	categorized := model.UserItemsCategorized{
		ID:          uuid.New().String(),
		UserID:      userID,
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

	itemResp := doJSONRequest(t, router, http.MethodPut, "/api/v1/auth/user/saves/categorized_items/add", categorized, userID)
	require.Equal(t, http.StatusOK, itemResp.Code)

	deleteResp := doJSONRequest(t, router, http.MethodDelete, "/api/v1/auth/user/saves/item_category", category, userID)
	require.Equal(t, http.StatusOK, deleteResp.Code)

	getCatResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/user/saves/item_category", nil, userID)
	require.Equal(t, http.StatusOK, getCatResp.Code)
	catPayload := decodeStandardResponse(t, getCatResp.Body)
	catItems := decodeSlice(t, catPayload.Data)
	require.Len(t, catItems, 0)

	itemsResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/user/saves/categorized_items", nil, userID)
	require.Equal(t, http.StatusOK, itemsResp.Code)
	itemsPayload := decodeStandardResponse(t, itemsResp.Body)
	items := decodeSlice(t, itemsPayload.Data)
	require.Len(t, items, 0)
}
