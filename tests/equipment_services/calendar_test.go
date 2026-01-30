package equipment_services_test

import (
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestEquipmentServicesCalendar(t *testing.T) {
	clearEquipmentServicesTables(t, testDB)
	ensureUser(t, testDB, "user-1")

	router := newTestRouter(t)
	shopID := createShop(t, router, "user-1", "Calendar Shop")
	equipmentID := createVehicle(t, router, "user-1", shopID)

	serviceDate := time.Now().AddDate(0, 0, 2)
	createEquipmentService(t, router, "user-1", shopID, equipmentID, "", "Calendar service", &serviceDate, false)

	startDate := time.Now().AddDate(0, 0, -1).Format(time.RFC3339)
	endDate := time.Now().AddDate(0, 0, 5).Format(time.RFC3339)

	params := url.Values{}
	params.Set("start_date", startDate)
	params.Set("end_date", endDate)

	calendarResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/"+shopID+"/equipment-services/calendar?"+params.Encode(), nil, "user-1")
	require.Equal(t, http.StatusOK, calendarResp.Code)

	calendarData := decodeStandardResponse(t, calendarResp.Body)
	payload := decodeMap(t, calendarData.Data)
	services, ok := payload["services"].([]interface{})
	require.True(t, ok)
	require.Len(t, services, 1)
}
