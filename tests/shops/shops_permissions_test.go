package shops_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUnauthorizedRequestRejected(t *testing.T) {
	clearShopTables(t, testDB)

	router := newTestRouter(t)

	createBody := map[string]interface{}{
		"name":    "Unauthorized Shop",
		"details": "Details",
	}

	resp := doJSONRequest(t, router, http.MethodPost, "/api/v1/auth/shops", createBody, "")
	require.Equal(t, http.StatusUnauthorized, resp.Code)
}

func TestNonMemberCannotAccessShop(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "user-1")
	ensureUser(t, testDB, "user-2")

	router := newTestRouter(t)

	shopID := createShop(t, router, "user-1", "Member Shop")

	resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/"+shopID, nil, "user-2")
	require.Equal(t, http.StatusInternalServerError, resp.Code)
}

func TestNonAdminCannotUpdateSettings(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "user-1")
	ensureUser(t, testDB, "user-2")

	router := newTestRouter(t)

	shopID := createShop(t, router, "user-1", "Settings Shop")

	updateSettingsBody := map[string]interface{}{
		"admin_only_lists": true,
	}

	resp := doJSONRequest(t, router, http.MethodPut, "/api/v1/auth/shops/"+shopID+"/settings", updateSettingsBody, "user-2")
	require.Equal(t, http.StatusInternalServerError, resp.Code)
}

func TestInvalidNotificationTypeRejected(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "user-1")

	router := newTestRouter(t)

	shopID := createShop(t, router, "user-1", "Notify Shop")
	vehicleID := createVehicle(t, router, "user-1", shopID)

	notificationBody := map[string]interface{}{
		"shop_id":     shopID,
		"vehicle_id":  vehicleID,
		"title":       "Invalid Type",
		"description": "desc",
		"type":        "BAD",
	}

	resp := doJSONRequest(t, router, http.MethodPost, "/api/v1/auth/shops/vehicles/notifications", notificationBody, "user-1")
	require.Equal(t, http.StatusInternalServerError, resp.Code)
}
