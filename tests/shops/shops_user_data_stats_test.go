package shops_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUserDataWithShopsStatsRemainScopedToAuthenticatedUser(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "user-1")
	ensureUser(t, testDB, "user-2")
	router := newTestRouter(t)

	visibleShopID := createShop(t, router, "user-1", "Visible Stats")
	hiddenShopID := createShop(t, router, "user-2", "Hidden Stats")
	_ = createVehicle(t, router, "user-1", visibleShopID)
	_ = createVehicle(t, router, "user-2", hiddenShopID)
	_ = createMessage(t, router, "user-1", visibleShopID, "visible")
	_ = createMessage(t, router, "user-2", hiddenShopID, "hidden")

	resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/user-data", nil, "user-1")
	require.Equal(t, http.StatusOK, resp.Code)
	payload := decodeMap(t, decodeStandardResponse(t, resp.Body).Data)
	shops := payload["shops"].([]interface{})
	require.Len(t, shops, 1)

	shopWithStats := shops[0].(map[string]interface{})
	shop := shopWithStats["shop"].(map[string]interface{})
	require.Equal(t, visibleShopID, shop["id"])
	require.Equal(t, float64(1), shopWithStats["vehicle_count"])
	require.Equal(t, true, shopWithStats["is_admin"])
	require.Equal(t, false, shopWithStats["is_lists_admin_only"])
}
