package equipment_services_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestEquipmentServicesQueriesDoNotDuplicateForMultipleShopMembers(t *testing.T) {
	clearEquipmentServicesTables(t, testDB)
	ensureUser(t, testDB, "owner")
	ensureUser(t, testDB, "member")
	router := newTestRouter(t)

	shopID := createShop(t, router, "owner", "Multi Member Services")
	_, inviteCode := createInviteCode(t, router, "owner", shopID)
	joinResp := doJSONRequest(t, router, http.MethodPost, "/api/v1/auth/shops/join", map[string]any{"invite_code": inviteCode}, "member")
	require.Equal(t, http.StatusOK, joinResp.Code)

	equipmentID := createVehicle(t, router, "owner", shopID)
	serviceDate := time.Now().AddDate(0, 0, 3)
	createEquipmentService(t, router, "owner", shopID, equipmentID, "", "One service", &serviceDate, false)

	listResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/"+shopID+"/equipment-services", nil, "owner")
	require.Equal(t, http.StatusOK, listResp.Code)
	listPayload := decodeMap(t, decodeStandardResponse(t, listResp.Body).Data)
	require.Equal(t, float64(1), listPayload["total_count"])
	require.Len(t, listPayload["services"].([]interface{}), 1)

	equipmentResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/"+shopID+"/equipment/"+equipmentID+"/services", nil, "owner")
	require.Equal(t, http.StatusOK, equipmentResp.Code)
	equipmentPayload := decodeMap(t, decodeStandardResponse(t, equipmentResp.Body).Data)
	require.Equal(t, float64(1), equipmentPayload["total_count"])
	require.Len(t, equipmentPayload["services"].([]interface{}), 1)
}

func TestEquipmentServicesCalendarAndStatusDoNotDuplicateForMultipleShopMembers(t *testing.T) {
	clearEquipmentServicesTables(t, testDB)
	ensureUser(t, testDB, "owner")
	ensureUser(t, testDB, "member")
	router := newTestRouter(t)

	shopID := createShop(t, router, "owner", "Multi Member Calendar")
	_, inviteCode := createInviteCode(t, router, "owner", shopID)
	joinResp := doJSONRequest(t, router, http.MethodPost, "/api/v1/auth/shops/join", map[string]any{"invite_code": inviteCode}, "member")
	require.Equal(t, http.StatusOK, joinResp.Code)

	equipmentID := createVehicle(t, router, "owner", shopID)
	overdueDate := time.Now().AddDate(0, 0, -2)
	dueSoonDate := time.Now().AddDate(0, 0, 3)
	createEquipmentService(t, router, "owner", shopID, equipmentID, "", "Overdue", &overdueDate, false)
	createEquipmentService(t, router, "owner", shopID, equipmentID, "", "Due soon", &dueSoonDate, false)

	start := time.Now().AddDate(0, 0, -7).Format(time.RFC3339)
	end := time.Now().AddDate(0, 0, 7).Format(time.RFC3339)
	calendarResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/"+shopID+"/equipment-services/calendar?start_date="+start+"&end_date="+end, nil, "owner")
	require.Equal(t, http.StatusOK, calendarResp.Code)
	calendarPayload := decodeMap(t, decodeStandardResponse(t, calendarResp.Body).Data)
	require.Equal(t, float64(2), calendarPayload["total_count"])
	require.Len(t, calendarPayload["services"].([]interface{}), 2)

	overdueResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/"+shopID+"/equipment-services/overdue", nil, "owner")
	require.Equal(t, http.StatusOK, overdueResp.Code)
	overduePayload := decodeMap(t, decodeStandardResponse(t, overdueResp.Body).Data)
	require.Equal(t, float64(1), overduePayload["total_count"])
	require.Len(t, overduePayload["overdue_services"].([]interface{}), 1)

	dueSoonResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/"+shopID+"/equipment-services/due-soon", nil, "owner")
	require.Equal(t, http.StatusOK, dueSoonResp.Code)
	dueSoonPayload := decodeMap(t, decodeStandardResponse(t, dueSoonResp.Body).Data)
	require.Equal(t, float64(1), dueSoonPayload["total_count"])
	require.Len(t, dueSoonPayload["due_soon_services"].([]interface{}), 1)
}

func createInviteCode(t *testing.T, router *gin.Engine, userID string, shopID string) (string, string) {
	t.Helper()

	inviteBody := map[string]interface{}{
		"shop_id": shopID,
	}

	inviteResp := doJSONRequest(t, router, http.MethodPost, "/api/v1/auth/shops/invite-codes", inviteBody, userID)
	require.Equal(t, http.StatusCreated, inviteResp.Code)

	invite := decodeStandardResponse(t, inviteResp.Body)
	inviteData := decodeMap(t, invite.Data)
	codeID, ok := inviteData["id"].(string)
	require.True(t, ok)
	require.NotEmpty(t, codeID)

	code, ok := inviteData["code"].(string)
	require.True(t, ok)
	require.NotEmpty(t, code)

	return codeID, code
}
