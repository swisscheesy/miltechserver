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
	notificationID := createNotification(t, router, "user-1", shopID, vehicleID, "PM")
	itemBody := map[string]any{"notification_id": notificationID, "niin": "123456789", "nomenclature": "Filter", "quantity": 1}
	addItemResp := doJSONRequest(t, router, http.MethodPost, "/api/v1/auth/shops/notifications/items", itemBody, "user-1")
	require.Equal(t, http.StatusCreated, addItemResp.Code)

	serviceDate := time.Now().AddDate(0, 0, 5)
	createEquipmentService(t, "user-1", shopID, vehicleID, "", "Scheduled service", &serviceDate, false)

	resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/vehicles/"+vehicleID+"/maintenance-snapshot", nil, "user-1")
	require.Equal(t, http.StatusOK, resp.Code)
	payload := decodeMap(t, decodeStandardResponse(t, resp.Body).Data)
	require.NotNil(t, payload["vehicle"])
	require.Len(t, payload["notifications"].([]interface{}), 1)
	require.Len(t, payload["services"].([]interface{}), 1)
	counts := payload["counts"].(map[string]interface{})
	require.Equal(t, float64(1), counts["notifications"])
	require.Equal(t, float64(1), counts["notification_items"])
	require.Equal(t, float64(1), counts["services"])
}

func TestVehicleMaintenanceSnapshotBoundsNotifications(t *testing.T) {
	const expectedNotificationLimit = 50

	clearShopTables(t, testDB)
	ensureUser(t, testDB, "user-1")
	router := newTestRouter(t)

	shopID := createShop(t, router, "user-1", "Bounded Maintenance Snapshot")
	vehicleID := createVehicle(t, router, "user-1", shopID)
	for i := 0; i < expectedNotificationLimit+1; i++ {
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
	require.Len(t, notifications, expectedNotificationLimit)
	for _, notification := range notifications {
		notificationWithItems := notification.(map[string]interface{})
		require.Len(t, notificationWithItems["items"].([]interface{}), 1)
	}
	counts := payload["counts"].(map[string]interface{})
	require.Equal(t, float64(expectedNotificationLimit), counts["notifications"])
	require.Equal(t, float64(expectedNotificationLimit), counts["notification_items"])
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
