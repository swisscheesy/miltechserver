package sb_700_20_test

import (
	"database/sql"
	"encoding/json"
	"math"
	"net/http"
	"testing"

	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/response"
	sb700 "miltechserver/api/sb_700_20"

	"github.com/stretchr/testify/require"
)

func TestBlankSearchParams(t *testing.T) {
	router := newTestRouter(t)
	endpoints := []string{
		"/api/v1/sb700-20/app-b/search/%20%20",
		"/api/v1/sb700-20/app-c/search/%20%20",
		"/api/v1/sb700-20/app-d/search/%20%20",
		"/api/v1/sb700-20/app-e/search/%20%20",
		"/api/v1/sb700-20/app-f/search/%20%20",
		"/api/v1/sb700-20/app-g/search/%20%20",
		"/api/v1/sb700-20/app-h1/search/%20%20",
		"/api/v1/sb700-20/app-h2/search/%20%20",
		"/api/v1/sb700-20/app-i/search/%20%20",
		"/api/v1/sb700-20/app-j/search/%20%20",
		"/api/v1/sb700-20/chp-4/search/%20%20",
		"/api/v1/sb700-20/chp-6/search/%20%20",
		"/api/v1/sb700-20/chp-8/search/%20%20",
		"/api/v1/sb700-20/app-e/search-new-lin/%20%20",
		"/api/v1/sb700-20/app-g/search-new-lin/%20%20",
		"/api/v1/sb700-20/app-h1/search-sublin/%20%20",
		"/api/v1/sb700-20/app-h2/search-sublin/%20%20",
		"/api/v1/sb700-20/chp-4/search-ric/%20%20",
		"/api/v1/sb700-20/chp-6/search-ric/%20%20",
		"/api/v1/sb700-20/chp-8/search-ric/%20%20",
	}
	for _, ep := range endpoints {
		resp := doJSONRequest(t, router, http.MethodGet, ep)
		require.Equal(t, http.StatusBadRequest, resp.Code, "endpoint: %s", ep)
	}
}

func TestInvalidPageParams(t *testing.T) {
	router := newTestRouter(t)
	endpoints := []string{
		"/api/v1/sb700-20/app-b/list",
		"/api/v1/sb700-20/app-c/list",
		"/api/v1/sb700-20/app-d/list",
		"/api/v1/sb700-20/app-e/list",
		"/api/v1/sb700-20/app-f/list",
		"/api/v1/sb700-20/app-g/list",
		"/api/v1/sb700-20/app-h1/list",
		"/api/v1/sb700-20/app-h2/list",
		"/api/v1/sb700-20/app-i/list",
		"/api/v1/sb700-20/app-j/list",
		"/api/v1/sb700-20/chp-4/list",
		"/api/v1/sb700-20/chp-6/list",
		"/api/v1/sb700-20/chp-8/list",
	}
	for _, ep := range endpoints {
		require.Equal(t, http.StatusBadRequest, doJSONRequest(t, router, http.MethodGet, ep+"?page=bad").Code, ep)
		require.Equal(t, http.StatusBadRequest, doJSONRequest(t, router, http.MethodGet, ep+"?page=0").Code, ep)
	}
}

func TestAppBEndpoints(t *testing.T) {
	router := newTestRouter(t)
	if !hasRelation(t, testDB, "sb_700_20_app_b") {
		t.Skip("sb_700_20_app_b missing in test DB")
	}
	rowCount := countRows(t, testDB, "sb_700_20_app_b")
	if linVal, ok := fetchSampleLIN(t, testDB, "sb_700_20_app_b"); ok {
		resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/sb700-20/app-b/search/"+linVal)
		require.Equal(t, http.StatusOK, resp.Code)
		var payload standardResponse
		require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &payload))
		var data []model.Sb70020AppB
		require.NoError(t, json.Unmarshal(payload.Data, &data))
		require.NotEmpty(t, data)
		require.Equal(t, linVal, data[0].Lin)
	}
	resp404 := doJSONRequest(t, router, http.MethodGet, "/api/v1/sb700-20/app-b/search/NOTREAL999")
	require.Equal(t, http.StatusNotFound, resp404.Code)
	var nf response.NoItemFoundResponse
	require.NoError(t, json.Unmarshal(resp404.Body.Bytes(), &nf))
	require.Equal(t, http.StatusNotFound, nf.Status)

	resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/sb700-20/app-b/list?page=1")
	if rowCount == 0 {
		require.Equal(t, http.StatusNotFound, resp.Code)
		return
	}
	require.Equal(t, http.StatusOK, resp.Code)
	var payload standardResponse
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &payload))
	var data sb700.PageResponse[model.Sb70020AppB]
	require.NoError(t, json.Unmarshal(payload.Data, &data))
	require.NotEmpty(t, data.Items)
	require.Equal(t, 1, data.Page)
	require.Equal(t, int(math.Ceil(float64(rowCount)/100.0)), data.TotalPages)
	require.Equal(t, http.StatusOK, doJSONRequest(t, router, http.MethodGet, "/api/v1/sb700-20/app-b/list").Code)
	require.Equal(t, http.StatusNotFound, doJSONRequest(t, router, http.MethodGet, "/api/v1/sb700-20/app-b/list?page=999999").Code)
}

func TestAppCEndpoints(t *testing.T) {
	router := newTestRouter(t)
	if !hasRelation(t, testDB, "sb_700_20_app_c") {
		t.Skip("sb_700_20_app_c missing in test DB")
	}
	rowCount := countRows(t, testDB, "sb_700_20_app_c")
	if linVal, ok := fetchSampleLIN(t, testDB, "sb_700_20_app_c"); ok {
		resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/sb700-20/app-c/search/"+linVal)
		require.Equal(t, http.StatusOK, resp.Code)
		var payload standardResponse
		require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &payload))
		var data model.Sb70020AppC
		require.NoError(t, json.Unmarshal(payload.Data, &data))
		require.Equal(t, linVal, data.Lin)
	}
	require.Equal(t, http.StatusNotFound, doJSONRequest(t, router, http.MethodGet, "/api/v1/sb700-20/app-c/search/NOTREAL999").Code)

	resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/sb700-20/app-c/list?page=1")
	if rowCount == 0 {
		require.Equal(t, http.StatusNotFound, resp.Code)
		return
	}
	require.Equal(t, http.StatusOK, resp.Code)
	var payload standardResponse
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &payload))
	var data sb700.PageResponse[model.Sb70020AppC]
	require.NoError(t, json.Unmarshal(payload.Data, &data))
	require.NotEmpty(t, data.Items)
	require.Equal(t, http.StatusOK, doJSONRequest(t, router, http.MethodGet, "/api/v1/sb700-20/app-c/list").Code)
	require.Equal(t, http.StatusNotFound, doJSONRequest(t, router, http.MethodGet, "/api/v1/sb700-20/app-c/list?page=999999").Code)
}

func TestAppDEndpoints(t *testing.T) {
	router := newTestRouter(t)
	if !hasRelation(t, testDB, "sb_700_20_app_d") {
		t.Skip("sb_700_20_app_d missing in test DB")
	}
	rowCount := countRows(t, testDB, "sb_700_20_app_d")
	if linVal, ok := fetchSampleLIN(t, testDB, "sb_700_20_app_d"); ok {
		resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/sb700-20/app-d/search/"+linVal)
		require.Equal(t, http.StatusOK, resp.Code)
		var payload standardResponse
		require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &payload))
		var data []model.Sb70020AppD
		require.NoError(t, json.Unmarshal(payload.Data, &data))
		require.NotEmpty(t, data)
	}
	require.Equal(t, http.StatusNotFound, doJSONRequest(t, router, http.MethodGet, "/api/v1/sb700-20/app-d/search/NOTREAL999").Code)

	resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/sb700-20/app-d/list?page=1")
	if rowCount == 0 {
		require.Equal(t, http.StatusNotFound, resp.Code)
		return
	}
	require.Equal(t, http.StatusOK, resp.Code)
	var payload standardResponse
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &payload))
	var data sb700.PageResponse[model.Sb70020AppD]
	require.NoError(t, json.Unmarshal(payload.Data, &data))
	require.NotEmpty(t, data.Items)
	require.Equal(t, http.StatusOK, doJSONRequest(t, router, http.MethodGet, "/api/v1/sb700-20/app-d/list").Code)
	require.Equal(t, http.StatusNotFound, doJSONRequest(t, router, http.MethodGet, "/api/v1/sb700-20/app-d/list?page=999999").Code)
}

func TestAppEEndpoints(t *testing.T) {
	router := newTestRouter(t)
	if !hasRelation(t, testDB, "sb_700_20_app_e") {
		t.Skip("sb_700_20_app_e missing in test DB")
	}
	rowCount := countRows(t, testDB, "sb_700_20_app_e")
	if linVal, ok := fetchSampleLIN(t, testDB, "sb_700_20_app_e"); ok {
		resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/sb700-20/app-e/search/"+linVal)
		require.Equal(t, http.StatusOK, resp.Code)
		var payload standardResponse
		require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &payload))
		var data []model.Sb70020AppE
		require.NoError(t, json.Unmarshal(payload.Data, &data))
		require.NotEmpty(t, data)
	}
	require.Equal(t, http.StatusNotFound, doJSONRequest(t, router, http.MethodGet, "/api/v1/sb700-20/app-e/search/NOTREAL999").Code)

	resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/sb700-20/app-e/list?page=1")
	if rowCount == 0 {
		require.Equal(t, http.StatusNotFound, resp.Code)
		return
	}
	require.Equal(t, http.StatusOK, resp.Code)
	var payload standardResponse
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &payload))
	var data sb700.PageResponse[model.Sb70020AppE]
	require.NoError(t, json.Unmarshal(payload.Data, &data))
	require.NotEmpty(t, data.Items)
	require.Equal(t, http.StatusOK, doJSONRequest(t, router, http.MethodGet, "/api/v1/sb700-20/app-e/list").Code)
	require.Equal(t, http.StatusNotFound, doJSONRequest(t, router, http.MethodGet, "/api/v1/sb700-20/app-e/list?page=999999").Code)
}

func TestAppFEndpoints(t *testing.T) {
	router := newTestRouter(t)
	if !hasRelation(t, testDB, "sb_700_20_app_f") {
		t.Skip("sb_700_20_app_f missing in test DB")
	}
	rowCount := countRows(t, testDB, "sb_700_20_app_f")
	if linVal, ok := fetchSampleLIN(t, testDB, "sb_700_20_app_f"); ok {
		resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/sb700-20/app-f/search/"+linVal)
		require.Equal(t, http.StatusOK, resp.Code)
		var payload standardResponse
		require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &payload))
		var data model.Sb70020AppF
		require.NoError(t, json.Unmarshal(payload.Data, &data))
		require.Equal(t, linVal, data.Lin)
	}
	require.Equal(t, http.StatusNotFound, doJSONRequest(t, router, http.MethodGet, "/api/v1/sb700-20/app-f/search/NOTREAL999").Code)

	resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/sb700-20/app-f/list?page=1")
	if rowCount == 0 {
		require.Equal(t, http.StatusNotFound, resp.Code)
		return
	}
	require.Equal(t, http.StatusOK, resp.Code)
	var payload standardResponse
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &payload))
	var data sb700.PageResponse[model.Sb70020AppF]
	require.NoError(t, json.Unmarshal(payload.Data, &data))
	require.NotEmpty(t, data.Items)
	require.Equal(t, http.StatusOK, doJSONRequest(t, router, http.MethodGet, "/api/v1/sb700-20/app-f/list").Code)
	require.Equal(t, http.StatusNotFound, doJSONRequest(t, router, http.MethodGet, "/api/v1/sb700-20/app-f/list?page=999999").Code)
}

func TestAppGEndpoints(t *testing.T) {
	router := newTestRouter(t)
	if !hasRelation(t, testDB, "sb_700_20_app_g") {
		t.Skip("sb_700_20_app_g missing in test DB")
	}
	rowCount := countRows(t, testDB, "sb_700_20_app_g")
	if linVal, ok := fetchSampleLIN(t, testDB, "sb_700_20_app_g"); ok {
		resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/sb700-20/app-g/search/"+linVal)
		require.Equal(t, http.StatusOK, resp.Code)
		var payload standardResponse
		require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &payload))
		var data model.Sb70020AppG
		require.NoError(t, json.Unmarshal(payload.Data, &data))
		require.Equal(t, linVal, data.Lin)
	}
	require.Equal(t, http.StatusNotFound, doJSONRequest(t, router, http.MethodGet, "/api/v1/sb700-20/app-g/search/NOTREAL999").Code)

	resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/sb700-20/app-g/list?page=1")
	if rowCount == 0 {
		require.Equal(t, http.StatusNotFound, resp.Code)
		return
	}
	require.Equal(t, http.StatusOK, resp.Code)
	var payload standardResponse
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &payload))
	var data sb700.PageResponse[model.Sb70020AppG]
	require.NoError(t, json.Unmarshal(payload.Data, &data))
	require.NotEmpty(t, data.Items)
	require.Equal(t, http.StatusOK, doJSONRequest(t, router, http.MethodGet, "/api/v1/sb700-20/app-g/list").Code)
	require.Equal(t, http.StatusNotFound, doJSONRequest(t, router, http.MethodGet, "/api/v1/sb700-20/app-g/list?page=999999").Code)
}

func TestAppH1Endpoints(t *testing.T) {
	router := newTestRouter(t)
	if !hasRelation(t, testDB, "sb_700_20_app_h1") {
		t.Skip("sb_700_20_app_h1 missing in test DB")
	}
	rowCount := countRows(t, testDB, "sb_700_20_app_h1")
	if linVal, ok := fetchSampleLinZmmLin(t, testDB, "sb_700_20_app_h1"); ok {
		resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/sb700-20/app-h1/search/"+linVal)
		require.Equal(t, http.StatusOK, resp.Code)
		var payload standardResponse
		require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &payload))
		var data []model.Sb70020AppH1
		require.NoError(t, json.Unmarshal(payload.Data, &data))
		require.NotEmpty(t, data)
		require.Equal(t, linVal, data[0].LinZmmLin)
	}
	require.Equal(t, http.StatusNotFound, doJSONRequest(t, router, http.MethodGet, "/api/v1/sb700-20/app-h1/search/NOTREAL999").Code)

	resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/sb700-20/app-h1/list?page=1")
	if rowCount == 0 {
		require.Equal(t, http.StatusNotFound, resp.Code)
		return
	}
	require.Equal(t, http.StatusOK, resp.Code)
	var payload standardResponse
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &payload))
	var data sb700.PageResponse[model.Sb70020AppH1]
	require.NoError(t, json.Unmarshal(payload.Data, &data))
	require.NotEmpty(t, data.Items)
	require.Equal(t, http.StatusOK, doJSONRequest(t, router, http.MethodGet, "/api/v1/sb700-20/app-h1/list").Code)
	require.Equal(t, http.StatusNotFound, doJSONRequest(t, router, http.MethodGet, "/api/v1/sb700-20/app-h1/list?page=999999").Code)
}

func TestAppH2Endpoints(t *testing.T) {
	router := newTestRouter(t)
	if !hasRelation(t, testDB, "sb_700_20_app_h2") {
		t.Skip("sb_700_20_app_h2 missing in test DB")
	}
	rowCount := countRows(t, testDB, "sb_700_20_app_h2")
	if linVal, ok := fetchSampleLinZmmLin(t, testDB, "sb_700_20_app_h2"); ok {
		resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/sb700-20/app-h2/search/"+linVal)
		require.Equal(t, http.StatusOK, resp.Code)
		var payload standardResponse
		require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &payload))
		var data []model.Sb70020AppH2
		require.NoError(t, json.Unmarshal(payload.Data, &data))
		require.NotEmpty(t, data)
		require.Equal(t, linVal, data[0].LinZmmLin)
	}
	require.Equal(t, http.StatusNotFound, doJSONRequest(t, router, http.MethodGet, "/api/v1/sb700-20/app-h2/search/NOTREAL999").Code)

	resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/sb700-20/app-h2/list?page=1")
	if rowCount == 0 {
		require.Equal(t, http.StatusNotFound, resp.Code)
		return
	}
	require.Equal(t, http.StatusOK, resp.Code)
	var payload standardResponse
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &payload))
	var data sb700.PageResponse[model.Sb70020AppH2]
	require.NoError(t, json.Unmarshal(payload.Data, &data))
	require.NotEmpty(t, data.Items)
	require.Equal(t, http.StatusOK, doJSONRequest(t, router, http.MethodGet, "/api/v1/sb700-20/app-h2/list").Code)
	require.Equal(t, http.StatusNotFound, doJSONRequest(t, router, http.MethodGet, "/api/v1/sb700-20/app-h2/list?page=999999").Code)
}

func TestAppIEndpoints(t *testing.T) {
	router := newTestRouter(t)
	if !hasRelation(t, testDB, "sb_700_20_app_i") {
		t.Skip("sb_700_20_app_i missing in test DB")
	}
	rowCount := countRows(t, testDB, "sb_700_20_app_i")
	if linVal, ok := fetchSampleLIN(t, testDB, "sb_700_20_app_i"); ok {
		resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/sb700-20/app-i/search/"+linVal)
		require.Equal(t, http.StatusOK, resp.Code)
		var payload standardResponse
		require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &payload))
		var data model.Sb70020AppI
		require.NoError(t, json.Unmarshal(payload.Data, &data))
		require.Equal(t, linVal, data.Lin)
	}
	require.Equal(t, http.StatusNotFound, doJSONRequest(t, router, http.MethodGet, "/api/v1/sb700-20/app-i/search/NOTREAL999").Code)

	resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/sb700-20/app-i/list?page=1")
	if rowCount == 0 {
		require.Equal(t, http.StatusNotFound, resp.Code)
		return
	}
	require.Equal(t, http.StatusOK, resp.Code)
	var payload standardResponse
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &payload))
	var data sb700.PageResponse[model.Sb70020AppI]
	require.NoError(t, json.Unmarshal(payload.Data, &data))
	require.NotEmpty(t, data.Items)
	require.Equal(t, http.StatusOK, doJSONRequest(t, router, http.MethodGet, "/api/v1/sb700-20/app-i/list").Code)
	require.Equal(t, http.StatusNotFound, doJSONRequest(t, router, http.MethodGet, "/api/v1/sb700-20/app-i/list?page=999999").Code)
}

func TestAppJEndpoints(t *testing.T) {
	router := newTestRouter(t)
	if !hasRelation(t, testDB, "sb_700_20_app_j") {
		t.Skip("sb_700_20_app_j missing in test DB")
	}
	rowCount := countRows(t, testDB, "sb_700_20_app_j")
	if linVal, ok := fetchSampleLIN(t, testDB, "sb_700_20_app_j"); ok {
		resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/sb700-20/app-j/search/"+linVal)
		require.Equal(t, http.StatusOK, resp.Code)
		var payload standardResponse
		require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &payload))
		var data model.Sb70020AppJ
		require.NoError(t, json.Unmarshal(payload.Data, &data))
		require.Equal(t, linVal, data.Lin)
	}
	require.Equal(t, http.StatusNotFound, doJSONRequest(t, router, http.MethodGet, "/api/v1/sb700-20/app-j/search/NOTREAL999").Code)

	resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/sb700-20/app-j/list?page=1")
	if rowCount == 0 {
		require.Equal(t, http.StatusNotFound, resp.Code)
		return
	}
	require.Equal(t, http.StatusOK, resp.Code)
	var payload standardResponse
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &payload))
	var data sb700.PageResponse[model.Sb70020AppJ]
	require.NoError(t, json.Unmarshal(payload.Data, &data))
	require.NotEmpty(t, data.Items)
	require.Equal(t, http.StatusOK, doJSONRequest(t, router, http.MethodGet, "/api/v1/sb700-20/app-j/list").Code)
	require.Equal(t, http.StatusNotFound, doJSONRequest(t, router, http.MethodGet, "/api/v1/sb700-20/app-j/list?page=999999").Code)
}

func TestChp4Endpoints(t *testing.T) {
	router := newTestRouter(t)
	if !hasRelation(t, testDB, "sb_700_20_chp_4") {
		t.Skip("sb_700_20_chp_4 missing in test DB")
	}
	rowCount := countRows(t, testDB, "sb_700_20_chp_4")
	if linVal, ok := fetchSampleLIN(t, testDB, "sb_700_20_chp_4"); ok {
		resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/sb700-20/chp-4/search/"+linVal)
		require.Equal(t, http.StatusOK, resp.Code)
		var payload standardResponse
		require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &payload))
		var data model.Sb70020Chp4
		require.NoError(t, json.Unmarshal(payload.Data, &data))
		require.Equal(t, linVal, data.Lin)
	}
	require.Equal(t, http.StatusNotFound, doJSONRequest(t, router, http.MethodGet, "/api/v1/sb700-20/chp-4/search/NOTREAL999").Code)

	resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/sb700-20/chp-4/list?page=1")
	if rowCount == 0 {
		require.Equal(t, http.StatusNotFound, resp.Code)
		return
	}
	require.Equal(t, http.StatusOK, resp.Code)
	var payload standardResponse
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &payload))
	var data sb700.PageResponse[model.Sb70020Chp4]
	require.NoError(t, json.Unmarshal(payload.Data, &data))
	require.NotEmpty(t, data.Items)
	require.Equal(t, http.StatusOK, doJSONRequest(t, router, http.MethodGet, "/api/v1/sb700-20/chp-4/list").Code)
	require.Equal(t, http.StatusNotFound, doJSONRequest(t, router, http.MethodGet, "/api/v1/sb700-20/chp-4/list?page=999999").Code)
}

func TestChp6Endpoints(t *testing.T) {
	router := newTestRouter(t)
	if !hasRelation(t, testDB, "sb_700_20_chp_6") {
		t.Skip("sb_700_20_chp_6 missing in test DB")
	}
	rowCount := countRows(t, testDB, "sb_700_20_chp_6")
	if linVal, ok := fetchSampleLIN(t, testDB, "sb_700_20_chp_6"); ok {
		resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/sb700-20/chp-6/search/"+linVal)
		require.Equal(t, http.StatusOK, resp.Code)
		var payload standardResponse
		require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &payload))
		var data []model.Sb70020Chp6
		require.NoError(t, json.Unmarshal(payload.Data, &data))
		require.NotEmpty(t, data)
	}
	require.Equal(t, http.StatusNotFound, doJSONRequest(t, router, http.MethodGet, "/api/v1/sb700-20/chp-6/search/NOTREAL999").Code)

	resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/sb700-20/chp-6/list?page=1")
	if rowCount == 0 {
		require.Equal(t, http.StatusNotFound, resp.Code)
		return
	}
	require.Equal(t, http.StatusOK, resp.Code)
	var payload standardResponse
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &payload))
	var data sb700.PageResponse[model.Sb70020Chp6]
	require.NoError(t, json.Unmarshal(payload.Data, &data))
	require.NotEmpty(t, data.Items)
	require.Equal(t, http.StatusOK, doJSONRequest(t, router, http.MethodGet, "/api/v1/sb700-20/chp-6/list").Code)
	require.Equal(t, http.StatusNotFound, doJSONRequest(t, router, http.MethodGet, "/api/v1/sb700-20/chp-6/list?page=999999").Code)
}

func TestChp8Endpoints(t *testing.T) {
	router := newTestRouter(t)
	if !hasRelation(t, testDB, "sb_700_20_chp_8") {
		t.Skip("sb_700_20_chp_8 missing in test DB")
	}
	rowCount := countRows(t, testDB, "sb_700_20_chp_8")
	if linVal, ok := fetchSampleLIN(t, testDB, "sb_700_20_chp_8"); ok {
		resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/sb700-20/chp-8/search/"+linVal)
		require.Equal(t, http.StatusOK, resp.Code)
		var payload standardResponse
		require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &payload))
		var data []model.Sb70020Chp8
		require.NoError(t, json.Unmarshal(payload.Data, &data))
		require.NotEmpty(t, data)
	}
	require.Equal(t, http.StatusNotFound, doJSONRequest(t, router, http.MethodGet, "/api/v1/sb700-20/chp-8/search/NOTREAL999").Code)

	resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/sb700-20/chp-8/list?page=1")
	if rowCount == 0 {
		require.Equal(t, http.StatusNotFound, resp.Code)
		return
	}
	require.Equal(t, http.StatusOK, resp.Code)
	var payload standardResponse
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &payload))
	var data sb700.PageResponse[model.Sb70020Chp8]
	require.NoError(t, json.Unmarshal(payload.Data, &data))
	require.NotEmpty(t, data.Items)
	require.Equal(t, http.StatusOK, doJSONRequest(t, router, http.MethodGet, "/api/v1/sb700-20/chp-8/list").Code)
	require.Equal(t, http.StatusNotFound, doJSONRequest(t, router, http.MethodGet, "/api/v1/sb700-20/chp-8/list?page=999999").Code)
}

func TestInternalError(t *testing.T) {
	db, err := sql.Open("postgres", "postgres://invalid:invalid@localhost:1/invalid?sslmode=disable")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	router := newTestRouterWithDB(t, db)

	for _, ep := range []string{
		"/api/v1/sb700-20/app-b/list?page=1",
		"/api/v1/sb700-20/app-c/list?page=1",
		"/api/v1/sb700-20/chp-4/list?page=1",
	} {
		resp := doJSONRequest(t, router, http.MethodGet, ep)
		require.Equal(t, http.StatusInternalServerError, resp.Code, "endpoint: %s", ep)
		var payload response.ErrorResponse
		require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &payload))
		require.Equal(t, http.StatusInternalServerError, payload.Status)
	}
}

func TestAppEByNewLINEndpoint(t *testing.T) {
	router := newTestRouter(t)
	if !hasRelation(t, testDB, "sb_700_20_app_e") {
		t.Skip("sb_700_20_app_e missing in test DB")
	}
	if val, ok := fetchSampleNewLIN(t, testDB, "sb_700_20_app_e"); ok {
		resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/sb700-20/app-e/search-new-lin/"+val)
		require.Equal(t, http.StatusOK, resp.Code)
		var payload standardResponse
		require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &payload))
		var data []model.Sb70020AppE
		require.NoError(t, json.Unmarshal(payload.Data, &data))
		require.NotEmpty(t, data)
	}
	require.Equal(t, http.StatusNotFound, doJSONRequest(t, router, http.MethodGet, "/api/v1/sb700-20/app-e/search-new-lin/NOTREAL999").Code)
}

func TestAppGByNewLINEndpoint(t *testing.T) {
	router := newTestRouter(t)
	if !hasRelation(t, testDB, "sb_700_20_app_g") {
		t.Skip("sb_700_20_app_g missing in test DB")
	}
	if val, ok := fetchSampleNewLIN(t, testDB, "sb_700_20_app_g"); ok {
		resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/sb700-20/app-g/search-new-lin/"+val)
		require.Equal(t, http.StatusOK, resp.Code)
		var payload standardResponse
		require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &payload))
		var data []model.Sb70020AppG
		require.NoError(t, json.Unmarshal(payload.Data, &data))
		require.NotEmpty(t, data)
	}
	require.Equal(t, http.StatusNotFound, doJSONRequest(t, router, http.MethodGet, "/api/v1/sb700-20/app-g/search-new-lin/NOTREAL999").Code)
}

func TestAppH1BySubLINEndpoint(t *testing.T) {
	router := newTestRouter(t)
	if !hasRelation(t, testDB, "sb_700_20_app_h1") {
		t.Skip("sb_700_20_app_h1 missing in test DB")
	}
	if val, ok := fetchSampleSubLINAppH1(t, testDB); ok {
		resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/sb700-20/app-h1/search-sublin/"+val)
		require.Equal(t, http.StatusOK, resp.Code)
		var payload standardResponse
		require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &payload))
		var data []model.Sb70020AppH1
		require.NoError(t, json.Unmarshal(payload.Data, &data))
		require.NotEmpty(t, data)
	}
	require.Equal(t, http.StatusNotFound, doJSONRequest(t, router, http.MethodGet, "/api/v1/sb700-20/app-h1/search-sublin/NOTREAL999").Code)
}

func TestAppH2BySubLINEndpoint(t *testing.T) {
	router := newTestRouter(t)
	if !hasRelation(t, testDB, "sb_700_20_app_h2") {
		t.Skip("sb_700_20_app_h2 missing in test DB")
	}
	if val, ok := fetchSampleSubLINAppH2(t, testDB); ok {
		resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/sb700-20/app-h2/search-sublin/"+val)
		require.Equal(t, http.StatusOK, resp.Code)
		var payload standardResponse
		require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &payload))
		var data []model.Sb70020AppH2
		require.NoError(t, json.Unmarshal(payload.Data, &data))
		require.NotEmpty(t, data)
	}
	require.Equal(t, http.StatusNotFound, doJSONRequest(t, router, http.MethodGet, "/api/v1/sb700-20/app-h2/search-sublin/NOTREAL999").Code)
}

func TestChp4ByRICEndpoint(t *testing.T) {
	router := newTestRouter(t)
	if !hasRelation(t, testDB, "sb_700_20_chp_4") {
		t.Skip("sb_700_20_chp_4 missing in test DB")
	}
	if val, ok := fetchSampleRIC(t, testDB, "sb_700_20_chp_4"); ok {
		resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/sb700-20/chp-4/search-ric/"+val)
		require.Equal(t, http.StatusOK, resp.Code)
		var payload standardResponse
		require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &payload))
		var data []model.Sb70020Chp4
		require.NoError(t, json.Unmarshal(payload.Data, &data))
		require.NotEmpty(t, data)
	}
	require.Equal(t, http.StatusNotFound, doJSONRequest(t, router, http.MethodGet, "/api/v1/sb700-20/chp-4/search-ric/NOTREAL999").Code)
}

func TestChp6ByRICEndpoint(t *testing.T) {
	router := newTestRouter(t)
	if !hasRelation(t, testDB, "sb_700_20_chp_6") {
		t.Skip("sb_700_20_chp_6 missing in test DB")
	}
	if val, ok := fetchSampleRIC(t, testDB, "sb_700_20_chp_6"); ok {
		resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/sb700-20/chp-6/search-ric/"+val)
		require.Equal(t, http.StatusOK, resp.Code)
		var payload standardResponse
		require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &payload))
		var data []model.Sb70020Chp6
		require.NoError(t, json.Unmarshal(payload.Data, &data))
		require.NotEmpty(t, data)
	}
	require.Equal(t, http.StatusNotFound, doJSONRequest(t, router, http.MethodGet, "/api/v1/sb700-20/chp-6/search-ric/NOTREAL999").Code)
}

func TestChp8ByRICEndpoint(t *testing.T) {
	router := newTestRouter(t)
	if !hasRelation(t, testDB, "sb_700_20_chp_8") {
		t.Skip("sb_700_20_chp_8 missing in test DB")
	}
	if val, ok := fetchSampleRIC(t, testDB, "sb_700_20_chp_8"); ok {
		resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/sb700-20/chp-8/search-ric/"+val)
		require.Equal(t, http.StatusOK, resp.Code)
		var payload standardResponse
		require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &payload))
		var data []model.Sb70020Chp8
		require.NoError(t, json.Unmarshal(payload.Data, &data))
		require.NotEmpty(t, data)
	}
	require.Equal(t, http.StatusNotFound, doJSONRequest(t, router, http.MethodGet, "/api/v1/sb700-20/chp-8/search-ric/NOTREAL999").Code)
}
