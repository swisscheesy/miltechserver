package sb_700_20_test

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"miltechserver/api/middleware"
	sb700 "miltechserver/api/sb_700_20"

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
	sb700.RegisterRoutes(sb700.Dependencies{DB: db}, publicGroup)
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

func fetchSampleLIN(t *testing.T, db *sql.DB, tableName string) (string, bool) {
	t.Helper()
	if !hasRelation(t, db, tableName) {
		return "", false
	}
	var lin sql.NullString
	err := db.QueryRow("SELECT lin FROM " + tableName + " LIMIT 1").Scan(&lin)
	if err == sql.ErrNoRows {
		return "", false
	}
	require.NoError(t, err)
	if !lin.Valid {
		return "", false
	}
	return lin.String, true
}

func fetchSampleLinZmmLin(t *testing.T, db *sql.DB, tableName string) (string, bool) {
	t.Helper()
	if !hasRelation(t, db, tableName) {
		return "", false
	}
	var lin sql.NullString
	err := db.QueryRow("SELECT lin_zmm_lin FROM " + tableName + " LIMIT 1").Scan(&lin)
	if err == sql.ErrNoRows {
		return "", false
	}
	require.NoError(t, err)
	if !lin.Valid {
		return "", false
	}
	return lin.String, true
}

func fetchSampleNewLIN(t *testing.T, db *sql.DB, tableName string) (string, bool) {
	t.Helper()
	if !hasRelation(t, db, tableName) {
		return "", false
	}
	var val sql.NullString
	// Exclude values containing URL-special characters (e.g. '#') that cannot
	// be safely embedded in a path segment without percent-encoding.
	err := db.QueryRow(
		"SELECT new_lin FROM "+tableName+
			" WHERE new_lin IS NOT NULL AND new_lin ~ '^[A-Za-z0-9_=-]+$' LIMIT 1",
	).Scan(&val)
	if err == sql.ErrNoRows {
		return "", false
	}
	require.NoError(t, err)
	if !val.Valid {
		return "", false
	}
	return val.String, true
}

func fetchSampleSubLINAppH1(t *testing.T, db *sql.DB) (string, bool) {
	t.Helper()
	if !hasRelation(t, db, "sb_700_20_app_h1") {
		return "", false
	}
	var val sql.NullString
	err := db.QueryRow("SELECT lin_zmm_sublin FROM sb_700_20_app_h1 LIMIT 1").Scan(&val)
	if err == sql.ErrNoRows {
		return "", false
	}
	require.NoError(t, err)
	if !val.Valid {
		return "", false
	}
	return val.String, true
}

func fetchSampleSubLINAppH2(t *testing.T, db *sql.DB) (string, bool) {
	t.Helper()
	if !hasRelation(t, db, "sb_700_20_app_h2") {
		return "", false
	}
	var val sql.NullString
	err := db.QueryRow("SELECT lin_zmmsublin FROM sb_700_20_app_h2 LIMIT 1").Scan(&val)
	if err == sql.ErrNoRows {
		return "", false
	}
	require.NoError(t, err)
	if !val.Valid {
		return "", false
	}
	return val.String, true
}

func fetchSampleRIC(t *testing.T, db *sql.DB, tableName string) (string, bool) {
	t.Helper()
	if !hasRelation(t, db, tableName) {
		return "", false
	}
	var val sql.NullString
	err := db.QueryRow("SELECT ric FROM " + tableName + " WHERE ric IS NOT NULL LIMIT 1").Scan(&val)
	if err == sql.ErrNoRows {
		return "", false
	}
	require.NoError(t, err)
	if !val.Valid {
		return "", false
	}
	return val.String, true
}
