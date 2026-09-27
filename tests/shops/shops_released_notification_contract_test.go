package shops_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"miltechserver/api/request"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// These field sets come from mobile source commit 62b5af53, not the current client.
func releasedCreateBody(shopID, vehicleID string) map[string]any {
	return map[string]any{
		"shop_id": shopID, "vehicle_id": vehicleID,
		"title": "PM due", "description": "Inspect", "type": "PM", "completed": false,
	}
}

func releasedBulkBody(shopID, notificationID, niin string) map[string]any {
	return map[string]any{
		"notification_id": notificationID,
		"items": []map[string]any{{
			"shop_id": shopID, "notification_id": notificationID,
			"niin": niin, "nomenclature": "Part", "quantity": 1,
			"save_time": "2026-09-26T12:00:00Z",
		}},
	}
}

func releasedEnvelope(t *testing.T, rec *httptest.ResponseRecorder, status int, message string) json.RawMessage {
	t.Helper()
	require.Equal(t, status, rec.Code, rec.Body.String())
	var body map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Contains(t, body, "status")
	require.Contains(t, body, "message")
	require.Contains(t, body, "data")
	var actualStatus int
	var actualMessage string
	require.NoError(t, json.Unmarshal(body["status"], &actualStatus))
	require.NoError(t, json.Unmarshal(body["message"], &actualMessage))
	require.Equal(t, status, actualStatus)
	require.Equal(t, message, actualMessage)
	return body["data"]
}

func releasedObject(t *testing.T, raw json.RawMessage) map[string]any {
	t.Helper()
	var data map[string]any
	require.NoError(t, json.Unmarshal(raw, &data))
	require.NotNil(t, data)
	return data
}

func releasedNotification(t *testing.T, raw json.RawMessage, shopID, vehicleID, title string) string {
	t.Helper()
	data := releasedObject(t, raw)
	id := releasedString(t, data, "id")
	require.NotEmpty(t, id)
	require.Equal(t, shopID, releasedString(t, data, "shop_id"))
	require.Equal(t, vehicleID, releasedString(t, data, "vehicle_id"))
	require.Equal(t, title, releasedString(t, data, "title"))
	require.Equal(t, "PM", releasedString(t, data, "type"))
	require.Equal(t, "Inspect", releasedString(t, data, "description"))
	require.Equal(t, false, data["completed"])
	releasedTime(t, data, "save_time")
	releasedTime(t, data, "last_updated")
	return id
}

func releasedString(t *testing.T, data map[string]any, field string) string {
	t.Helper()
	value, ok := data[field].(string)
	require.True(t, ok, "%s must be a string, got %T", field, data[field])
	return value
}

func releasedTime(t *testing.T, data map[string]any, field string) {
	t.Helper()
	value := releasedString(t, data, field)
	_, err := time.Parse(time.RFC3339Nano, value)
	require.NoError(t, err)
}

func releasedItems(t *testing.T, raw json.RawMessage, shopID, notificationID, niin string) string {
	t.Helper()
	var items []map[string]any
	require.NoError(t, json.Unmarshal(raw, &items))
	require.Len(t, items, 1)
	item := items[0]
	id := releasedString(t, item, "id")
	require.NotEmpty(t, id)
	require.Equal(t, shopID, releasedString(t, item, "shop_id"))
	require.Equal(t, notificationID, releasedString(t, item, "notification_id"))
	require.Equal(t, niin, releasedString(t, item, "niin"))
	require.Equal(t, "Part", releasedString(t, item, "nomenclature"))
	require.Equal(t, float64(1), item["quantity"])
	releasedTime(t, item, "save_time")
	return id
}

func TestReleasedNotificationNoHeaderCreateItemsEditAndDenial(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "user-1")
	ensureUser(t, testDB, "outsider")
	router := newTestRouter(t)
	shopID := createShop(t, router, "user-1", "Released client")
	vehicleID := createVehicle(t, router, "user-1", shopID)
	path := "/api/v1/auth/shops/vehicles/notifications"
	itemPath := "/api/v1/auth/shops/notifications/items/bulk"

	create := doJSONRequest(t, router, http.MethodPost, path, releasedCreateBody(shopID, vehicleID), "user-1")
	id := releasedNotification(t, releasedEnvelope(t, create, http.StatusCreated, "Notification created successfully"), shopID, vehicleID, "PM due")
	require.Nil(t, releasedObject(t, decodeStandardResponse(t, create.Body).Data)["attached_shop_list"])

	bulk := doJSONRequest(t, router, http.MethodPost, itemPath, releasedBulkBody(shopID, id, "1234"), "user-1")
	releasedItems(t, releasedEnvelope(t, bulk, http.StatusCreated, "Items added successfully"), shopID, id, "1234")
	getItems := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/notifications/"+id+"/items", nil, "user-1")
	releasedItems(t, releasedEnvelope(t, getItems, http.StatusOK, ""), shopID, id, "1234")

	editBody := map[string]any{
		"notification_id": id, "title": "PM due edited", "description": "Inspect",
		"type": "PM", "completed": true, "attached_shop_list": nil,
	}
	edit := doJSONRequest(t, router, http.MethodPut, path, editBody, "user-1")
	require.Equal(t, "Notification updated successfully", releasedString(t, releasedObject(t, releasedEnvelope(t, edit, http.StatusOK, "")), "message"))
	get := doJSONRequest(t, router, http.MethodGet, path+"/"+id, nil, "user-1")
	updated := releasedObject(t, releasedEnvelope(t, get, http.StatusOK, ""))
	require.Equal(t, "PM due edited", releasedString(t, updated, "title"))
	require.Equal(t, true, updated["completed"])
	require.Nil(t, updated["attached_shop_list"])

	for _, denied := range []*httptest.ResponseRecorder{
		doJSONRequest(t, router, http.MethodPost, path, releasedCreateBody(shopID, vehicleID), "outsider"),
		doJSONRequest(t, router, http.MethodPut, path, editBody, "outsider"),
		doJSONRequest(t, router, http.MethodPost, itemPath, releasedBulkBody(shopID, id, "blocked"), "outsider"),
	} {
		require.Equal(t, http.StatusInternalServerError, denied.Code)
		body := decodeStandardResponse(t, denied.Body)
		require.Equal(t, denied.Code, body.Status)
		require.NotEmpty(t, body.Message)
		require.Equal(t, "null", string(body.Data))
	}
	require.Zero(t, atomicRowCount(t, "shop_notification_items", "notification_id=$1 AND niin=$2", id, "blocked"))
}

func TestMixedNotificationAlternatingLegacyAndAtomicWriters(t *testing.T) {
	router, shopID, vehicleID := atomicFixture(t, "user-1")
	path := "/api/v1/auth/shops/vehicles/notifications"
	create := doJSONRequest(t, router, http.MethodPost, path, releasedCreateBody(shopID, vehicleID), "user-1")
	id := releasedNotification(t, releasedEnvelope(t, create, http.StatusCreated, "Notification created successfully"), shopID, vehicleID, "PM due")
	legacyEdit := map[string]any{"notification_id": id, "title": "Legacy first", "description": "Inspect", "type": "PM", "completed": false, "attached_shop_list": nil}
	releasedEnvelope(t, doJSONRequest(t, router, http.MethodPut, path, legacyEdit, "user-1"), http.StatusOK, "")

	atomic := atomicNotificationRequest(shopID, vehicleID)
	atomic.NotificationID = &id
	atomic.Details.Title = "Atomic middle"
	atomic.Items = []request.NotificationSaveItem{{ID: uuid.NewString(), Niin: "atomic-item", Nomenclature: "Part", Quantity: 1}}
	atomicReceipt(t, doContract2JSONRequest(t, router, atomic, "user-1"))
	releasedEnvelope(t, doJSONRequest(t, router, http.MethodDelete, "/api/v1/auth/shops/notifications/items/"+atomic.Items[0].ID, nil, "user-1"), http.StatusOK, "")
	bulk := doJSONRequest(t, router, http.MethodPost, "/api/v1/auth/shops/notifications/items/bulk", releasedBulkBody(shopID, id, "legacy-item"), "user-1")
	releasedItems(t, releasedEnvelope(t, bulk, http.StatusCreated, "Items added successfully"), shopID, id, "legacy-item")
	require.Equal(t, 1, atomicRowCount(t, "shop_notification_items", "notification_id=$1 AND niin=$2", id, "legacy-item"))
	require.Zero(t, atomicRowCount(t, "shop_notification_items", "notification_id=$1 AND niin=$2", id, "atomic-item"))

	atomic = atomicNotificationRequest(shopID, vehicleID)
	atomic.NotificationID = &id
	atomic.Details.Title = "Atomic before legacy"
	atomic.Items = []request.NotificationSaveItem{{ID: uuid.NewString(), Niin: "atomic-last", Nomenclature: "Part", Quantity: 1}}
	atomicReceipt(t, doContract2JSONRequest(t, router, atomic, "user-1"))
	legacyEdit["title"] = "Legacy last"
	releasedEnvelope(t, doJSONRequest(t, router, http.MethodPut, path, legacyEdit, "user-1"), http.StatusOK, "")
	releasedEnvelope(t, doJSONRequest(t, router, http.MethodDelete, "/api/v1/auth/shops/notifications/items/"+atomic.Items[0].ID, nil, "user-1"), http.StatusOK, "")
	bulk = doJSONRequest(t, router, http.MethodPost, "/api/v1/auth/shops/notifications/items/bulk", releasedBulkBody(shopID, id, "legacy-last"), "user-1")
	releasedItems(t, releasedEnvelope(t, bulk, http.StatusCreated, "Items added successfully"), shopID, id, "legacy-last")
	get := doJSONRequest(t, router, http.MethodGet, path+"/"+id, nil, "user-1")
	require.Equal(t, "Legacy last", releasedString(t, releasedObject(t, releasedEnvelope(t, get, http.StatusOK, "")), "title"))
	require.Equal(t, 1, atomicRowCount(t, "shop_notification_items", "notification_id=$1 AND niin=$2", id, "legacy-last"))
	require.Zero(t, atomicRowCount(t, "shop_notification_items", "notification_id=$1 AND niin=$2", id, "atomic-last"))
	require.Equal(t, 2, atomicRowCount(t, "shop_notification_operations", "notification_id=$1", id))
}

func TestMixedNotificationConcurrentUnrelatedWrites(t *testing.T) {
	router, shopID, vehicleID := atomicFixture(t, "user-1")
	itemTarget := doJSONRequest(t, router, http.MethodPost, "/api/v1/auth/shops/vehicles/notifications", releasedCreateBody(shopID, vehicleID), "user-1")
	itemTargetID := releasedNotification(t, releasedEnvelope(t, itemTarget, http.StatusCreated, "Notification created successfully"), shopID, vehicleID, "PM due")
	atomic := atomicNotificationRequest(shopID, vehicleID)
	start := make(chan struct{})
	var wg sync.WaitGroup
	var atomicResponse, legacyResponse, bulkResponse *httptest.ResponseRecorder
	wg.Add(3)
	go func() {
		defer wg.Done()
		<-start
		atomicResponse = doContract2JSONRequest(t, router, atomic, "user-1")
	}()
	go func() {
		defer wg.Done()
		<-start
		legacyResponse = doJSONRequest(t, router, http.MethodPost, "/api/v1/auth/shops/vehicles/notifications", releasedCreateBody(shopID, vehicleID), "user-1")
	}()
	go func() {
		defer wg.Done()
		<-start
		bulkResponse = doJSONRequest(t, router, http.MethodPost, "/api/v1/auth/shops/notifications/items/bulk", releasedBulkBody(shopID, itemTargetID, "1234"), "user-1")
	}()
	close(start)
	wg.Wait()
	atomicID := atomicReceipt(t, atomicResponse).NotificationID
	legacyID := releasedNotification(t, releasedEnvelope(t, legacyResponse, http.StatusCreated, "Notification created successfully"), shopID, vehicleID, "PM due")
	require.NotEqual(t, atomicID, legacyID)
	releasedItems(t, releasedEnvelope(t, bulkResponse, http.StatusCreated, "Items added successfully"), shopID, itemTargetID, "1234")
	require.Equal(t, 1, atomicRowCount(t, "shop_notification_items", "notification_id=$1", itemTargetID))
	require.Zero(t, atomicRowCount(t, "shop_notification_items", "notification_id=$1", legacyID))
	require.Equal(t, len(atomic.Items), atomicRowCount(t, "shop_notification_items", "notification_id=$1", atomicID))
	require.Zero(t, atomicRowCount(t, "shop_notification_items", "NOT EXISTS (SELECT 1 FROM shop_vehicle_notifications n WHERE n.id=shop_notification_items.notification_id)"))
}
