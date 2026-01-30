package equipment_services_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestEquipmentServicesStatus(t *testing.T) {
	clearEquipmentServicesTables(t, testDB)
	ensureUser(t, testDB, "user-1")

	router := newTestRouter(t)
	shopID := createShop(t, router, "user-1", "Status Shop")
	equipmentID := createVehicle(t, router, "user-1", shopID)

	overdueDate := time.Now().AddDate(0, 0, -5)
	dueSoonDate := time.Now().AddDate(0, 0, 3)

	createEquipmentService(t, router, "user-1", shopID, equipmentID, "", "Overdue service", &overdueDate, false)
	createEquipmentService(t, router, "user-1", shopID, equipmentID, "", "Due soon service", &dueSoonDate, false)

	overdueResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/"+shopID+"/equipment-services/overdue", nil, "user-1")
	require.Equal(t, http.StatusOK, overdueResp.Code)

	overdueData := decodeStandardResponse(t, overdueResp.Body)
	overduePayload := decodeMap(t, overdueData.Data)
	overdueServices, ok := overduePayload["overdue_services"].([]interface{})
	require.True(t, ok)
	require.Len(t, overdueServices, 1)

	dueSoonResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/"+shopID+"/equipment-services/due-soon", nil, "user-1")
	require.Equal(t, http.StatusOK, dueSoonResp.Code)

	dueSoonData := decodeStandardResponse(t, dueSoonResp.Body)
	dueSoonPayload := decodeMap(t, dueSoonData.Data)
	dueSoonServices, ok := dueSoonPayload["due_soon_services"].([]interface{})
	require.True(t, ok)
	require.Len(t, dueSoonServices, 1)
}
