package shops_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestShopSnapshotDefaultIncludes(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "user-1")
	router := newTestRouter(t)

	shopID := createShop(t, router, "user-1", "Snapshot Shop")
	vehicleID := createVehicle(t, router, "user-1", shopID)
	listID := createList(t, router, "user-1", shopID)
	createListItem(t, router, "user-1", listID, "123456789", "Part")
	notificationID := createNotification(t, router, "user-1", shopID, vehicleID, "PM")
	require.NotEmpty(t, notificationID)
	serviceDate := time.Now().AddDate(0, 0, 2)
	createEquipmentService(t, "user-1", shopID, vehicleID, "", "Service", &serviceDate, false)

	resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/"+shopID+"/snapshot", nil, "user-1")
	require.Equal(t, http.StatusOK, resp.Code)
	payload := decodeMap(t, decodeStandardResponse(t, resp.Body).Data)
	shop := payload["shop"].(map[string]interface{})
	require.Equal(t, shopID, shop["id"])
	require.Len(t, payload["vehicles"].([]interface{}), 1)
	require.Len(t, payload["lists"].([]interface{}), 1)
	require.Len(t, payload["notifications"].([]interface{}), 1)
	require.Len(t, payload["services"].([]interface{}), 1)
}

func TestShopSnapshotIncludeMessages(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "user-1")
	router := newTestRouter(t)

	shopID := createShop(t, router, "user-1", "Message Snapshot")
	_ = createMessage(t, router, "user-1", shopID, "hello")

	resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/"+shopID+"/snapshot?include=messages&message_limit=1", nil, "user-1")
	require.Equal(t, http.StatusOK, resp.Code)
	payload := decodeMap(t, decodeStandardResponse(t, resp.Body).Data)
	require.Len(t, payload["messages"].([]interface{}), 1)
	require.Empty(t, payload["vehicles"].([]interface{}))
}

func TestShopSnapshotBoundsNotificationItems(t *testing.T) {
	const expectedItemLimit = 25

	clearShopTables(t, testDB)
	ensureUser(t, testDB, "user-1")
	router := newTestRouter(t)

	shopID := createShop(t, router, "user-1", "Bounded Items Snapshot")
	vehicleID := createVehicle(t, router, "user-1", shopID)
	notificationID := createNotification(t, router, "user-1", shopID, vehicleID, "PM")
	baseTime := time.Now().Add(-time.Hour).UTC()
	for i := 0; i < expectedItemLimit+5; i++ {
		_, err := testDB.Exec(
			`INSERT INTO shop_notification_items (
				id, shop_id, notification_id, niin, nomenclature, quantity, save_time
			) VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			fmt.Sprintf("snapshot-item-%02d", i),
			shopID,
			notificationID,
			fmt.Sprintf("000000%03d", i),
			fmt.Sprintf("Part %02d", i),
			1,
			baseTime.Add(time.Duration(i)*time.Second),
		)
		require.NoError(t, err)
	}

	resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/"+shopID+"/snapshot", nil, "user-1")
	require.Equal(t, http.StatusOK, resp.Code)
	payload := decodeMap(t, decodeStandardResponse(t, resp.Body).Data)
	notifications := payload["notifications"].([]interface{})
	require.Len(t, notifications, 1)
	items := notifications[0].(map[string]interface{})["items"].([]interface{})
	require.Len(t, items, expectedItemLimit)
	require.Equal(t, "000000000", items[0].(map[string]interface{})["niin"])
	require.Equal(t, "000000024", items[expectedItemLimit-1].(map[string]interface{})["niin"])
}

func TestShopSnapshotRejectsInvalidIncludeAndLimits(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "user-1")
	router := newTestRouter(t)

	resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/shop-1/snapshot?include=vehicles,bad", nil, "user-1")
	require.Equal(t, http.StatusBadRequest, resp.Code)
	require.Equal(t, "invalid include", decodeStandardResponse(t, resp.Body).Message)

	for _, path := range []string{
		"/api/v1/auth/shops/shop-1/snapshot?message_limit=0",
		"/api/v1/auth/shops/shop-1/snapshot?changes_limit=-1",
		"/api/v1/auth/shops/shop-1/snapshot?services_limit=",
	} {
		resp := doJSONRequest(t, router, http.MethodGet, path, nil, "user-1")
		require.Equal(t, http.StatusBadRequest, resp.Code)
	}
}
