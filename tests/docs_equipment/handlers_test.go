package docs_equipment_test

import (
	"database/sql"
	"encoding/json"
	"math"
	"net/http"
	"testing"

	"miltechserver/api/docs_equipment"
	"miltechserver/api/response"

	"github.com/stretchr/testify/require"
)

func TestEquipmentDetailsBlankParams(t *testing.T) {
	router := newTestRouter(t)

	invalidPageResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/equipment-details?page=bad")
	require.Equal(t, http.StatusBadRequest, invalidPageResp.Code)

	zeroPageResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/equipment-details?page=0")
	require.Equal(t, http.StatusBadRequest, zeroPageResp.Code)

	emptySearchResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/equipment-details/search")
	require.Equal(t, http.StatusBadRequest, emptySearchResp.Code)
}

func TestEquipmentDetailsPaginated(t *testing.T) {
	router := newTestRouter(t)

	if !hasRelation(t, testDB, "docs_equipment_details") {
		t.Skip("docs_equipment_details table missing")
	}

	rowCount := countRows(t, testDB, "docs_equipment_details")
	resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/equipment-details?page=1")

	if rowCount == 0 {
		require.Equal(t, http.StatusNotFound, resp.Code)
		return
	}

	require.Equal(t, http.StatusOK, resp.Code)

	var payload standardResponse
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &payload))

	var data docs_equipment.EquipmentDetailsPageResponse
	require.NoError(t, json.Unmarshal(payload.Data, &data))
	require.NotEmpty(t, data.Items)
	require.Equal(t, 1, data.Page)

	expectedTotalPages := int(math.Ceil(float64(rowCount) / 40.0))
	require.Equal(t, expectedTotalPages, data.TotalPages)
	require.Equal(t, 1 >= expectedTotalPages, data.IsLastPage)
}

func TestEquipmentFamilies(t *testing.T) {
	router := newTestRouter(t)

	if !hasRelation(t, testDB, "docs_equipment_details") {
		t.Skip("docs_equipment_details table missing")
	}

	resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/equipment-details/families")
	require.Equal(t, http.StatusOK, resp.Code)

	var payload standardResponse
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &payload))

	var data docs_equipment.FamiliesResponse
	require.NoError(t, json.Unmarshal(payload.Data, &data))

	if countRows(t, testDB, "docs_equipment_details") > 0 {
		require.NotEmpty(t, data.Families)
		require.Greater(t, data.Count, 0)
	}
}

func TestEquipmentByFamily(t *testing.T) {
	router := newTestRouter(t)

	if !hasRelation(t, testDB, "docs_equipment_details") {
		t.Skip("docs_equipment_details table missing")
	}

	family, ok := fetchSampleFamily(t, testDB)
	if !ok {
		t.Skip("no family data found")
	}

	resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/equipment-details/family/"+family+"?page=1")
	require.Equal(t, http.StatusOK, resp.Code)

	var payload standardResponse
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &payload))

	var data docs_equipment.EquipmentDetailsPageResponse
	require.NoError(t, json.Unmarshal(payload.Data, &data))
	require.NotEmpty(t, data.Items)
}

func TestEquipmentSearch(t *testing.T) {
	router := newTestRouter(t)

	if !hasRelation(t, testDB, "docs_equipment_details") {
		t.Skip("docs_equipment_details table missing")
	}

	modelValue, ok := fetchSampleModel(t, testDB)
	if !ok {
		t.Skip("no model data found")
	}

	resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/equipment-details/search?q="+modelValue+"&page=1")
	require.Equal(t, http.StatusOK, resp.Code)

	var payload standardResponse
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &payload))

	var data docs_equipment.EquipmentDetailsPageResponse
	require.NoError(t, json.Unmarshal(payload.Data, &data))
	require.NotEmpty(t, data.Items)
}

func TestEquipmentInternalError(t *testing.T) {
	db, err := sql.Open("postgres", "postgres://invalid:invalid@localhost:1/invalid?sslmode=disable")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	router := newTestRouterWithDB(t, db)
	resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/equipment-details?page=1")
	require.Equal(t, http.StatusInternalServerError, resp.Code)

	var payload response.ErrorResponse
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &payload))
	require.Equal(t, http.StatusInternalServerError, payload.Status)
}
