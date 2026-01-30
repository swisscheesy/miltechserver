package shops_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestShopSettingsEndpoints(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "user-1")

	router := newTestRouter(t)

	shopID := createShop(t, router, "user-1", "Settings Shop")

	getAdminOnlyResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/"+shopID+"/settings/admin-only-lists", nil, "user-1")
	require.Equal(t, http.StatusOK, getAdminOnlyResp.Code)

	adminOnly := decodeStandardResponse(t, getAdminOnlyResp.Body)
	adminOnlyData := decodeMap(t, adminOnly.Data)
	require.Equal(t, shopID, adminOnlyData["shop_id"])

	updateAdminOnlyBody := map[string]interface{}{
		"admin_only_lists": true,
	}

	updateAdminOnlyResp := doJSONRequest(t, router, http.MethodPut, "/api/v1/auth/shops/"+shopID+"/settings/admin-only-lists", updateAdminOnlyBody, "user-1")
	require.Equal(t, http.StatusOK, updateAdminOnlyResp.Code)

	getSettingsResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/"+shopID+"/settings", nil, "user-1")
	require.Equal(t, http.StatusOK, getSettingsResp.Code)

	settings := decodeStandardResponse(t, getSettingsResp.Body)
	settingsData := decodeMap(t, settings.Data)
	_, ok := settingsData["admin_only_lists"]
	require.True(t, ok)

	updateSettingsBody := map[string]interface{}{
		"admin_only_lists": false,
	}

	updateSettingsResp := doJSONRequest(t, router, http.MethodPut, "/api/v1/auth/shops/"+shopID+"/settings", updateSettingsBody, "user-1")
	require.Equal(t, http.StatusOK, updateSettingsResp.Code)
}
