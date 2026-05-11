package tmde_test

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"miltechserver/api/middleware"
	"miltechserver/api/tmde"

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
	tmde.RegisterRoutes(tmde.Dependencies{DB: db}, publicGroup)
	return router
}

func doJSONRequest(t *testing.T, router *gin.Engine, method, path string) *httptest.ResponseRecorder {
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

func fetchTmdeSample(t *testing.T, db *sql.DB) (string, bool) {
	t.Helper()
	if !hasRelation(t, db, "tmde_interval_mat") {
		return "", false
	}
	var niin sql.NullString
	err := db.QueryRow("SELECT niin FROM tmde_interval_mat LIMIT 1").Scan(&niin)
	if err == sql.ErrNoRows {
		return "", false
	}
	require.NoError(t, err)
	if !niin.Valid {
		return "", false
	}
	return niin.String, true
}
