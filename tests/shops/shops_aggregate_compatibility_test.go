package shops_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAggregateRoutesDoNotBreakExistingShopRoutes(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "user-1")
	router := newTestRouter(t)

	shopID := createShop(t, router, "user-1", "Compatibility Shop")

	legacyShopsResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops", nil, "user-1")
	require.Equal(t, http.StatusOK, legacyShopsResp.Code)

	legacyShopDetailResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/"+shopID, nil, "user-1")
	require.Equal(t, http.StatusOK, legacyShopDetailResp.Code)

	legacyUserDataResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/user-data", nil, "user-1")
	require.Equal(t, http.StatusOK, legacyUserDataResp.Code)

	legacyOverviewResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/equipment/overview", nil, "user-1")
	require.Equal(t, http.StatusOK, legacyOverviewResp.Code)

	bootstrapResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/bootstrap", nil, "user-1")
	require.Equal(t, http.StatusOK, bootstrapResp.Code)

	snapshotResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/"+shopID+"/snapshot", nil, "user-1")
	require.Equal(t, http.StatusOK, snapshotResp.Code)
}

func TestAggregateRoutesRequireAuthentication(t *testing.T) {
	router := newTestRouter(t)

	paths := []string{
		"/api/v1/auth/shops/bootstrap",
		"/api/v1/auth/shops/shop-1/lists-with-items",
		"/api/v1/auth/shops/shop-1/snapshot",
		"/api/v1/auth/shops/vehicles/vehicle-1/maintenance-snapshot",
	}

	for _, path := range paths {
		resp := doJSONRequest(t, router, http.MethodGet, path, nil, "")
		require.Equal(t, http.StatusUnauthorized, resp.Code, path)
	}
}
