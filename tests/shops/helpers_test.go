package shops_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"miltechserver/api/middleware"
	"miltechserver/api/shops"
	"miltechserver/bootstrap"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type standardResponse struct {
	Status  int             `json:"status"`
	Data    json.RawMessage `json:"data"`
	Message string          `json:"message"`
}

func newTestRouter(t *testing.T) *gin.Engine {
	t.Helper()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.ErrorHandler)
	router.Use(testUserMiddleware())

	group := router.Group("/api/v1/auth")

	deps := shops.Dependencies{
		DB:         testDB,
		BlobClient: (*azblob.Client)(nil),
		Env:        &bootstrap.Env{BlobAccountName: "test-account"},
	}

	shops.RegisterRoutes(deps, group)

	return router
}

func testUserMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetHeader("X-User-ID")
		if userID == "" {
			c.Next()
			return
		}

		user := &bootstrap.User{
			UserID:   userID,
			Username: c.GetHeader("X-User-Name"),
			Email:    c.GetHeader("X-User-Email"),
			Role:     "user",
		}

		if user.Username == "" {
			user.Username = "test-user"
		}
		if user.Email == "" {
			user.Email = userID + "@example.com"
		}

		c.Set("user", user)
		c.Next()
	}
}

func doJSONRequest(t *testing.T, router *gin.Engine, method string, path string, body interface{}, userID string) *httptest.ResponseRecorder {
	t.Helper()

	var reader *strings.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		require.NoError(t, err)
		reader = strings.NewReader(string(payload))
	} else {
		reader = strings.NewReader("")
	}

	req, err := http.NewRequest(method, path, reader)
	require.NoError(t, err)

	req.Header.Set("Content-Type", "application/json")
	if userID != "" {
		req.Header.Set("X-User-ID", userID)
	}

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func decodeStandardResponse(t *testing.T, body *bytes.Buffer) standardResponse {
	t.Helper()

	var resp standardResponse
	err := json.Unmarshal(body.Bytes(), &resp)
	require.NoError(t, err)
	return resp
}

func decodeMap(t *testing.T, data json.RawMessage) map[string]interface{} {
	t.Helper()

	var result map[string]interface{}
	err := json.Unmarshal(data, &result)
	require.NoError(t, err)
	return result
}

func decodeSlice(t *testing.T, data json.RawMessage) []interface{} {
	t.Helper()

	var result []interface{}
	err := json.Unmarshal(data, &result)
	require.NoError(t, err)
	return result
}

func ensureUser(t *testing.T, db *sql.DB, userID string) {
	t.Helper()

	now := time.Now().UTC()
	_, err := db.Exec(
		`INSERT INTO users (uid, email, username, created_at, is_enabled)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (uid) DO NOTHING`,
		userID,
		userID+"@example.com",
		"test-user",
		now,
		true,
	)
	require.NoError(t, err)
}

func clearShopTables(t *testing.T, db *sql.DB) {
	t.Helper()

	_, err := db.Exec(
		`TRUNCATE TABLE
			shop_notification_items,
			shop_vehicle_notification_changes,
			shop_vehicle_notifications,
			shop_vehicle,
			shop_list_items,
			shop_lists,
			shop_messages,
			shop_invite_codes,
			shop_members,
			shops
		RESTART IDENTITY CASCADE`,
	)
	require.NoError(t, err)
}

func createShop(t *testing.T, router *gin.Engine, userID string, name string) string {
	t.Helper()

	createBody := map[string]interface{}{
		"name":    name,
		"details": "Details",
	}

	createResp := doJSONRequest(t, router, http.MethodPost, "/api/v1/auth/shops", createBody, userID)
	require.Equal(t, http.StatusCreated, createResp.Code)

	created := decodeStandardResponse(t, createResp.Body)
	shopData := decodeMap(t, created.Data)
	shopID, ok := shopData["id"].(string)
	require.True(t, ok)
	require.NotEmpty(t, shopID)

	return shopID
}

func createVehicle(t *testing.T, router *gin.Engine, userID string, shopID string) string {
	t.Helper()

	vehicleBody := map[string]interface{}{
		"shop_id": shopID,
		"admin":   "admin",
	}

	createResp := doJSONRequest(t, router, http.MethodPost, "/api/v1/auth/shops/vehicles", vehicleBody, userID)
	require.Equal(t, http.StatusCreated, createResp.Code)

	created := decodeStandardResponse(t, createResp.Body)
	vehicleData := decodeMap(t, created.Data)
	vehicleID, ok := vehicleData["id"].(string)
	require.True(t, ok)
	require.NotEmpty(t, vehicleID)

	return vehicleID
}

func createList(t *testing.T, router *gin.Engine, userID string, shopID string) string {
	t.Helper()

	body := map[string]interface{}{
		"shop_id":     shopID,
		"description": "Test list",
	}

	resp := doJSONRequest(t, router, http.MethodPost, "/api/v1/auth/shops/lists", body, userID)
	require.Equal(t, http.StatusCreated, resp.Code)

	data := decodeMap(t, decodeStandardResponse(t, resp.Body).Data)
	listID, ok := data["id"].(string)
	require.True(t, ok)
	require.NotEmpty(t, listID)

	return listID
}

func createListItem(t *testing.T, router *gin.Engine, userID string, listID string, niin string, nomenclature string) string {
	t.Helper()

	body := map[string]interface{}{
		"list_id":         listID,
		"niin":            niin,
		"nomenclature":    nomenclature,
		"quantity":        1,
		"unit_of_measure": "ea",
	}

	resp := doJSONRequest(t, router, http.MethodPost, "/api/v1/auth/shops/lists/items", body, userID)
	require.Equal(t, http.StatusCreated, resp.Code)

	data := decodeMap(t, decodeStandardResponse(t, resp.Body).Data)
	itemID, ok := data["id"].(string)
	require.True(t, ok)
	require.NotEmpty(t, itemID)

	return itemID
}

func createNotification(t *testing.T, router *gin.Engine, userID string, shopID string, vehicleID string, title string) string {
	t.Helper()

	notificationBody := map[string]interface{}{
		"shop_id":     shopID,
		"vehicle_id":  vehicleID,
		"title":       title,
		"description": "desc",
		"type":        "PM",
	}

	createResp := doJSONRequest(t, router, http.MethodPost, "/api/v1/auth/shops/vehicles/notifications", notificationBody, userID)
	require.Equal(t, http.StatusCreated, createResp.Code)

	created := decodeStandardResponse(t, createResp.Body)
	notificationData := decodeMap(t, created.Data)
	notificationID, ok := notificationData["id"].(string)
	require.True(t, ok)
	require.NotEmpty(t, notificationID)

	return notificationID
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

func createMessage(t *testing.T, router *gin.Engine, userID string, shopID string, message string) string {
	t.Helper()

	messageBody := map[string]interface{}{
		"shop_id": shopID,
		"message": message,
	}

	createResp := doJSONRequest(t, router, http.MethodPost, "/api/v1/auth/shops/messages", messageBody, userID)
	require.Equal(t, http.StatusCreated, createResp.Code)

	created := decodeStandardResponse(t, createResp.Body)
	messageData := decodeMap(t, created.Data)
	messageID, ok := messageData["id"].(string)
	require.True(t, ok)
	require.NotEmpty(t, messageID)

	return messageID
}

func createPmcsInspection(t *testing.T, db *sql.DB, equipmentID string, guideManual string, performedDate time.Time, performedBy string) string {
	t.Helper()

	id := uuid.New().String()
	now := time.Now().UTC()
	_, err := db.Exec(
		`INSERT INTO pmcs_sbs_inspections (id, equipment_id, guide_manual, performed_date, performed_by, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $6)`,
		id, equipmentID, guideManual, performedDate, performedBy, now,
	)
	require.NoError(t, err)
	return id
}

func createPmcsFault(t *testing.T, db *sql.DB, pmcsID string, sectionID string, itemIndex int) {
	t.Helper()

	now := time.Now().UTC()
	_, err := db.Exec(
		`INSERT INTO pmcs_sbs_faults (pmcs_id, section_id, item_index, item_no, status, fault_text, corrective_action, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $8)`,
		pmcsID, sectionID, itemIndex, "1", "x", "test fault", "", now,
	)
	require.NoError(t, err)
}
