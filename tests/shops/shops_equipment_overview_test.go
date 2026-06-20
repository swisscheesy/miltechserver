package shops_test

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	shopcore "miltechserver/api/shops/core"
	"miltechserver/bootstrap"

	"github.com/stretchr/testify/require"
)

func TestShopEquipmentOverviewRepository(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "overview-user")
	ensureUser(t, testDB, "other-user")
	router := newTestRouter(t)

	newerShopID := createShop(t, router, "overview-user", "Newer Shop")
	olderShopID := createShop(t, router, "overview-user", "Older Shop")
	hiddenShopID := createShop(t, router, "other-user", "Hidden Shop")
	newerEquipmentID := createVehicle(t, router, "overview-user", newerShopID)
	_, err := testDB.Exec(`UPDATE shop_vehicle SET admin=$1, serial=$2 WHERE id=$3`,
		"TEMP", "SER-TEMP", newerEquipmentID)
	require.NoError(t, err)
	olderEquipmentID := createVehicle(t, router, "overview-user", newerShopID)
	_ = createVehicle(t, router, "other-user", hiddenShopID)

	newerTime := time.Date(2026, time.June, 20, 12, 0, 0, 0, time.UTC)
	olderTime := newerTime.Add(-time.Hour)
	_, err = testDB.Exec(`UPDATE shops SET created_at = $1 WHERE id = $2`, newerTime, newerShopID)
	require.NoError(t, err)
	_, err = testDB.Exec(`UPDATE shops SET created_at = $1 WHERE id = $2`, olderTime, olderShopID)
	require.NoError(t, err)
	_, err = testDB.Exec(`UPDATE shop_vehicle SET admin=$1, model=$2, serial=$3, niin=$4, save_time=$5 WHERE id=$6`,
		"NEW", "M1097", "SER-NEW", "000000001", newerTime, newerEquipmentID)
	require.NoError(t, err)
	_, err = testDB.Exec(`UPDATE shop_vehicle SET admin=$1, model=$2, serial=$3, niin=$4, save_time=$5 WHERE id=$6`,
		"OLD", "M998", "SER-OLD", "000000002", olderTime, olderEquipmentID)
	require.NoError(t, err)

	repository := shopcore.NewRepository(testDB, nil, &bootstrap.Env{})
	shops, err := repository.GetShopEquipmentOverview(context.Background(), &bootstrap.User{UserID: "overview-user"})
	require.NoError(t, err)
	require.Len(t, shops, 2)
	require.Equal(t, newerShopID, shops[0].ID)
	require.Equal(t, olderShopID, shops[1].ID)
	require.Len(t, shops[0].Equipment, 2)
	require.Equal(t, newerEquipmentID, shops[0].Equipment[0].ID)
	require.Equal(t, olderEquipmentID, shops[0].Equipment[1].ID)
	require.Equal(t, "NEW", shops[0].Equipment[0].Admin)
	require.Equal(t, "M1097", shops[0].Equipment[0].Model)
	require.Equal(t, "SER-NEW", shops[0].Equipment[0].Serial)
	require.Equal(t, "000000001", shops[0].Equipment[0].Niin)
	require.Empty(t, shops[1].Equipment)
	for _, shop := range shops {
		require.NotEqual(t, hiddenShopID, shop.ID)
	}
}

func TestShopEquipmentOverviewRepositoryHonorsCanceledContext(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "overview-user")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	repository := shopcore.NewRepository(testDB, nil, &bootstrap.Env{})
	_, err := repository.GetShopEquipmentOverview(ctx, &bootstrap.User{UserID: "overview-user"})
	require.Error(t, err)
	require.True(t, errors.Is(err, context.Canceled), err.Error())
}

func TestShopEquipmentOverviewEndpoint(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "overview-user")
	ensureUser(t, testDB, "other-user")
	router := newTestRouter(t)

	visibleShopID := createShop(t, router, "overview-user", "Visible Shop")
	emptyShopID := createShop(t, router, "overview-user", "Empty Shop")
	hiddenShopID := createShop(t, router, "other-user", "Hidden Shop")
	visibleEquipmentID := createVehicle(t, router, "overview-user", visibleShopID)
	_ = createVehicle(t, router, "other-user", hiddenShopID)
	_, err := testDB.Exec(`UPDATE shop_vehicle SET admin='A123', model='M1097', serial='SER-1', niin='012345678' WHERE id=$1`, visibleEquipmentID)
	require.NoError(t, err)

	resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/equipment/overview", nil, "overview-user")
	require.Equal(t, http.StatusOK, resp.Code)
	payload := decodeStandardResponse(t, resp.Body)
	data := decodeMap(t, payload.Data)
	shops := data["shops"].([]interface{})
	require.Len(t, shops, 2)

	shopsByID := make(map[string]map[string]interface{}, len(shops))
	for _, rawShop := range shops {
		shop := rawShop.(map[string]interface{})
		shopsByID[shop["id"].(string)] = shop
	}
	require.NotContains(t, shopsByID, hiddenShopID)
	require.Contains(t, shopsByID, visibleShopID)
	require.Contains(t, shopsByID, emptyShopID)
	require.Equal(t, float64(1), shopsByID[visibleShopID]["equipment_count"])
	require.Equal(t, float64(0), shopsByID[emptyShopID]["equipment_count"])
	require.Empty(t, shopsByID[emptyShopID]["equipment"].([]interface{}))

	equipment := shopsByID[visibleShopID]["equipment"].([]interface{})[0].(map[string]interface{})
	require.Equal(t, map[string]interface{}{
		"id": visibleEquipmentID, "admin": "A123", "model": "M1097",
		"serial": "SER-1", "niin": "012345678",
	}, equipment)
}

func TestShopEquipmentOverviewEndpointRequiresAuthentication(t *testing.T) {
	router := newTestRouter(t)
	resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/equipment/overview", nil, "")
	require.Equal(t, http.StatusUnauthorized, resp.Code)
}

func TestShopEquipmentOverviewEndpointGzipIsScoped(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "overview-user")
	router := newTestRouter(t)
	_ = createShop(t, router, "overview-user", "Compressed Shop")

	req, err := http.NewRequest(http.MethodGet, "/api/v1/auth/shops/equipment/overview", nil)
	require.NoError(t, err)
	req.Header.Set("X-User-ID", "overview-user")
	req.Header.Set("Accept-Encoding", "gzip")
	compressed := httptest.NewRecorder()
	router.ServeHTTP(compressed, req)
	require.Equal(t, http.StatusOK, compressed.Code)
	require.Equal(t, "gzip", compressed.Header().Get("Content-Encoding"))

	reader, err := gzip.NewReader(bytes.NewReader(compressed.Body.Bytes()))
	require.NoError(t, err)
	decompressed, err := io.ReadAll(reader)
	require.NoError(t, err)
	require.NoError(t, reader.Close())
	var decoded standardResponse
	require.NoError(t, json.Unmarshal(decompressed, &decoded))
	normalReq, err := http.NewRequest(http.MethodGet, "/api/v1/auth/shops/equipment/overview", nil)
	require.NoError(t, err)
	normalReq.Header.Set("X-User-ID", "overview-user")
	normal := httptest.NewRecorder()
	router.ServeHTTP(normal, normalReq)
	require.Empty(t, normal.Header().Get("Content-Encoding"))
	require.JSONEq(t, normal.Body.String(), string(decompressed))

	shopsReq, err := http.NewRequest(http.MethodGet, "/api/v1/auth/shops", nil)
	require.NoError(t, err)
	shopsReq.Header.Set("X-User-ID", "overview-user")
	shopsReq.Header.Set("Accept-Encoding", "gzip")
	uncompressed := httptest.NewRecorder()
	router.ServeHTTP(uncompressed, shopsReq)
	require.Empty(t, uncompressed.Header().Get("Content-Encoding"))
}
