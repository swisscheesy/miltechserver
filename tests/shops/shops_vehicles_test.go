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

func TestVehicleUsageUpdateRequiresShopMembership(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "vehicle-owner")
	ensureUser(t, testDB, "shop-member")
	ensureUser(t, testDB, "shop-outsider")

	router := newTestRouter(t)
	shopID := createShop(t, router, "vehicle-owner", "Usage Shop")

	createBody := map[string]interface{}{
		"shop_id": shopID,
		"admin":   "ADMIN-001",
		"niin":    "1234-00-123-4567",
		"model":   "M1123",
		"serial":  "SERIAL-001",
		"uoc":     "ABC",
		"mileage": 100,
		"hours":   50,
		"comment": "owner-managed metadata",
	}
	createResp := doJSONRequest(t, router, http.MethodPost, "/api/v1/auth/shops/vehicles", createBody, "vehicle-owner")
	require.Equal(t, http.StatusCreated, createResp.Code)
	createdVehicle := decodeMap(t, decodeStandardResponse(t, createResp.Body).Data)
	vehicleID, ok := createdVehicle["id"].(string)
	require.True(t, ok)

	_, inviteCode := createInviteCode(t, router, "vehicle-owner", shopID)
	joinResp := doJSONRequest(
		t,
		router,
		http.MethodPost,
		"/api/v1/auth/shops/join",
		map[string]interface{}{"invite_code": inviteCode},
		"shop-member",
	)
	require.Equal(t, http.StatusOK, joinResp.Code)

	ownerUpdate := map[string]interface{}{
		"vehicle_id":      vehicleID,
		"admin":           "ADMIN-001",
		"niin":            "1234-00-123-4567",
		"model":           "M1123",
		"serial":          "SERIAL-001",
		"uoc":             "ABC",
		"mileage":         110,
		"hours":           55,
		"tracked_mileage": 110,
		"tracked_hours":   55,
		"comment":         "owner-managed metadata",
	}
	ownerUpdateResp := doJSONRequest(t, router, http.MethodPut, "/api/v1/auth/shops/vehicles", ownerUpdate, "vehicle-owner")
	require.Equal(t, http.StatusOK, ownerUpdateResp.Code)

	memberUpdate := map[string]interface{}{
		"vehicle_id":      vehicleID,
		"admin":           "ADMIN-001",
		"mileage":         100,
		"hours":           50,
		"tracked_mileage": 125,
		"tracked_hours":   60,
	}
	memberUpdateResp := doJSONRequest(t, router, http.MethodPut, "/api/v1/auth/shops/vehicles", memberUpdate, "shop-member")
	require.Equal(t, http.StatusOK, memberUpdateResp.Code)

	getAfterMemberUpdate := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/vehicles/"+vehicleID, nil, "shop-member")
	require.Equal(t, http.StatusOK, getAfterMemberUpdate.Code)
	updatedVehicle := decodeMap(t, decodeStandardResponse(t, getAfterMemberUpdate.Body).Data)
	require.Equal(t, float64(125), updatedVehicle["tracked_mileage"])
	require.Equal(t, float64(60), updatedVehicle["tracked_hours"])
	require.Equal(t, "ADMIN-001", updatedVehicle["admin"])
	require.Equal(t, "1234-00-123-4567", updatedVehicle["niin"])
	require.Equal(t, "M1123", updatedVehicle["model"])
	require.Equal(t, "SERIAL-001", updatedVehicle["serial"])
	require.Equal(t, "ABC", updatedVehicle["uoc"])
	require.Equal(t, float64(110), updatedVehicle["mileage"])
	require.Equal(t, float64(55), updatedVehicle["hours"])
	require.Equal(t, "owner-managed metadata", updatedVehicle["comment"])

	memberMetadataUpdate := map[string]interface{}{
		"vehicle_id":      vehicleID,
		"admin":           "ADMIN-001",
		"model":           "UNAUTHORIZED-METADATA-CHANGE",
		"mileage":         100,
		"hours":           50,
		"tracked_mileage": 500,
		"tracked_hours":   500,
	}
	memberMetadataUpdateResp := doJSONRequest(t, router, http.MethodPut, "/api/v1/auth/shops/vehicles", memberMetadataUpdate, "shop-member")
	require.Equal(t, http.StatusInternalServerError, memberMetadataUpdateResp.Code)

	outsiderUpdate := map[string]interface{}{
		"vehicle_id":      vehicleID,
		"admin":           "ADMIN-001",
		"tracked_mileage": 500,
		"tracked_hours":   500,
	}
	outsiderUpdateResp := doJSONRequest(t, router, http.MethodPut, "/api/v1/auth/shops/vehicles", outsiderUpdate, "shop-outsider")
	require.Equal(t, http.StatusInternalServerError, outsiderUpdateResp.Code)

	getAfterOutsiderUpdate := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/vehicles/"+vehicleID, nil, "shop-member")
	require.Equal(t, http.StatusOK, getAfterOutsiderUpdate.Code)
	vehicleAfterOutsiderUpdate := decodeMap(t, decodeStandardResponse(t, getAfterOutsiderUpdate.Body).Data)
	require.Equal(t, float64(125), vehicleAfterOutsiderUpdate["tracked_mileage"])
	require.Equal(t, float64(60), vehicleAfterOutsiderUpdate["tracked_hours"])
}
