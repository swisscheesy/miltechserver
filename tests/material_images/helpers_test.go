package material_images_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"miltechserver/api/material_images"
	"miltechserver/api/middleware"
	"miltechserver/bootstrap"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func newTestRouter(t *testing.T) *gin.Engine {
	t.Helper()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.ErrorHandler)
	router.Use(testUserMiddleware())

	publicGroup := router.Group("/api/v1")
	authGroup := router.Group("/api/v1/auth")

	deps := material_images.Dependencies{
		DB:         testDB,
		BlobClient: (*azblob.Client)(nil),
		Env:        &bootstrap.Env{BlobAccountName: "test-account"},
		AuthClient: nil,
	}

	material_images.RegisterRoutes(deps, publicGroup, authGroup)

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

func doMultipartRequest(t *testing.T, router *gin.Engine, method string, path string, fields map[string]string, fileField string, filename string, fileData []byte, userID string) *httptest.ResponseRecorder {
	t.Helper()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	for key, value := range fields {
		err := writer.WriteField(key, value)
		require.NoError(t, err)
	}

	part, err := writer.CreateFormFile(fileField, filename)
	require.NoError(t, err)
	_, err = part.Write(fileData)
	require.NoError(t, err)

	err = writer.Close()
	require.NoError(t, err)

	req, err := http.NewRequest(method, path, body)
	require.NoError(t, err)

	req.Header.Set("Content-Type", writer.FormDataContentType())
	if userID != "" {
		req.Header.Set("X-User-ID", userID)
	}

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
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

func clearMaterialImagesTables(t *testing.T, db *sql.DB) {
	t.Helper()

	if !hasTable(t, db, "material_images_upload_limits") {
		t.Skip("material_images_upload_limits table missing in test DB")
	}

	_, err := db.Exec(
		`TRUNCATE TABLE
			material_images_flags,
			material_images_votes,
			material_images_upload_limits,
			material_images
		RESTART IDENTITY CASCADE`,
	)
	require.NoError(t, err)
}

func hasTable(t *testing.T, db *sql.DB, tableName string) bool {
	t.Helper()

	var exists bool
	err := db.QueryRow(`SELECT to_regclass($1) IS NOT NULL`, "public."+tableName).Scan(&exists)
	require.NoError(t, err)
	return exists
}
