package shops_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestListsAndItems(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "user-1")

	router := newTestRouter(t)

	createShopBody := map[string]interface{}{
		"name":    "List Shop",
		"details": "List flow",
	}

	createShopResp := doJSONRequest(t, router, http.MethodPost, "/api/v1/auth/shops", createShopBody, "user-1")
	require.Equal(t, http.StatusCreated, createShopResp.Code)

	created := decodeStandardResponse(t, createShopResp.Body)
	shopData := decodeMap(t, created.Data)
	shopID, ok := shopData["id"].(string)
	require.True(t, ok)

	createListBody := map[string]interface{}{
		"shop_id":     shopID,
		"description": "Parts list",
	}

	createListResp := doJSONRequest(t, router, http.MethodPost, "/api/v1/auth/shops/lists", createListBody, "user-1")
	require.Equal(t, http.StatusCreated, createListResp.Code)

	listResp := decodeStandardResponse(t, createListResp.Body)
	listData := decodeMap(t, listResp.Data)
	listID, ok := listData["id"].(string)
	require.True(t, ok)

	addItemBody := map[string]interface{}{
		"list_id":         listID,
		"niin":            "0000-00-000-0000",
		"nomenclature":    "Test Part",
		"quantity":        2,
		"unit_of_measure": "ea",
	}

	addItemResp := doJSONRequest(t, router, http.MethodPost, "/api/v1/auth/shops/lists/items", addItemBody, "user-1")
	require.Equal(t, http.StatusCreated, addItemResp.Code)

	getItemsResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/lists/"+listID+"/items", nil, "user-1")
	require.Equal(t, http.StatusOK, getItemsResp.Code)

	items := decodeStandardResponse(t, getItemsResp.Body)
	itemsList := decodeSlice(t, items.Data)
	require.Len(t, itemsList, 1)
}
