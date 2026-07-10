package shops_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestVehicleMaintenanceSnapshot(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "user-1")
	router := newTestRouter(t)

	shopID := createShop(t, router, "user-1", "Maintenance Snapshot")
	vehicleID := createVehicle(t, router, "user-1", shopID)
	listID := createList(t, router, "user-1", shopID)
	notificationBody := map[string]interface{}{
		"shop_id":            shopID,
		"vehicle_id":         vehicleID,
		"title":              "PM",
		"description":        "desc",
		"type":               "PM",
		"attached_shop_list": listID,
	}
	createNotificationResp := doJSONRequest(t, router, http.MethodPost, "/api/v1/auth/shops/vehicles/notifications", notificationBody, "user-1")
	require.Equal(t, http.StatusCreated, createNotificationResp.Code)
	notificationData := decodeMap(t, decodeStandardResponse(t, createNotificationResp.Body).Data)
	notificationID, ok := notificationData["id"].(string)
	require.True(t, ok)
	require.NotEmpty(t, notificationID)
	itemBody := map[string]any{"notification_id": notificationID, "niin": "123456789", "nomenclature": "Filter", "quantity": 1}
	addItemResp := doJSONRequest(t, router, http.MethodPost, "/api/v1/auth/shops/notifications/items", itemBody, "user-1")
	require.Equal(t, http.StatusCreated, addItemResp.Code)

	serviceDate := time.Now().AddDate(0, 0, 5)
	createEquipmentService(t, "user-1", shopID, vehicleID, "", "Scheduled service", &serviceDate, false)

	resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/vehicles/"+vehicleID+"/maintenance-snapshot", nil, "user-1")
	require.Equal(t, http.StatusOK, resp.Code)
	payload := decodeMap(t, decodeStandardResponse(t, resp.Body).Data)
	require.NotNil(t, payload["vehicle"])
	notifications := payload["notifications"].([]interface{})
	require.Len(t, notifications, 1)
	notification := notifications[0].(map[string]interface{})["notification"].(map[string]interface{})
	require.Equal(t, listID, notification["attached_shop_list"])
	require.Len(t, payload["services"].([]interface{}), 1)
	counts := payload["counts"].(map[string]interface{})
	require.Equal(t, float64(1), counts["notifications"])
	require.Equal(t, float64(1), counts["notification_items"])
	require.Equal(t, float64(1), counts["services"])
}

func TestVehicleMaintenanceSnapshotReturnsAllNotificationsWhenLimitOmitted(t *testing.T) {
	const expectedNotificationCount = 51

	clearShopTables(t, testDB)
	ensureUser(t, testDB, "user-1")
	router := newTestRouter(t)

	shopID := createShop(t, router, "user-1", "Unbounded Maintenance Snapshot")
	vehicleID := createVehicle(t, router, "user-1", shopID)
	for i := 0; i < expectedNotificationCount; i++ {
		notificationID := createNotification(t, router, "user-1", shopID, vehicleID, fmt.Sprintf("PM-%02d", i))
		itemBody := map[string]any{
			"notification_id": notificationID,
			"niin":            fmt.Sprintf("123456%03d", i),
			"nomenclature":    "Filter",
			"quantity":        1,
		}
		addItemResp := doJSONRequest(t, router, http.MethodPost, "/api/v1/auth/shops/notifications/items", itemBody, "user-1")
		require.Equal(t, http.StatusCreated, addItemResp.Code)
	}

	resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/vehicles/"+vehicleID+"/maintenance-snapshot", nil, "user-1")
	require.Equal(t, http.StatusOK, resp.Code)
	payload := decodeMap(t, decodeStandardResponse(t, resp.Body).Data)
	notifications := payload["notifications"].([]interface{})
	require.Len(t, notifications, expectedNotificationCount)
	for _, notification := range notifications {
		notificationWithItems := notification.(map[string]interface{})
		require.Len(t, notificationWithItems["items"].([]interface{}), 1)
	}
	counts := payload["counts"].(map[string]interface{})
	require.Equal(t, float64(expectedNotificationCount), counts["notifications"])
	require.Equal(t, float64(expectedNotificationCount), counts["notification_items"])
	limits := payload["limits"].(map[string]interface{})
	require.Nil(t, limits["notifications"])
}

func TestVehicleMaintenanceSnapshotReturnsAllNotificationItemsWhenLimitOmitted(t *testing.T) {
	const expectedItemCount = 26

	clearShopTables(t, testDB)
	ensureUser(t, testDB, "user-1")
	router := newTestRouter(t)

	shopID := createShop(t, router, "user-1", "Unbounded Maintenance Items")
	vehicleID := createVehicle(t, router, "user-1", shopID)
	notificationID := createNotification(t, router, "user-1", shopID, vehicleID, "PM")
	baseTime := time.Now().Add(-time.Hour).UTC()
	for i := 0; i < expectedItemCount; i++ {
		_, err := testDB.Exec(
			`INSERT INTO shop_notification_items (
				id, shop_id, notification_id, niin, nomenclature, quantity, save_time
			) VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			fmt.Sprintf("maintenance-item-%02d", i),
			shopID,
			notificationID,
			fmt.Sprintf("111111%03d", i),
			fmt.Sprintf("Part %02d", i),
			1,
			baseTime.Add(time.Duration(i)*time.Second),
		)
		require.NoError(t, err)
	}

	resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/vehicles/"+vehicleID+"/maintenance-snapshot", nil, "user-1")
	require.Equal(t, http.StatusOK, resp.Code)
	payload := decodeMap(t, decodeStandardResponse(t, resp.Body).Data)
	notifications := payload["notifications"].([]interface{})
	require.Len(t, notifications, 1)
	items := notifications[0].(map[string]interface{})["items"].([]interface{})
	require.Len(t, items, expectedItemCount)
	counts := payload["counts"].(map[string]interface{})
	require.Equal(t, float64(expectedItemCount), counts["notification_items"])
	limits := payload["limits"].(map[string]interface{})
	require.Nil(t, limits["notification_items_per_notification"])
}

func TestVehicleMaintenanceSnapshotClampsNotificationItemsMaxCap(t *testing.T) {
	const expectedMaxItemLimit = 100

	clearShopTables(t, testDB)
	ensureUser(t, testDB, "user-1")
	router := newTestRouter(t)

	shopID := createShop(t, router, "user-1", "Max Maintenance Items")
	vehicleID := createVehicle(t, router, "user-1", shopID)
	notificationID := createNotification(t, router, "user-1", shopID, vehicleID, "PM")
	baseTime := time.Now().Add(-time.Hour).UTC()
	for i := 0; i < expectedMaxItemLimit+1; i++ {
		_, err := testDB.Exec(
			`INSERT INTO shop_notification_items (
				id, shop_id, notification_id, niin, nomenclature, quantity, save_time
			) VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			fmt.Sprintf("maintenance-max-item-%03d", i),
			shopID,
			notificationID,
			fmt.Sprintf("222222%03d", i),
			fmt.Sprintf("Part %03d", i),
			1,
			baseTime.Add(time.Duration(i)*time.Second),
		)
		require.NoError(t, err)
	}

	path := "/api/v1/auth/shops/vehicles/" + vehicleID + "/maintenance-snapshot?notification_items_limit=999"
	resp := doJSONRequest(t, router, http.MethodGet, path, nil, "user-1")
	require.Equal(t, http.StatusOK, resp.Code)
	payload := decodeMap(t, decodeStandardResponse(t, resp.Body).Data)
	notifications := payload["notifications"].([]interface{})
	require.Len(t, notifications, 1)
	items := notifications[0].(map[string]interface{})["items"].([]interface{})
	require.Len(t, items, expectedMaxItemLimit)
	limits := payload["limits"].(map[string]interface{})
	require.Equal(t, float64(expectedMaxItemLimit), limits["notification_items_per_notification"])
}

func createEquipmentService(t *testing.T, userID, shopID, equipmentID, listID, description string, serviceDate *time.Time, isCompleted bool) string {
	t.Helper()

	serviceID := uuid.New().String()
	now := time.Now().UTC()
	_, err := testDB.Exec(
		`INSERT INTO equipment_services (
			id, shop_id, equipment_id, list_id, description, service_type,
			created_by, is_completed, created_at, updated_at, service_date
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		serviceID,
		shopID,
		equipmentID,
		listID,
		description,
		"inspection",
		userID,
		isCompleted,
		now,
		now,
		serviceDate,
	)
	require.NoError(t, err)

	return serviceID
}
