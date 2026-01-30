package equipment_services_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestEquipmentServicesQueries(t *testing.T) {
	clearEquipmentServicesTables(t, testDB)
	ensureUser(t, testDB, "user-1")

	router := newTestRouter(t)
	shopID := createShop(t, router, "user-1", "Query Shop")
	equipmentID := createVehicle(t, router, "user-1", shopID)
	otherEquipmentID := createVehicle(t, router, "user-1", shopID)

	serviceDate := time.Now().AddDate(0, 0, 3)
	createEquipmentService(t, router, "user-1", shopID, equipmentID, "", "Service A", &serviceDate, false)
	createEquipmentService(t, router, "user-1", shopID, otherEquipmentID, "", "Service B", &serviceDate, false)

	listResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/"+shopID+"/equipment-services", nil, "user-1")
	require.Equal(t, http.StatusOK, listResp.Code)

	listData := decodeStandardResponse(t, listResp.Body)
	payload := decodeMap(t, listData.Data)
	services, ok := payload["services"].([]interface{})
	require.True(t, ok)
	require.Len(t, services, 2)

	equipmentResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/"+shopID+"/equipment/"+equipmentID+"/services", nil, "user-1")
	require.Equal(t, http.StatusOK, equipmentResp.Code)

	equipmentData := decodeStandardResponse(t, equipmentResp.Body)
	equipmentPayload := decodeMap(t, equipmentData.Data)
	equipmentServices, ok := equipmentPayload["services"].([]interface{})
	require.True(t, ok)
	require.Len(t, equipmentServices, 1)
}
