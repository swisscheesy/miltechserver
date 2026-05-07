package tmde_test

import (
	"database/sql"
	"encoding/json"
	"math"
	"net/http"
	"testing"

	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/response"
	"miltechserver/api/tmde"

	"github.com/stretchr/testify/require"
)

func TestTmdeBlankParams(t *testing.T) {
	router := newTestRouter(t)

	blankNiinResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/tmde/niin/%20%20")
	require.Equal(t, http.StatusBadRequest, blankNiinResp.Code)

	invalidPageResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/tmde/requirements?page=bad")
	require.Equal(t, http.StatusBadRequest, invalidPageResp.Code)

	zeroPageResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/tmde/requirements?page=0")
	require.Equal(t, http.StatusBadRequest, zeroPageResp.Code)
}

func TestTmdeLookupByNIIN(t *testing.T) {
	router := newTestRouter(t)

	if !hasRelation(t, testDB, "tmde_requirements") {
		t.Skip("tmde_requirements table missing in test DB")
	}

	rowCount := countRows(t, testDB, "tmde_requirements")
	niinValue, ok := fetchTmdeSample(t, testDB)

	if ok {
		resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/tmde/niin/"+niinValue)
		require.Equal(t, http.StatusOK, resp.Code)

		var payload standardResponse
		require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &payload))

		var data model.TmdeRequirements
		require.NoError(t, json.Unmarshal(payload.Data, &data))
		require.Equal(t, niinValue, data.Niin)
	} else if rowCount == 0 {
		resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/tmde/niin/TEST")
		require.Equal(t, http.StatusNotFound, resp.Code)
	}
}

func TestTmdeNiinNotFound(t *testing.T) {
	router := newTestRouter(t)

	if !hasRelation(t, testDB, "tmde_requirements") {
		t.Skip("tmde_requirements table missing in test DB")
	}

	resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/tmde/niin/000000000NOTREAL")
	require.Equal(t, http.StatusNotFound, resp.Code)

	var payload response.NoItemFoundResponse
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &payload))
	require.Equal(t, http.StatusNotFound, payload.Status)
}

func TestTmdeListPaginated(t *testing.T) {
	router := newTestRouter(t)

	if !hasRelation(t, testDB, "tmde_requirements") {
		t.Skip("tmde_requirements table missing in test DB")
	}

	rowCount := countRows(t, testDB, "tmde_requirements")
	resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/tmde/requirements?page=1")

	if rowCount == 0 {
		require.Equal(t, http.StatusNotFound, resp.Code)
		return
	}

	require.Equal(t, http.StatusOK, resp.Code)

	var payload standardResponse
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &payload))

	var data tmde.TmdePageResponse
	require.NoError(t, json.Unmarshal(payload.Data, &data))
	require.NotEmpty(t, data.Items)
	require.Equal(t, 1, data.Page)

	expectedTotalPages := int(math.Ceil(float64(rowCount) / 100.0))
	require.Equal(t, expectedTotalPages, data.TotalPages)
	require.Equal(t, 1 >= expectedTotalPages, data.IsLastPage)
}

func TestTmdeListDefaultPage(t *testing.T) {
	router := newTestRouter(t)

	if !hasRelation(t, testDB, "tmde_requirements") {
		t.Skip("tmde_requirements table missing in test DB")
	}

	rowCount := countRows(t, testDB, "tmde_requirements")

	// omitting ?page should default to page 1
	resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/tmde/requirements")
	if rowCount == 0 {
		require.Equal(t, http.StatusNotFound, resp.Code)
		return
	}
	require.Equal(t, http.StatusOK, resp.Code)
}

func TestTmdeListBeyondLastPage(t *testing.T) {
	router := newTestRouter(t)

	if !hasRelation(t, testDB, "tmde_requirements") {
		t.Skip("tmde_requirements table missing in test DB")
	}

	resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/tmde/requirements?page=999999")
	require.Equal(t, http.StatusNotFound, resp.Code)
}

func TestTmdeInternalError(t *testing.T) {
	db, err := sql.Open("postgres", "postgres://invalid:invalid@localhost:1/invalid?sslmode=disable")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	router := newTestRouterWithDB(t, db)

	resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/tmde/requirements?page=1")
	require.Equal(t, http.StatusInternalServerError, resp.Code)

	var payload response.ErrorResponse
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &payload))
	require.Equal(t, http.StatusInternalServerError, payload.Status)
}
