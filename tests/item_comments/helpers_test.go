package item_comments_test

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"miltechserver/api/item_comments"
	"miltechserver/api/middleware"
	"miltechserver/bootstrap"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type standardResponse struct {
	Status  int             `json:"status"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func newTestRouter(t *testing.T) *gin.Engine {
	t.Helper()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.ErrorHandler)

	publicGroup := router.Group("/api/v1")
	authGroup := router.Group("/api/v1/auth")
	authGroup.Use(testUserMiddleware())

	item_comments.RegisterRoutes(item_comments.Dependencies{DB: testDB}, publicGroup, authGroup)

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

func doJSONRequest(t *testing.T, router *gin.Engine, method string, path string, body string, userID string) *httptest.ResponseRecorder {
	t.Helper()

	reader := strings.NewReader(body)

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

func createComment(t *testing.T, router *gin.Engine, niin string, text string, userID string, parentID *string) commentResponse {
	t.Helper()

	payload := map[string]interface{}{"text": text}
	if parentID != nil {
		payload["parent_id"] = *parentID
	}
	body, err := json.Marshal(payload)
	require.NoError(t, err)

	resp := doJSONRequest(t, router, http.MethodPost, "/api/v1/auth/items/"+niin+"/comments", string(body), userID)
	require.Equal(t, http.StatusCreated, resp.Code)

	var payloadResp standardResponse
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &payloadResp))
	var created commentResponse
	require.NoError(t, json.Unmarshal(payloadResp.Data, &created))
	return created
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

func ensureNsn(t *testing.T, db *sql.DB, niin string) {
	t.Helper()

	_, err := db.Exec(
		`INSERT INTO nsn (service, category, fsc, niin)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (niin) DO NOTHING`,
		"test-service",
		"test-category",
		"TEST",
		niin,
	)
	require.NoError(t, err)
}

func clearItemCommentsTables(t *testing.T, db *sql.DB) {
	t.Helper()

	_, err := db.Exec(
		`TRUNCATE TABLE
			item_comment_flags,
			item_comments
		RESTART IDENTITY CASCADE`,
	)
	require.NoError(t, err)
}

func hasRelation(t *testing.T, db *sql.DB, relation string) bool {
	t.Helper()

	var exists bool
	err := db.QueryRow(`SELECT to_regclass($1) IS NOT NULL`, "public."+relation).Scan(&exists)
	require.NoError(t, err)
	return exists
}
