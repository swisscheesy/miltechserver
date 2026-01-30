package equipment_services_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestEquipmentServicesCRUDAndCompletion(t *testing.T) {
	clearEquipmentServicesTables(t, testDB)
	ensureUser(t, testDB, "user-1")

	router := newTestRouter(t)
	shopID := createShop(t, router, "user-1", "Equipment Shop")
	equipmentID := createVehicle(t, router, "user-1", shopID)
	listID := createList(t, router, "user-1", shopID)

	serviceDate := time.Now().AddDate(0, 0, 7)
	serviceID := createEquipmentService(t, router, "user-1", shopID, equipmentID, listID, "Initial service", &serviceDate, false)

	getResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/"+shopID+"/equipment-services/"+serviceID, nil, "user-1")
	require.Equal(t, http.StatusOK, getResp.Code)

	getData := decodeStandardResponse(t, getResp.Body)
	serviceMap := decodeMap(t, getData.Data)
	require.Equal(t, serviceID, serviceMap["id"])

	updateBody := map[string]interface{}{
		"service_id":   serviceID,
		"description":  "Updated service",
		"service_type": "maintenance",
		"list_id":      listID,
		"is_completed": false,
		"service_date": serviceDate,
	}
	updateResp := doJSONRequest(t, router, http.MethodPut, "/api/v1/auth/shops/"+shopID+"/equipment-services/"+serviceID, updateBody, "user-1")
	require.Equal(t, http.StatusOK, updateResp.Code)

	completeResp := doJSONRequest(t, router, http.MethodPost, "/api/v1/auth/shops/"+shopID+"/equipment-services/"+serviceID+"/complete", map[string]interface{}{}, "user-1")
	require.Equal(t, http.StatusOK, completeResp.Code)

	deleteResp := doJSONRequest(t, router, http.MethodDelete, "/api/v1/auth/shops/"+shopID+"/equipment-services/"+serviceID, nil, "user-1")
	require.Equal(t, http.StatusOK, deleteResp.Code)
}
