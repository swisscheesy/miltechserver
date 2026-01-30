package equipment_services_test

import (
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestEquipmentServicesListMismatch(t *testing.T) {
	clearEquipmentServicesTables(t, testDB)
	ensureUser(t, testDB, "user-1")

	router := newTestRouter(t)
	shopID := createShop(t, router, "user-1", "Mismatch Shop")
	equipmentID := createVehicle(t, router, "user-1", shopID)

	otherShopID := createShop(t, router, "user-1", "Other Shop")
	listID := createList(t, router, "user-1", otherShopID)

	createBody := map[string]interface{}{
		"equipment_id": equipmentID,
		"list_id":      listID,
		"description":  "Mismatch",
		"service_type": "inspection",
		"is_completed": false,
	}

	createResp := doJSONRequest(t, router, http.MethodPost, "/api/v1/auth/shops/"+shopID+"/equipment-services", createBody, "user-1")
	require.Equal(t, http.StatusInternalServerError, createResp.Code)
}

func TestEquipmentServicesPaginationAndFilters(t *testing.T) {
	clearEquipmentServicesTables(t, testDB)
	ensureUser(t, testDB, "user-1")

	router := newTestRouter(t)
	shopID := createShop(t, router, "user-1", "Filter Shop")
	equipmentID := createVehicle(t, router, "user-1", shopID)

	serviceDate := time.Now().AddDate(0, 0, 5)
	createEquipmentService(t, router, "user-1", shopID, equipmentID, "", "Service 1", &serviceDate, false)
	createEquipmentService(t, router, "user-1", shopID, equipmentID, "", "Service 2", &serviceDate, false)
	createEquipmentService(t, router, "user-1", shopID, equipmentID, "", "Service 3", &serviceDate, false)

	params := url.Values{}
	params.Set("limit", "1")
	params.Set("offset", "1")

	listResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/"+shopID+"/equipment-services?"+params.Encode(), nil, "user-1")
	require.Equal(t, http.StatusOK, listResp.Code)

	listData := decodeStandardResponse(t, listResp.Body)
	payload := decodeMap(t, listData.Data)
	services, ok := payload["services"].([]interface{})
	require.True(t, ok)
	require.Len(t, services, 1)
}

func TestEquipmentServicesEquipmentAccessDenied(t *testing.T) {
	clearEquipmentServicesTables(t, testDB)
	ensureUser(t, testDB, "user-1")
	ensureUser(t, testDB, "user-2")

	router := newTestRouter(t)
	shopID := createShop(t, router, "user-1", "Equipment Access Shop")
	equipmentID := createVehicle(t, router, "user-1", shopID)

	resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/"+shopID+"/equipment/"+equipmentID+"/services", nil, "user-2")
	require.Equal(t, http.StatusInternalServerError, resp.Code)
}
