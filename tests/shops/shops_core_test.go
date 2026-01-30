package shops_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestShopCreateGetUpdate(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "user-1")

	router := newTestRouter(t)

	createBody := map[string]interface{}{
		"name":    "Test Shop",
		"details": "Initial details",
	}

	createResp := doJSONRequest(t, router, http.MethodPost, "/api/v1/auth/shops", createBody, "user-1")
	require.Equal(t, http.StatusCreated, createResp.Code)

	created := decodeStandardResponse(t, createResp.Body)
	shopData := decodeMap(t, created.Data)
	shopID, ok := shopData["id"].(string)
	require.True(t, ok)
	require.NotEmpty(t, shopID)

	getResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops", nil, "user-1")
	require.Equal(t, http.StatusOK, getResp.Code)

	getList := decodeStandardResponse(t, getResp.Body)
	shopsList := decodeSlice(t, getList.Data)
	require.Len(t, shopsList, 1)

	updateBody := map[string]interface{}{
		"name":    "Updated Shop",
		"details": "Updated details",
	}

	updateResp := doJSONRequest(t, router, http.MethodPut, "/api/v1/auth/shops/"+shopID, updateBody, "user-1")
	require.Equal(t, http.StatusOK, updateResp.Code)

	getByIDResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/"+shopID, nil, "user-1")
	require.Equal(t, http.StatusOK, getByIDResp.Code)

	shopDetail := decodeStandardResponse(t, getByIDResp.Body)
	shopDetailData := decodeMap(t, shopDetail.Data)
	require.Equal(t, "Updated Shop", shopDetailData["name"])
}
