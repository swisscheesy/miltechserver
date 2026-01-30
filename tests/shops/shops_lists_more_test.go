package shops_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestListUpdateDeleteAndBatchItems(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "user-1")

	router := newTestRouter(t)

	shopID := createShop(t, router, "user-1", "Batch Shop")

	createListBody := map[string]interface{}{
		"shop_id":     shopID,
		"description": "Initial list",
	}

	createListResp := doJSONRequest(t, router, http.MethodPost, "/api/v1/auth/shops/lists", createListBody, "user-1")
	require.Equal(t, http.StatusCreated, createListResp.Code)

	listResp := decodeStandardResponse(t, createListResp.Body)
	listData := decodeMap(t, listResp.Data)
	listID, ok := listData["id"].(string)
	require.True(t, ok)

	updateListBody := map[string]interface{}{
		"list_id":     listID,
		"description": "Updated list",
	}

	updateListResp := doJSONRequest(t, router, http.MethodPut, "/api/v1/auth/shops/lists", updateListBody, "user-1")
	require.Equal(t, http.StatusOK, updateListResp.Code)

	batchItemsBody := map[string]interface{}{
		"list_id": listID,
		"items": []map[string]interface{}{
			{
				"list_id":         listID,
				"niin":            "3333-33-333-3333",
				"nomenclature":    "Bulk 1",
				"quantity":        1,
				"unit_of_measure": "ea",
			},
			{
				"list_id":         listID,
				"niin":            "4444-44-444-4444",
				"nomenclature":    "Bulk 2",
				"quantity":        2,
				"unit_of_measure": "ea",
			},
		},
	}

	addBatchResp := doJSONRequest(t, router, http.MethodPost, "/api/v1/auth/shops/lists/items/bulk", batchItemsBody, "user-1")
	require.Equal(t, http.StatusCreated, addBatchResp.Code)

	getItemsResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/lists/"+listID+"/items", nil, "user-1")
	require.Equal(t, http.StatusOK, getItemsResp.Code)

	items := decodeStandardResponse(t, getItemsResp.Body)
	itemsList := decodeSlice(t, items.Data)
	require.Len(t, itemsList, 2)

	itemIDs := []string{}
	for _, item := range itemsList {
		itemMap, ok := item.(map[string]interface{})
		require.True(t, ok)
		id, ok := itemMap["id"].(string)
		require.True(t, ok)
		itemIDs = append(itemIDs, id)
	}

	removeBatchBody := map[string]interface{}{
		"item_ids": itemIDs,
	}

	removeBatchResp := doJSONRequest(t, router, http.MethodDelete, "/api/v1/auth/shops/lists/items/bulk", removeBatchBody, "user-1")
	require.Equal(t, http.StatusOK, removeBatchResp.Code)

	deleteListBody := map[string]interface{}{
		"list_id": listID,
	}

	deleteListResp := doJSONRequest(t, router, http.MethodDelete, "/api/v1/auth/shops/lists", deleteListBody, "user-1")
	require.Equal(t, http.StatusOK, deleteListResp.Code)
}
