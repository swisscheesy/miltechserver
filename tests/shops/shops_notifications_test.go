package shops_test

import (
	"net/http"
	"net/http/httptest"
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

func TestVehicleNotificationAttachedShopList(t *testing.T) {
	t.Run("create and direct reads include attachment", func(t *testing.T) {
		clearShopTables(t, testDB)
		ensureUser(t, testDB, "user-1")

		router := newTestRouter(t)

		shopID := createShop(t, router, "user-1", "Attached Notification Shop")
		vehicleID := createVehicle(t, router, "user-1", shopID)
		listID := createList(t, router, "user-1", shopID)

		createBody := map[string]interface{}{
			"shop_id":            shopID,
			"vehicle_id":         vehicleID,
			"title":              "Attached PM",
			"description":        "desc",
			"type":               "PM",
			"attached_shop_list": listID,
		}

		createResp := doJSONRequest(t, router, http.MethodPost, "/api/v1/auth/shops/vehicles/notifications", createBody, "user-1")
		require.Equal(t, http.StatusCreated, createResp.Code)

		created := decodeStandardResponse(t, createResp.Body)
		notificationData := decodeMap(t, created.Data)
		require.Equal(t, listID, notificationData["attached_shop_list"])

		notificationID, ok := notificationData["id"].(string)
		require.True(t, ok)
		require.NotEmpty(t, notificationID)

		vehicleNotificationsResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/vehicles/"+vehicleID+"/notifications", nil, "user-1")
		require.Equal(t, http.StatusOK, vehicleNotificationsResp.Code)
		assertNotificationAttachmentInListResponse(t, vehicleNotificationsResp, notificationID, listID)

		withItemsResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/vehicles/"+vehicleID+"/notifications-with-items", nil, "user-1")
		require.Equal(t, http.StatusOK, withItemsResp.Code)
		assertNotificationAttachmentInWithItemsResponse(t, withItemsResp, notificationID, listID)

		shopNotificationsResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/"+shopID+"/notifications", nil, "user-1")
		require.Equal(t, http.StatusOK, shopNotificationsResp.Code)
		assertNotificationAttachmentInListResponse(t, shopNotificationsResp, notificationID, listID)

		getByIDResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/vehicles/notifications/"+notificationID, nil, "user-1")
		require.Equal(t, http.StatusOK, getByIDResp.Code)
		assertNotificationAttachmentInResponse(t, getByIDResp, listID)
	})

	t.Run("update sets preserves and clears attachment", func(t *testing.T) {
		clearShopTables(t, testDB)
		ensureUser(t, testDB, "user-1")

		router := newTestRouter(t)

		shopID := createShop(t, router, "user-1", "Update Attached Notification Shop")
		vehicleID := createVehicle(t, router, "user-1", shopID)
		listID := createList(t, router, "user-1", shopID)
		notificationID := createNotification(t, router, "user-1", shopID, vehicleID, "Unattached PM")

		updateBody := map[string]interface{}{
			"notification_id":    notificationID,
			"title":              "Attached PM",
			"description":        "desc",
			"type":               "PM",
			"completed":          false,
			"attached_shop_list": listID,
		}

		updateResp := doJSONRequest(t, router, http.MethodPut, "/api/v1/auth/shops/vehicles/notifications", updateBody, "user-1")
		require.Equal(t, http.StatusOK, updateResp.Code)

		getByIDResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/vehicles/notifications/"+notificationID, nil, "user-1")
		require.Equal(t, http.StatusOK, getByIDResp.Code)
		assertNotificationAttachmentInResponse(t, getByIDResp, listID)

		omittedAttachmentUpdate := map[string]interface{}{
			"notification_id": notificationID,
			"title":           "Still Attached PM",
			"description":     "desc",
			"type":            "PM",
			"completed":       false,
		}

		omitResp := doJSONRequest(t, router, http.MethodPut, "/api/v1/auth/shops/vehicles/notifications", omittedAttachmentUpdate, "user-1")
		require.Equal(t, http.StatusOK, omitResp.Code)

		getAfterOmitResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/vehicles/notifications/"+notificationID, nil, "user-1")
		require.Equal(t, http.StatusOK, getAfterOmitResp.Code)
		assertNotificationAttachmentInResponse(t, getAfterOmitResp, listID)

		clearAttachmentUpdate := map[string]interface{}{
			"notification_id":    notificationID,
			"title":              "Cleared PM",
			"description":        "desc",
			"type":               "PM",
			"completed":          false,
			"attached_shop_list": nil,
		}

		clearResp := doJSONRequest(t, router, http.MethodPut, "/api/v1/auth/shops/vehicles/notifications", clearAttachmentUpdate, "user-1")
		require.Equal(t, http.StatusOK, clearResp.Code)

		getAfterClearResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/vehicles/notifications/"+notificationID, nil, "user-1")
		require.Equal(t, http.StatusOK, getAfterClearResp.Code)

		clearedNotificationData := decodeMap(t, decodeStandardResponse(t, getAfterClearResp.Body).Data)
		require.Nil(t, clearedNotificationData["attached_shop_list"])
	})

	t.Run("cross-shop list is rejected and attachment remains unchanged", func(t *testing.T) {
		clearShopTables(t, testDB)
		ensureUser(t, testDB, "user-1")

		router := newTestRouter(t)

		shopID := createShop(t, router, "user-1", "First Notification Shop")
		vehicleID := createVehicle(t, router, "user-1", shopID)
		listID := createList(t, router, "user-1", shopID)
		notificationID := createNotification(t, router, "user-1", shopID, vehicleID, "Original PM")

		_, err := testDB.Exec(
			"UPDATE shop_vehicle_notifications SET attached_shop_list = $1 WHERE id = $2",
			listID,
			notificationID,
		)
		require.NoError(t, err)

		secondShopID := createShop(t, router, "user-1", "Second Notification Shop")
		secondListID := createList(t, router, "user-1", secondShopID)

		crossShopUpdate := map[string]interface{}{
			"notification_id":    notificationID,
			"title":              "Cross Shop PM",
			"description":        "desc",
			"type":               "PM",
			"completed":          false,
			"attached_shop_list": secondListID,
		}

		crossShopResp := doJSONRequest(t, router, http.MethodPut, "/api/v1/auth/shops/vehicles/notifications", crossShopUpdate, "user-1")
		require.NotContains(t, []int{http.StatusOK, http.StatusCreated, http.StatusNoContent}, crossShopResp.Code)
		// Current shared error handling maps this service validation error to 500.
		require.Equal(t, http.StatusInternalServerError, crossShopResp.Code)
		errorResponse := decodeStandardResponse(t, crossShopResp.Body)
		require.Equal(t, "invalid attached_shop_list", errorResponse.Message)

		getByIDResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/vehicles/notifications/"+notificationID, nil, "user-1")
		require.Equal(t, http.StatusOK, getByIDResp.Code)
		assertNotificationAttachmentInResponse(t, getByIDResp, listID)
	})
}

func assertNotificationAttachmentInListResponse(t *testing.T, resp *httptest.ResponseRecorder, notificationID string, listID string) {
	t.Helper()

	notifications := decodeStandardResponse(t, resp.Body)
	notificationsList := decodeSlice(t, notifications.Data)
	notificationData := findNotificationData(t, notificationsList, notificationID)
	require.Equal(t, listID, notificationData["attached_shop_list"])
}

func assertNotificationAttachmentInWithItemsResponse(t *testing.T, resp *httptest.ResponseRecorder, notificationID string, listID string) {
	t.Helper()

	notifications := decodeStandardResponse(t, resp.Body)
	notificationsList := decodeSlice(t, notifications.Data)

	for _, item := range notificationsList {
		withItemsData := mapFromDecodedJSON(t, item)

		notificationData := mapFromDecodedJSON(t, withItemsData["notification"])
		if notificationData["id"] == notificationID {
			require.Equal(t, listID, notificationData["attached_shop_list"])
			return
		}
	}

	require.Fail(t, "notification not found", "notification_id=%s", notificationID)
}

func assertNotificationAttachmentInResponse(t *testing.T, resp *httptest.ResponseRecorder, listID string) {
	t.Helper()

	notificationData := decodeMap(t, decodeStandardResponse(t, resp.Body).Data)
	require.Equal(t, listID, notificationData["attached_shop_list"])
}

func findNotificationData(t *testing.T, notifications []interface{}, notificationID string) map[string]interface{} {
	t.Helper()

	for _, item := range notifications {
		notificationData := mapFromDecodedJSON(t, item)
		if notificationData["id"] == notificationID {
			return notificationData
		}
	}

	require.Fail(t, "notification not found", "notification_id=%s", notificationID)
	return nil
}

func mapFromDecodedJSON(t *testing.T, value interface{}) map[string]interface{} {
	t.Helper()

	data, ok := value.(map[string]interface{})
	require.True(t, ok)
	return data
}

// Helper functions live in helpers_test.go.
