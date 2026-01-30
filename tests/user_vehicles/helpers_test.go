package user_vehicles_test

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
	"miltechserver/api/user_vehicles"
	"miltechserver/bootstrap"

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

	deps := user_vehicles.Dependencies{DB: testDB}
	user_vehicles.RegisterRoutes(deps, group)

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

func doRawRequest(t *testing.T, router *gin.Engine, method string, path string, body string, userID string) *httptest.ResponseRecorder {
	t.Helper()

	req, err := http.NewRequest(method, path, strings.NewReader(body))
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

func clearUserVehicleTables(t *testing.T, db *sql.DB) {
	t.Helper()

	_, err := db.Exec(
		`TRUNCATE TABLE
			user_notification_items,
			user_vehicle_notifications,
			user_vehicle
		RESTART IDENTITY CASCADE`,
	)
	require.NoError(t, err)
}

func insertVehicleRow(t *testing.T, db *sql.DB, vehicleID string, userID string, now time.Time) {
	t.Helper()

	_, err := db.Exec(
		`INSERT INTO user_vehicle (id, user_id, niin, admin, model, serial, uoc, mileage, hours, comment, save_time, last_updated)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
		vehicleID,
		userID,
		"N-1",
		"admin",
		"Model",
		"SER-1",
		"UNK",
		int32(5),
		int32(2),
		"comment",
		now,
		now,
	)
	require.NoError(t, err)
}

func insertNotificationRow(t *testing.T, db *sql.DB, notificationID string, userID string, vehicleID string, now time.Time) {
	t.Helper()

	_, err := db.Exec(
		`INSERT INTO user_vehicle_notifications (id, user_id, vehicle_id, title, description, type, completed, save_time, last_updated)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		notificationID,
		userID,
		vehicleID,
		"Title",
		"Desc",
		"PM",
		false,
		now,
		now,
	)
	require.NoError(t, err)
}

func insertNotificationItemRow(t *testing.T, db *sql.DB, itemID string, userID string, notificationID string, now time.Time) {
	t.Helper()

	_, err := db.Exec(
		`INSERT INTO user_notification_items (id, user_id, notification_id, niin, nomenclature, quantity, save_time)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		itemID,
		userID,
		notificationID,
		"I-1",
		"Item",
		int32(2),
		now,
	)
	require.NoError(t, err)
}
