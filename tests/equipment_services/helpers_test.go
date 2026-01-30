package equipment_services_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"miltechserver/api/equipment_services"
	"miltechserver/api/middleware"
	"miltechserver/api/shops"
	"miltechserver/bootstrap"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/gin-gonic/gin"
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

	shops.RegisterRoutes(shops.Dependencies{
		DB:         testDB,
		BlobClient: (*azblob.Client)(nil),
		Env:        &bootstrap.Env{BlobAccountName: "test-account"},
	}, group)

	equipment_services.RegisterRoutes(equipment_services.Dependencies{DB: testDB}, group)

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

func clearEquipmentServicesTables(t *testing.T, db *sql.DB) {
	t.Helper()

	_, err := db.Exec(
		`TRUNCATE TABLE
			equipment_services,
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

	admin := fmt.Sprintf("admin-%d", time.Now().UnixNano())
	vehicleBody := map[string]interface{}{
		"shop_id": shopID,
		"admin":   admin,
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

	listBody := map[string]interface{}{
		"shop_id":     shopID,
		"description": "Parts list",
	}

	createResp := doJSONRequest(t, router, http.MethodPost, "/api/v1/auth/shops/lists", listBody, userID)
	require.Equal(t, http.StatusCreated, createResp.Code)

	created := decodeStandardResponse(t, createResp.Body)
	listData := decodeMap(t, created.Data)
	listID, ok := listData["id"].(string)
	require.True(t, ok)
	require.NotEmpty(t, listID)

	return listID
}

func createEquipmentService(t *testing.T, router *gin.Engine, userID, shopID, equipmentID, listID, description string, serviceDate *time.Time, isCompleted bool) string {
	t.Helper()

	body := map[string]interface{}{
		"equipment_id": equipmentID,
		"list_id":      listID,
		"description":  description,
		"service_type": "inspection",
		"is_completed": isCompleted,
		"service_date": serviceDate,
	}

	createResp := doJSONRequest(t, router, http.MethodPost, "/api/v1/auth/shops/"+shopID+"/equipment-services", body, userID)
	require.Equal(t, http.StatusCreated, createResp.Code)

	created := decodeStandardResponse(t, createResp.Body)
	serviceData := decodeMap(t, created.Data)
	serviceID, ok := serviceData["id"].(string)
	require.True(t, ok)
	require.NotEmpty(t, serviceID)

	return serviceID
}
