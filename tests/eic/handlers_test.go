package eic_test

import (
	"database/sql"
	"encoding/json"
	"math"
	"net/http"
	"testing"

	"miltechserver/api/response"

	"github.com/stretchr/testify/require"
)

func TestEICBlankParams(t *testing.T) {
	router := newTestRouter(t)

	blankNiinResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/eic/niin/%20%20")
	require.Equal(t, http.StatusBadRequest, blankNiinResp.Code)

	blankLinResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/eic/lin/%20%20")
	require.Equal(t, http.StatusBadRequest, blankLinResp.Code)

	blankFscResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/eic/fsc/%20%20")
	require.Equal(t, http.StatusBadRequest, blankFscResp.Code)

	invalidPageResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/eic/fsc/ABCD?page=bad")
	require.Equal(t, http.StatusBadRequest, invalidPageResp.Code)

	zeroPageResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/eic/items?page=0")
	require.Equal(t, http.StatusBadRequest, zeroPageResp.Code)
}

func TestEICLookupByNIINAndLIN(t *testing.T) {
	router := newTestRouter(t)

	if !hasRelation(t, testDB, "eic") {
		t.Skip("eic table missing in test DB")
	}

	rowCount := countRows(t, testDB, "eic")
	niinValue, linValue, _, ok := fetchEicSample(t, testDB)
	if ok {
		byNiinResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/eic/niin/"+niinValue)
		require.Equal(t, http.StatusOK, byNiinResp.Code)

		var payload standardResponse
		require.NoError(t, json.Unmarshal(byNiinResp.Body.Bytes(), &payload))
		var data response.EICSearchResponse
		require.NoError(t, json.Unmarshal(payload.Data, &data))
		require.NotEmpty(t, data.Items)

		byLinResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/eic/lin/"+linValue)
		require.Equal(t, http.StatusOK, byLinResp.Code)

		require.NoError(t, json.Unmarshal(byLinResp.Body.Bytes(), &payload))
		require.NoError(t, json.Unmarshal(payload.Data, &data))
		require.NotEmpty(t, data.Items)
	} else if rowCount == 0 {
		byNiinResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/eic/niin/TEST")
		require.Equal(t, http.StatusNotFound, byNiinResp.Code)
	}
}

func TestEICBrowseByFSC(t *testing.T) {
	router := newTestRouter(t)

	if !hasRelation(t, testDB, "eic") {
		t.Skip("eic table missing in test DB")
	}

	_, _, fscValue, ok := fetchEicSample(t, testDB)
	resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/eic/fsc/"+fscValue+"?page=1")
	if ok {
		require.Equal(t, http.StatusOK, resp.Code)

		var payload standardResponse
		require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &payload))
		var data response.EICPageResponse
		require.NoError(t, json.Unmarshal(payload.Data, &data))
		require.NotEmpty(t, data.Items)
	} else {
		require.Equal(t, http.StatusNotFound, resp.Code)
	}
}

func TestEICBrowseAll(t *testing.T) {
	router := newTestRouter(t)

	if !hasRelation(t, testDB, "eic") {
		t.Skip("eic table missing in test DB")
	}

	resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/eic/items?page=1")
	consolidatedCount := countConsolidatedEic(t, testDB, "")
	if consolidatedCount == 0 {
		require.Equal(t, http.StatusNotFound, resp.Code)
		return
	}

	require.Equal(t, http.StatusOK, resp.Code)
	var payload standardResponse
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &payload))
	var data response.EICPageResponse
	require.NoError(t, json.Unmarshal(payload.Data, &data))
	require.NotEmpty(t, data.Items)

	expectedTotalPages := int(math.Ceil(float64(consolidatedCount) / 40.0))
	require.Equal(t, 1, data.Page)
	require.Equal(t, expectedTotalPages, data.TotalPages)
	require.Equal(t, 1 >= expectedTotalPages, data.IsLastPage)
}

func TestEICSearchAll(t *testing.T) {
	router := newTestRouter(t)

	if !hasRelation(t, testDB, "eic") {
		t.Skip("eic table missing in test DB")
	}

	searchValue := "TEST"
	if niinValue, _, _, ok := fetchEicSample(t, testDB); ok {
		searchValue = niinValue
	}

	resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/eic/items?page=1&search="+searchValue)
	require.Equal(t, http.StatusInternalServerError, resp.Code)

	var payload response.ErrorResponse
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &payload))
	require.Equal(t, http.StatusInternalServerError, payload.Status)
}

func TestEICInternalError(t *testing.T) {
	db, err := sql.Open("postgres", "postgres://invalid:invalid@localhost:1/invalid?sslmode=disable")
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = db.Close()
	})

	router := newTestRouterWithDB(t, db)
	resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/eic/items?page=1")
	require.Equal(t, http.StatusInternalServerError, resp.Code)

	var payload response.ErrorResponse
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &payload))
	require.Equal(t, http.StatusInternalServerError, payload.Status)
}
