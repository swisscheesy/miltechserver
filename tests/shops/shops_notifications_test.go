package shops_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVehicleNotificationsAndChanges(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "user-1")

	router := newTestRouter(t)

	shopID := createShop(t, router, "user-1", "Notify Shop")
	vehicleID := createVehicle(t, router, "user-1", shopID)

	notificationID := createNotification(t, router, "user-1", shopID, vehicleID, "Initial PM")

	getNotificationsResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/vehicles/"+vehicleID+"/notifications", nil, "user-1")
	require.Equal(t, http.StatusOK, getNotificationsResp.Code)

	notifications := decodeStandardResponse(t, getNotificationsResp.Body)
	notificationsList := decodeSlice(t, notifications.Data)
	require.Len(t, notificationsList, 1)

	getWithItemsResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/vehicles/"+vehicleID+"/notifications-with-items", nil, "user-1")
	require.Equal(t, http.StatusOK, getWithItemsResp.Code)

	getByIDResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/vehicles/notifications/"+notificationID, nil, "user-1")
	require.Equal(t, http.StatusOK, getByIDResp.Code)

	changesResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/notifications/"+notificationID+"/changes", nil, "user-1")
	require.Equal(t, http.StatusOK, changesResp.Code)

	changes := decodeStandardResponse(t, changesResp.Body)
	changesList := decodeSlice(t, changes.Data)
	require.GreaterOrEqual(t, len(changesList), 1)

	updateBody := map[string]interface{}{
		"notification_id": notificationID,
		"title":           "Updated PM",
		"description":     "Updated details",
		"type":            "PM",
		"completed":       true,
	}

	updateResp := doJSONRequest(t, router, http.MethodPut, "/api/v1/auth/shops/vehicles/notifications", updateBody, "user-1")
	require.Equal(t, http.StatusOK, updateResp.Code)

	changesAfterUpdate := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/notifications/"+notificationID+"/changes", nil, "user-1")
	require.Equal(t, http.StatusOK, changesAfterUpdate.Code)

	updatedChanges := decodeStandardResponse(t, changesAfterUpdate.Body)
	updatedChangesList := decodeSlice(t, updatedChanges.Data)
	require.GreaterOrEqual(t, len(updatedChangesList), 2)
}

func TestNotificationItemsAndChanges(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "user-1")

	router := newTestRouter(t)

	shopID := createShop(t, router, "user-1", "Item Shop")
	vehicleID := createVehicle(t, router, "user-1", shopID)
	notificationID := createNotification(t, router, "user-1", shopID, vehicleID, "Item M1")

	itemBody := map[string]interface{}{
		"notification_id": notificationID,
		"niin":            "1111-11-111-1111",
		"nomenclature":    "Test Item",
		"quantity":        3,
	}

	addItemResp := doJSONRequest(t, router, http.MethodPost, "/api/v1/auth/shops/notifications/items", itemBody, "user-1")
	require.Equal(t, http.StatusCreated, addItemResp.Code)

	itemResponse := decodeStandardResponse(t, addItemResp.Body)
	itemData := decodeMap(t, itemResponse.Data)
	itemID, ok := itemData["id"].(string)
	require.True(t, ok)
	require.NotEmpty(t, itemID)

	getItemsResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/notifications/"+notificationID+"/items", nil, "user-1")
	require.Equal(t, http.StatusOK, getItemsResp.Code)

	items := decodeStandardResponse(t, getItemsResp.Body)
	itemsList := decodeSlice(t, items.Data)
	require.Len(t, itemsList, 1)

	changesResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/notifications/"+notificationID+"/changes", nil, "user-1")
	require.Equal(t, http.StatusOK, changesResp.Code)

	changes := decodeStandardResponse(t, changesResp.Body)
	changesList := decodeSlice(t, changes.Data)
	require.GreaterOrEqual(t, len(changesList), 2)

	removeResp := doJSONRequest(t, router, http.MethodDelete, "/api/v1/auth/shops/notifications/items/"+itemID, nil, "user-1")
	require.Equal(t, http.StatusOK, removeResp.Code)

	changesAfterRemoval := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/notifications/"+notificationID+"/changes", nil, "user-1")
	require.Equal(t, http.StatusOK, changesAfterRemoval.Code)

	updatedChanges := decodeStandardResponse(t, changesAfterRemoval.Body)
	updatedChangesList := decodeSlice(t, updatedChanges.Data)
	require.GreaterOrEqual(t, len(updatedChangesList), 3)
}

func TestShopAndVehicleChangeLists(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "user-1")

	router := newTestRouter(t)

	shopID := createShop(t, router, "user-1", "Change Shop")
	vehicleID := createVehicle(t, router, "user-1", shopID)
	notificationID := createNotification(t, router, "user-1", shopID, vehicleID, "Change PM")

	shopNotificationsResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/"+shopID+"/notifications", nil, "user-1")
	require.Equal(t, http.StatusOK, shopNotificationsResp.Code)

	shopNotifications := decodeStandardResponse(t, shopNotificationsResp.Body)
	shopNotificationsList := decodeSlice(t, shopNotifications.Data)
	require.Len(t, shopNotificationsList, 1)

	shopChangesResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/"+shopID+"/notifications/changes", nil, "user-1")
	require.Equal(t, http.StatusOK, shopChangesResp.Code)

	shopChanges := decodeStandardResponse(t, shopChangesResp.Body)
	shopChangesList := decodeSlice(t, shopChanges.Data)
	require.GreaterOrEqual(t, len(shopChangesList), 1)

	vehicleChangesResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/vehicles/"+vehicleID+"/notifications/changes", nil, "user-1")
	require.Equal(t, http.StatusOK, vehicleChangesResp.Code)

	vehicleChanges := decodeStandardResponse(t, vehicleChangesResp.Body)
	vehicleChangesList := decodeSlice(t, vehicleChanges.Data)
	require.GreaterOrEqual(t, len(vehicleChangesList), 1)

	// Ensure notification-specific changes work for the same notification.
	notificationChangesResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/notifications/"+notificationID+"/changes", nil, "user-1")
	require.Equal(t, http.StatusOK, notificationChangesResp.Code)

	notificationChanges := decodeStandardResponse(t, notificationChangesResp.Body)
	notificationChangesList := decodeSlice(t, notificationChanges.Data)
	require.GreaterOrEqual(t, len(notificationChangesList), 1)
}

// Helper functions live in helpers_test.go.
