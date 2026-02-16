package item_lookup_test

import (
	"database/sql"
	"encoding/json"
	"math"
	"net/http"
	"testing"

	"miltechserver/api/item_lookup/shared"
	"miltechserver/api/response"

	"github.com/stretchr/testify/require"
)

func TestItemLookupLinRoutes(t *testing.T) {
	router := newTestRouter(t)

	missingPage := doJSONRequest(t, router, http.MethodGet, "/api/v1/lookup/lin")
	require.Equal(t, http.StatusBadRequest, missingPage.Code)

	invalidPage := doJSONRequest(t, router, http.MethodGet, "/api/v1/lookup/lin?page=bad")
	require.Equal(t, http.StatusBadRequest, invalidPage.Code)

	zeroPage := doJSONRequest(t, router, http.MethodGet, "/api/v1/lookup/lin?page=0")
	require.Equal(t, http.StatusBadRequest, zeroPage.Code)

	if !hasRelation(t, testDB, "lookup_lin_niin_mat") {
		t.Skip("lookup_lin_niin_mat view missing in test DB")
	}

	rowCount := countRows(t, testDB, "lookup_lin_niin_mat")
	pageResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/lookup/lin?page=1")
	if rowCount == 0 {
		require.Equal(t, http.StatusNotFound, pageResp.Code)
	} else {
		require.Equal(t, http.StatusOK, pageResp.Code)
		var payload standardResponse
		require.NoError(t, json.Unmarshal(pageResp.Body.Bytes(), &payload))
		var data response.LINPageResponse
		require.NoError(t, json.Unmarshal(payload.Data, &data))
		require.NotEmpty(t, data.Lins)
	}

	linValue, niinValue, ok := fetchLinSample(t, testDB)
	if ok {
		byNiinResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/lookup/lin/by-niin/"+niinValue)
		require.Equal(t, http.StatusOK, byNiinResp.Code)

		var payload standardResponse
		require.NoError(t, json.Unmarshal(byNiinResp.Body.Bytes(), &payload))
		var data response.LINPageResponse
		require.NoError(t, json.Unmarshal(payload.Data, &data))
		require.NotEmpty(t, data.Lins)

		legacyByNiinResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/lookup/lin/lin/"+niinValue)
		require.Equal(t, http.StatusOK, legacyByNiinResp.Code)

		byLinResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/lookup/niin/by-lin/"+linValue)
		require.Equal(t, http.StatusOK, byLinResp.Code)

		require.NoError(t, json.Unmarshal(byLinResp.Body.Bytes(), &payload))
		require.NoError(t, json.Unmarshal(payload.Data, &data))
		require.NotEmpty(t, data.Lins)

		legacyByLinResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/lookup/lin/niin/"+linValue)
		require.Equal(t, http.StatusOK, legacyByLinResp.Code)
	} else if rowCount == 0 {
		byNiinResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/lookup/lin/by-niin/TEST")
		require.Equal(t, http.StatusNotFound, byNiinResp.Code)
	}
}

func TestItemLookupUocRoutes(t *testing.T) {
	router := newTestRouter(t)

	missingPage := doJSONRequest(t, router, http.MethodGet, "/api/v1/lookup/uoc")
	require.Equal(t, http.StatusBadRequest, missingPage.Code)

	invalidPage := doJSONRequest(t, router, http.MethodGet, "/api/v1/lookup/uoc?page=bad")
	require.Equal(t, http.StatusBadRequest, invalidPage.Code)

	zeroPage := doJSONRequest(t, router, http.MethodGet, "/api/v1/lookup/uoc?page=0")
	require.Equal(t, http.StatusBadRequest, zeroPage.Code)

	if !hasRelation(t, testDB, "lookup_uoc") {
		t.Skip("lookup_uoc table missing in test DB")
	}

	rowCount := countRows(t, testDB, "lookup_uoc")
	pageResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/lookup/uoc?page=1")
	if rowCount == 0 {
		require.Equal(t, http.StatusNotFound, pageResp.Code)
	} else {
		require.Equal(t, http.StatusOK, pageResp.Code)
		var payload standardResponse
		require.NoError(t, json.Unmarshal(pageResp.Body.Bytes(), &payload))
		var data response.UOCPageResponse
		require.NoError(t, json.Unmarshal(payload.Data, &data))
		require.NotEmpty(t, data.UOCs)
	}

	uocValue, modelValue, ok := fetchUocSample(t, testDB)
	if ok {
		byUocResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/lookup/uoc/"+uocValue)
		require.Equal(t, http.StatusOK, byUocResp.Code)

		var payload standardResponse
		require.NoError(t, json.Unmarshal(byUocResp.Body.Bytes(), &payload))
		var data response.UOCPageResponse
		require.NoError(t, json.Unmarshal(payload.Data, &data))
		require.NotEmpty(t, data.UOCs)

		byModelResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/lookup/uoc/by-model/"+modelValue)
		require.Equal(t, http.StatusOK, byModelResp.Code)

		require.NoError(t, json.Unmarshal(byModelResp.Body.Bytes(), &payload))
		require.NoError(t, json.Unmarshal(payload.Data, &data))
		require.NotEmpty(t, data.UOCs)

		legacyByModelResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/lookup/uoc/model/"+modelValue)
		require.Equal(t, http.StatusOK, legacyByModelResp.Code)
	} else if rowCount == 0 {
		byUocResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/lookup/uoc/TEST")
		require.Equal(t, http.StatusNotFound, byUocResp.Code)
	}
}

func TestItemLookupCageRoute(t *testing.T) {
	router := newTestRouter(t)

	blankCageResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/lookup/cage/%20%20")
	require.Equal(t, http.StatusBadRequest, blankCageResp.Code)

	if !hasRelation(t, testDB, "cage_address") {
		t.Skip("cage_address table missing in test DB")
	}

	rowCount := countRows(t, testDB, "cage_address")
	cageValue, ok := fetchCageSample(t, testDB)
	if ok {
		resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/lookup/cage/"+cageValue)
		require.Equal(t, http.StatusOK, resp.Code)
	} else if rowCount == 0 {
		resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/lookup/cage/TEST")
		require.Equal(t, http.StatusNotFound, resp.Code)
	}
}

func TestItemLookupSubstituteRoute(t *testing.T) {
	router := newTestRouter(t)

	if !hasRelation(t, testDB, "army_substitute_lin") {
		t.Skip("army_substitute_lin table missing in test DB")
	}

	rowCount := countRows(t, testDB, "army_substitute_lin")
	resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/lookup/substitute-lin")
	if rowCount == 0 {
		require.Equal(t, http.StatusNotFound, resp.Code)
	} else {
		require.Equal(t, http.StatusOK, resp.Code)
	}
}

func TestItemLookupUocBlankParams(t *testing.T) {
	router := newTestRouter(t)

	blankUocResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/lookup/uoc/%20")
	require.Equal(t, http.StatusBadRequest, blankUocResp.Code)

	blankModelResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/lookup/uoc/by-model/%20")
	require.Equal(t, http.StatusBadRequest, blankModelResp.Code)

	legacyBlankModelResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/lookup/uoc/model/%20")
	require.Equal(t, http.StatusBadRequest, legacyBlankModelResp.Code)
}

func TestItemLookupLinBlankParams(t *testing.T) {
	router := newTestRouter(t)

	blankNiinResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/lookup/lin/by-niin/%20")
	require.Equal(t, http.StatusBadRequest, blankNiinResp.Code)

	legacyBlankNiinResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/lookup/lin/lin/%20")
	require.Equal(t, http.StatusBadRequest, legacyBlankNiinResp.Code)

	blankLinResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/lookup/niin/by-lin/%20")
	require.Equal(t, http.StatusBadRequest, blankLinResp.Code)

	legacyBlankLinResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/lookup/lin/niin/%20")
	require.Equal(t, http.StatusBadRequest, legacyBlankLinResp.Code)
}

func TestItemLookupInternalError(t *testing.T) {
	db, err := sql.Open("postgres", "postgres://invalid:invalid@localhost:1/invalid?sslmode=disable")
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = db.Close()
	})

	router := newTestRouterWithDB(t, db)
	resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/lookup/lin?page=1")
	require.Equal(t, http.StatusInternalServerError, resp.Code)

	var payload response.ErrorResponse
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &payload))
	require.Equal(t, http.StatusInternalServerError, payload.Status)
}

func TestItemLookupPaginationMetadata(t *testing.T) {
	router := newTestRouter(t)

	if !hasRelation(t, testDB, "lookup_lin_niin_mat") {
		t.Skip("lookup_lin_niin_mat view missing in test DB")
	}

	total := countRows(t, testDB, "lookup_lin_niin_mat")
	if total == 0 {
		t.Skip("no LIN data available for pagination metadata test")
	}

	resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/lookup/lin?page=1")
	require.Equal(t, http.StatusOK, resp.Code)

	var payload standardResponse
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &payload))
	var data response.LINPageResponse
	require.NoError(t, json.Unmarshal(payload.Data, &data))

	expectedTotalPages := int(math.Ceil(float64(total) / float64(shared.DefaultPageSize)))
	require.Equal(t, total, data.Count)
	require.Equal(t, 1, data.Page)
	require.Equal(t, expectedTotalPages, data.TotalPages)
	require.Equal(t, 1 >= expectedTotalPages, data.IsLastPage)
}

func TestItemLookupUocPaginationMetadata(t *testing.T) {
	router := newTestRouter(t)

	if !hasRelation(t, testDB, "lookup_uoc") {
		t.Skip("lookup_uoc table missing in test DB")
	}

	total := countRows(t, testDB, "lookup_uoc")
	if total == 0 {
		t.Skip("no UOC data available for pagination metadata test")
	}

	resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/lookup/uoc?page=1")
	require.Equal(t, http.StatusOK, resp.Code)

	var payload standardResponse
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &payload))
	var data response.UOCPageResponse
	require.NoError(t, json.Unmarshal(payload.Data, &data))

	expectedTotalPages := int(math.Ceil(float64(total) / float64(shared.DefaultPageSize)))
	require.Equal(t, total, data.Count)
	require.Equal(t, 1, data.Page)
	require.Equal(t, expectedTotalPages, data.TotalPages)
	require.Equal(t, 1 >= expectedTotalPages, data.IsLastPage)
}
