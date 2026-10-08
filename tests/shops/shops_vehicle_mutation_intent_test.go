package shops_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func intentVehicle(t *testing.T) vehicleUsageFixture {
	t.Helper()
	f := newVehicleUsageFixture(t, 100, 50)
	_, err := testDB.Exec(`UPDATE shop_vehicle SET niin='NIIN',model='MODEL',serial='SERIAL',uoc='UOC',comment='COMMENT' WHERE id=$1`, f.vehicleID)
	require.NoError(t, err)
	return f
}

func fetchedIntentVehicle(t *testing.T, f vehicleUsageFixture) map[string]interface{} {
	t.Helper()
	r := doJSONRequest(t, f.router, http.MethodGet, "/api/v1/auth/shops/vehicles/"+f.vehicleID, nil, f.memberID)
	require.Equal(t, http.StatusOK, r.Code)
	return decodeMap(t, decodeStandardResponse(t, r.Body).Data)
}

func TestUsageOnlyPreservesMetadataForEveryRole(t *testing.T) {
	for _, role := range []string{"creator", "admin", "member"} {
		t.Run(role, func(t *testing.T) {
			f := intentVehicle(t)
			user := "member"
			if role == "creator" {
				user = "owner"
				_, err := testDB.Exec(`UPDATE shop_members SET role='member' WHERE user_id='owner'`)
				require.NoError(t, err)
			}
			if role == "admin" {
				_, err := testDB.Exec(`UPDATE shop_members SET role='admin' WHERE user_id='member'`)
				require.NoError(t, err)
			}
			before := fetchedIntentVehicle(t, f)
			r := doJSONRequest(t, f.router, http.MethodPut, "/api/v1/auth/shops/vehicles", map[string]interface{}{
				"vehicle_id": f.vehicleID, "admin": "STALE-SNAPSHOT", "mileage": -100, "hours": -50,
				"niin": "", "model": nil, "serial": "", "uoc": "", "comment": nil, "tracked_mileage": 125, "tracked_hours": nil,
			}, user)
			require.Equal(t, http.StatusOK, r.Code)
			after := fetchedIntentVehicle(t, f)
			for _, field := range []string{"admin", "niin", "model", "serial", "uoc", "comment", "mileage", "hours"} {
				require.Equal(t, before[field], after[field], field)
			}
			require.Equal(t, float64(125), after["tracked_mileage"])
			require.Nil(t, after["tracked_hours"])
		})
	}
}

func TestAdminEditPersists(t *testing.T) {
	f := intentVehicle(t)
	r := doJSONRequest(t, f.router, http.MethodPut, "/api/v1/auth/shops/vehicles", map[string]interface{}{"vehicle_id": f.vehicleID, "admin": "NEW-ADMIN"}, "owner")
	require.Equal(t, http.StatusOK, r.Code)
	got := fetchedIntentVehicle(t, f)
	require.Equal(t, "NEW-ADMIN", got["admin"])
	require.Equal(t, "MODEL", got["model"])
	require.Equal(t, float64(100), got["mileage"])
}

func TestBaseUsageNonnegative(t *testing.T) {
	for _, field := range []string{"mileage", "hours"} {
		t.Run("metadata-"+field, func(t *testing.T) {
			f := intentVehicle(t)
			r := doJSONRequest(t, f.router, http.MethodPut, "/api/v1/auth/shops/vehicles", map[string]interface{}{"vehicle_id": f.vehicleID, "admin": "ADMIN-001", field: -1}, "owner")
			require.Equal(t, http.StatusBadRequest, r.Code)
			got := fetchedIntentVehicle(t, f)
			require.Equal(t, float64(100), got["mileage"])
			require.Equal(t, float64(50), got["hours"])
		})
		t.Run("create-"+field, func(t *testing.T) {
			f := intentVehicle(t)
			var shop string
			require.NoError(t, testDB.QueryRow(`SELECT shop_id FROM shop_vehicle WHERE id=$1`, f.vehicleID).Scan(&shop))
			r := doJSONRequest(t, f.router, http.MethodPost, "/api/v1/auth/shops/vehicles", map[string]interface{}{"shop_id": shop, "admin": "NEGATIVE", field: -1}, "owner")
			require.Equal(t, http.StatusBadRequest, r.Code)
			var count int
			require.NoError(t, testDB.QueryRow(`SELECT count(*) FROM shop_vehicle WHERE shop_id=$1`, shop).Scan(&count))
			require.Equal(t, 1, count)
		})
		t.Run("database-"+field, func(t *testing.T) {
			f := intentVehicle(t)
			tx, err := testDB.Begin()
			require.NoError(t, err)
			defer tx.Rollback()
			_, err = tx.Exec(`UPDATE shop_vehicle SET `+field+`=-1 WHERE id=$1`, f.vehicleID)
			require.Error(t, err)
		})
	}
}

func TestVehicleMetadataPresenceAndTrackedCompatibility(t *testing.T) {
	f := intentVehicle(t)
	seedTrackedUsage(t, testDB, f.vehicleID, 120, 60)
	r := doJSONRequest(t, f.router, http.MethodPut, "/api/v1/auth/shops/vehicles", map[string]interface{}{
		"vehicle_id": f.vehicleID, "model": "NEW-MODEL", "comment": "", "serial": nil, "uoc": "", "mileage": 0, "tracked_mileage": 130, "tracked_hours": nil,
	}, "owner")
	require.Equal(t, http.StatusOK, r.Code)
	got := fetchedIntentVehicle(t, f)
	require.Equal(t, "ADMIN-001", got["admin"])
	require.Equal(t, "NIIN", got["niin"])
	require.Equal(t, "NEW-MODEL", got["model"])
	require.Equal(t, "SERIAL", got["serial"])
	require.Equal(t, "", got["comment"])
	require.Equal(t, "UNK", got["uoc"])
	require.Equal(t, float64(0), got["mileage"])
	require.Equal(t, float64(50), got["hours"])
	require.Equal(t, float64(130), got["tracked_mileage"])
	require.Equal(t, float64(60), got["tracked_hours"])
	r = doJSONRequest(t, f.router, http.MethodPut, "/api/v1/auth/shops/vehicles", map[string]interface{}{"vehicle_id": f.vehicleID, "admin": "ADMIN-ONLY", "mileage": nil, "hours": nil}, "owner")
	require.Equal(t, http.StatusOK, r.Code)
	got = fetchedIntentVehicle(t, f)
	require.Equal(t, "ADMIN-ONLY", got["admin"])
	require.Equal(t, float64(130), got["tracked_mileage"])
	require.Equal(t, float64(60), got["tracked_hours"])
	for _, body := range []map[string]interface{}{
		{"vehicle_id": f.vehicleID, "admin": "UNAUTHORIZED"},
		{"vehicle_id": f.vehicleID, "model": "UNAUTHORIZED", "tracked_mileage": 200},
	} {
		before := fetchedIntentVehicle(t, f)
		denied := doJSONRequest(t, f.router, http.MethodPut, "/api/v1/auth/shops/vehicles", body, "member")
		require.GreaterOrEqual(t, denied.Code, 400)
		require.Equal(t, before, fetchedIntentVehicle(t, f))
	}
	emptyAdmin := doJSONRequest(t, f.router, http.MethodPut, "/api/v1/auth/shops/vehicles", map[string]interface{}{"vehicle_id": f.vehicleID, "admin": ""}, "owner")
	require.Equal(t, http.StatusBadRequest, emptyAdmin.Code)
}

func TestUsageOnlyIgnoresEmptyAdminAndNullTrackedPreserves(t *testing.T) {
	f := intentVehicle(t)
	seedTrackedUsage(t, testDB, f.vehicleID, 120, 60)
	before := fetchedIntentVehicle(t, f)
	r := doJSONRequest(t, f.router, http.MethodPut, "/api/v1/auth/shops/vehicles", map[string]interface{}{"vehicle_id": f.vehicleID, "admin": "", "tracked_mileage": 125, "tracked_hours": nil}, "owner")
	require.Equal(t, http.StatusOK, r.Code)
	got := fetchedIntentVehicle(t, f)
	require.Equal(t, before["admin"], got["admin"])
	require.Equal(t, float64(125), got["tracked_mileage"])
	require.Equal(t, float64(60), got["tracked_hours"])
}

func TestVehicleMetadataExplicitClearsAndAdminAuthority(t *testing.T) {
	f := intentVehicle(t)
	_, err := testDB.Exec(`UPDATE shop_members SET role='admin' WHERE user_id='member'`)
	require.NoError(t, err)
	r := doJSONRequest(t, f.router, http.MethodPut, "/api/v1/auth/shops/vehicles", map[string]interface{}{
		"vehicle_id": f.vehicleID, "admin": "ADMIN-EDIT", "niin": "", "model": "", "serial": "", "comment": "", "uoc": nil, "tracked_mileage": nil, "tracked_hours": nil,
	}, "member")
	require.Equal(t, http.StatusOK, r.Code)
	got := fetchedIntentVehicle(t, f)
	require.Equal(t, "ADMIN-EDIT", got["admin"])
	for _, field := range []string{"niin", "model", "serial", "comment"} {
		require.Equal(t, "", got[field], field)
	}
	require.Equal(t, "UOC", got["uoc"])
	require.Nil(t, got["tracked_mileage"])
	require.Nil(t, got["tracked_hours"])
}
