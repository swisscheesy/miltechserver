package shops_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestShopsBootstrapReturnsOnlyAuthenticatedUserShops(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "user-1")
	ensureUser(t, testDB, "user-2")
	router := newTestRouter(t)

	visibleShopID := createShop(t, router, "user-1", "Visible Bootstrap")
	hiddenShopID := createShop(t, router, "user-2", "Hidden Bootstrap")
	visibleEquipmentID := createVehicle(t, router, "user-1", visibleShopID)
	_ = createVehicle(t, router, "user-2", hiddenShopID)
	_, err := testDB.Exec(`UPDATE shop_vehicle SET admin='A1', model='M1', serial='S1', niin='000000001' WHERE id=$1`, visibleEquipmentID)
	require.NoError(t, err)

	resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/bootstrap", nil, "user-1")
	require.Equal(t, http.StatusOK, resp.Code)
	payload := decodeMap(t, decodeStandardResponse(t, resp.Body).Data)
	shops := payload["shops"].([]interface{})
	require.Len(t, shops, 1)
	shop := shops[0].(map[string]interface{})
	require.Equal(t, visibleShopID, shop["id"])
	require.Len(t, shop["equipment"].([]interface{}), 1)
}
