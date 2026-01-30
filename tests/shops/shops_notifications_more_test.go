package shops_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNotificationDeleteAndBulkItems(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "user-1")

	router := newTestRouter(t)

	shopID := createShop(t, router, "user-1", "Bulk Notify")
	vehicleID := createVehicle(t, router, "user-1", shopID)
	notificationID := createNotification(t, router, "user-1", shopID, vehicleID, "Bulk Items")

	bulkItemsBody := map[string]interface{}{
		"notification_id": notificationID,
		"items": []map[string]interface{}{
			{
				"notification_id": notificationID,
				"niin":            "5555-55-555-5555",
				"nomenclature":    "Bulk Item 1",
				"quantity":        1,
			},
			{
				"notification_id": notificationID,
				"niin":            "6666-66-666-6666",
				"nomenclature":    "Bulk Item 2",
				"quantity":        2,
			},
		},
	}

	addBulkResp := doJSONRequest(t, router, http.MethodPost, "/api/v1/auth/shops/notifications/items/bulk", bulkItemsBody, "user-1")
	require.Equal(t, http.StatusCreated, addBulkResp.Code)

	getItemsResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/notifications/"+notificationID+"/items", nil, "user-1")
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

	removeBulkBody := map[string]interface{}{
		"item_ids": itemIDs,
	}

	removeBulkResp := doJSONRequest(t, router, http.MethodDelete, "/api/v1/auth/shops/notifications/items/bulk", removeBulkBody, "user-1")
	require.Equal(t, http.StatusOK, removeBulkResp.Code)

	deleteNotificationResp := doJSONRequest(t, router, http.MethodDelete, "/api/v1/auth/shops/vehicles/notifications/"+notificationID, nil, "user-1")
	require.Equal(t, http.StatusOK, deleteNotificationResp.Code)
}
