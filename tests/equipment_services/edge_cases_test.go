package equipment_services_test

import (
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestEquipmentServicesUnauthorized(t *testing.T) {
	clearEquipmentServicesTables(t, testDB)

	router := newTestRouter(t)

	resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/any/equipment-services", nil, "")
	require.Equal(t, http.StatusUnauthorized, resp.Code)
}

func TestEquipmentServicesAccessDenied(t *testing.T) {
	clearEquipmentServicesTables(t, testDB)
	ensureUser(t, testDB, "user-1")
	ensureUser(t, testDB, "user-2")

	router := newTestRouter(t)
	shopID := createShop(t, router, "user-1", "Access Shop")
	equipmentID := createVehicle(t, router, "user-1", shopID)

	serviceDate := time.Now().AddDate(0, 0, 1)
	serviceID := createEquipmentService(t, router, "user-1", shopID, equipmentID, "", "Access test", &serviceDate, false)

	listResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/"+shopID+"/equipment-services", nil, "user-2")
	require.Equal(t, http.StatusInternalServerError, listResp.Code)

	updateBody := map[string]interface{}{
		"service_id":   serviceID,
		"description":  "Update denied",
		"service_type": "maintenance",
		"list_id":      "",
		"is_completed": false,
		"service_date": serviceDate,
	}

	updateResp := doJSONRequest(t, router, http.MethodPut, "/api/v1/auth/shops/"+shopID+"/equipment-services/"+serviceID, updateBody, "user-2")
	require.Equal(t, http.StatusInternalServerError, updateResp.Code)

	deleteResp := doJSONRequest(t, router, http.MethodDelete, "/api/v1/auth/shops/"+shopID+"/equipment-services/"+serviceID, nil, "user-2")
	require.Equal(t, http.StatusInternalServerError, deleteResp.Code)
}

func TestEquipmentServicesValidationErrors(t *testing.T) {
	clearEquipmentServicesTables(t, testDB)
	ensureUser(t, testDB, "user-1")

	router := newTestRouter(t)
	shopID := createShop(t, router, "user-1", "Validation Shop")
	equipmentID := createVehicle(t, router, "user-1", shopID)

	badHours := int32(-1)
	createBody := map[string]interface{}{
		"equipment_id":  equipmentID,
		"list_id":       "",
		"description":   "Bad hours",
		"service_type":  "inspection",
		"is_completed":  false,
		"service_hours": badHours,
	}

	createResp := doJSONRequest(t, router, http.MethodPost, "/api/v1/auth/shops/"+shopID+"/equipment-services", createBody, "user-1")
	require.Equal(t, http.StatusBadRequest, createResp.Code)

	serviceDate := time.Now().AddDate(0, 0, 2)
	serviceID := createEquipmentService(t, router, "user-1", shopID, equipmentID, "", "Valid", &serviceDate, false)

	updateBody := map[string]interface{}{
		"service_id":    serviceID,
		"description":   "Bad update",
		"service_type":  "maintenance",
		"list_id":       "",
		"is_completed":  false,
		"service_hours": badHours,
	}

	updateResp := doJSONRequest(t, router, http.MethodPut, "/api/v1/auth/shops/"+shopID+"/equipment-services/"+serviceID, updateBody, "user-1")
	require.Equal(t, http.StatusBadRequest, updateResp.Code)
}

func TestEquipmentServicesDateValidation(t *testing.T) {
	clearEquipmentServicesTables(t, testDB)
	ensureUser(t, testDB, "user-1")

	router := newTestRouter(t)
	shopID := createShop(t, router, "user-1", "Date Shop")
	equipmentID := createVehicle(t, router, "user-1", shopID)

	params := url.Values{}
	params.Set("start_date", "not-a-date")
	params.Set("end_date", time.Now().AddDate(0, 0, 1).Format(time.RFC3339))

	calendarResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/"+shopID+"/equipment-services/calendar?"+params.Encode(), nil, "user-1")
	require.Equal(t, http.StatusInternalServerError, calendarResp.Code)

	queryParams := url.Values{}
	queryParams.Set("start_date", "bad-date")
	listResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/"+shopID+"/equipment/"+equipmentID+"/services?"+queryParams.Encode(), nil, "user-1")
	require.Equal(t, http.StatusBadRequest, listResp.Code)
}

func TestEquipmentServicesCompletionDates(t *testing.T) {
	clearEquipmentServicesTables(t, testDB)
	ensureUser(t, testDB, "user-1")

	router := newTestRouter(t)
	shopID := createShop(t, router, "user-1", "Completion Shop")
	equipmentID := createVehicle(t, router, "user-1", shopID)

	createBody := map[string]interface{}{
		"equipment_id": equipmentID,
		"list_id":      "",
		"description":  "Auto completion",
		"service_type": "inspection",
		"is_completed": true,
	}

	createResp := doJSONRequest(t, router, http.MethodPost, "/api/v1/auth/shops/"+shopID+"/equipment-services", createBody, "user-1")
	require.Equal(t, http.StatusCreated, createResp.Code)

	created := decodeStandardResponse(t, createResp.Body)
	serviceData := decodeMap(t, created.Data)
	require.NotNil(t, serviceData["completion_date"])

	serviceID, ok := serviceData["id"].(string)
	require.True(t, ok)

	updateBody := map[string]interface{}{
		"service_id":   serviceID,
		"description":  "Clear completion",
		"service_type": "maintenance",
		"list_id":      "",
		"is_completed": false,
	}

	updateResp := doJSONRequest(t, router, http.MethodPut, "/api/v1/auth/shops/"+shopID+"/equipment-services/"+serviceID, updateBody, "user-1")
	require.Equal(t, http.StatusOK, updateResp.Code)

	updated := decodeStandardResponse(t, updateResp.Body)
	updatedData := decodeMap(t, updated.Data)
	require.Nil(t, updatedData["completion_date"])
}

func TestEquipmentServicesNotFound(t *testing.T) {
	clearEquipmentServicesTables(t, testDB)
	ensureUser(t, testDB, "user-1")

	router := newTestRouter(t)
	shopID := createShop(t, router, "user-1", "NotFound Shop")

	resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/"+shopID+"/equipment-services/non-existent", nil, "user-1")
	require.Equal(t, http.StatusInternalServerError, resp.Code)
}
