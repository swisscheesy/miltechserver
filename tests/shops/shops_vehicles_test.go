package shops_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVehicleCRUD(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "user-1")

	router := newTestRouter(t)

	shopID := createShop(t, router, "user-1", "Vehicle Shop")
	vehicleID := createVehicle(t, router, "user-1", shopID)

	getListResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/"+shopID+"/vehicles", nil, "user-1")
	require.Equal(t, http.StatusOK, getListResp.Code)

	list := decodeStandardResponse(t, getListResp.Body)
	vehicles := decodeSlice(t, list.Data)
	require.Len(t, vehicles, 1)

	getByIDResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/vehicles/"+vehicleID, nil, "user-1")
	require.Equal(t, http.StatusOK, getByIDResp.Code)

	updateBody := map[string]interface{}{
		"vehicle_id": vehicleID,
		"admin":      "updated-admin",
		"niin":       "2222-22-222-2222",
		"model":      "Model X",
		"serial":     "SERIAL-1",
		"uoc":        "UOC",
		"mileage":    5,
		"hours":      10,
		"comment":    "updated",
	}

	updateResp := doJSONRequest(t, router, http.MethodPut, "/api/v1/auth/shops/vehicles", updateBody, "user-1")
	require.Equal(t, http.StatusOK, updateResp.Code)

	deleteResp := doJSONRequest(t, router, http.MethodDelete, "/api/v1/auth/shops/vehicles/"+vehicleID, nil, "user-1")
	require.Equal(t, http.StatusOK, deleteResp.Code)

	getListAfterDelete := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/"+shopID+"/vehicles", nil, "user-1")
	require.Equal(t, http.StatusOK, getListAfterDelete.Code)

	listAfterDelete := decodeStandardResponse(t, getListAfterDelete.Body)
	vehiclesAfterDelete := decodeSlice(t, listAfterDelete.Data)
	require.Len(t, vehiclesAfterDelete, 0)
}
