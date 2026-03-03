package docs_equipment_test

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"miltechserver/api/docs_equipment"
	"miltechserver/api/middleware"

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
	return newTestRouterWithDB(t, testDB)
}

func newTestRouterWithDB(t *testing.T, db *sql.DB) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.ErrorHandler)
	publicGroup := router.Group("/api/v1")
	// Register only data routes (no blob client in test)
	docs_equipment.RegisterRoutes(docs_equipment.Dependencies{DB: db, BlobClient: nil}, publicGroup)
	return router
}

func doJSONRequest(t *testing.T, router *gin.Engine, method string, path string) *httptest.ResponseRecorder {
	t.Helper()
	req, err := http.NewRequest(method, path, strings.NewReader(""))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func hasRelation(t *testing.T, db *sql.DB, relation string) bool {
	t.Helper()
	var exists bool
	err := db.QueryRow(`SELECT to_regclass($1) IS NOT NULL`, "public."+relation).Scan(&exists)
	require.NoError(t, err)
	return exists
}

func countRows(t *testing.T, db *sql.DB, relation string) int {
	t.Helper()
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM " + relation).Scan(&count)
	require.NoError(t, err)
	return count
}

func fetchSampleFamily(t *testing.T, db *sql.DB) (string, bool) {
	t.Helper()
	var family sql.NullString
	err := db.QueryRow("SELECT family FROM docs_equipment_details WHERE family IS NOT NULL LIMIT 1").Scan(&family)
	if err == sql.ErrNoRows || !family.Valid {
		return "", false
	}
	require.NoError(t, err)
	return family.String, true
}

func fetchSampleModel(t *testing.T, db *sql.DB) (string, bool) {
	t.Helper()
	var model sql.NullString
	err := db.QueryRow("SELECT model FROM docs_equipment_details WHERE model IS NOT NULL LIMIT 1").Scan(&model)
	if err == sql.ErrNoRows || !model.Valid {
		return "", false
	}
	require.NoError(t, err)
	return model.String, true
}
