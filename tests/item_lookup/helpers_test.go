package item_lookup_test

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"miltechserver/api/item_lookup"
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

	item_lookup.RegisterRoutes(item_lookup.Dependencies{DB: db}, publicGroup)

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

func fetchLinSample(t *testing.T, db *sql.DB) (string, string, bool) {
	t.Helper()

	if !hasRelation(t, db, "lookup_lin_niin_mat") {
		return "", "", false
	}

	var lin sql.NullString
	var niin sql.NullString
	err := db.QueryRow("SELECT lin, niin FROM lookup_lin_niin_mat LIMIT 1").Scan(&lin, &niin)
	if err == sql.ErrNoRows {
		return "", "", false
	}
	require.NoError(t, err)
	if !lin.Valid || !niin.Valid {
		return "", "", false
	}
	return lin.String, niin.String, true
}

func fetchUocSample(t *testing.T, db *sql.DB) (string, string, bool) {
	t.Helper()

	if !hasRelation(t, db, "lookup_uoc") {
		return "", "", false
	}

	var uoc sql.NullString
	var model sql.NullString
	err := db.QueryRow("SELECT uoc, model FROM lookup_uoc LIMIT 1").Scan(&uoc, &model)
	if err == sql.ErrNoRows {
		return "", "", false
	}
	require.NoError(t, err)
	if !uoc.Valid || !model.Valid {
		return "", "", false
	}
	return uoc.String, model.String, true
}

func fetchCageSample(t *testing.T, db *sql.DB) (string, bool) {
	t.Helper()

	if !hasRelation(t, db, "cage_address") {
		return "", false
	}

	var cage sql.NullString
	err := db.QueryRow("SELECT cage_code FROM cage_address LIMIT 1").Scan(&cage)
	if err == sql.ErrNoRows {
		return "", false
	}
	require.NoError(t, err)
	if !cage.Valid {
		return "", false
	}
	return cage.String, true
}
