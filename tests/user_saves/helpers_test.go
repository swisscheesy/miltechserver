package user_saves_test

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
	"miltechserver/api/user_saves"
	"miltechserver/bootstrap"
	"miltechserver/tests/testutil"

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
	router.Use(testutil.FakeAuthMiddleware())

	group := router.Group("/api/v1/auth")

	deps := user_saves.Dependencies{
		DB:         testDB,
		BlobClient: (*azblob.Client)(nil),
		Env:        &bootstrap.Env{BlobAccountName: "test-account"},
	}

	user_saves.RegisterRoutes(deps, group)

	return router
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

func clearUserSavesTables(t *testing.T, db *sql.DB) {
	t.Helper()

	_, err := db.Exec(
		`TRUNCATE TABLE
			user_items_categorized,
			user_item_category,
			user_items_quick,
			user_items_serialized
		RESTART IDENTITY CASCADE`,
	)
	require.NoError(t, err)
}
