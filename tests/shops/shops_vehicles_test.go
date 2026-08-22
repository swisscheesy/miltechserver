package shops_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type vehicleUsageFixture struct {
	router    *gin.Engine
	vehicleID string
	memberID  string
}

func TestAdjustVehicleUsageAddsAndReturnsPersistedVehicle(t *testing.T) {
	fixture := newVehicleUsageFixture(t, 100, 50)

	resp := doJSONRequest(
		t,
		fixture.router,
		http.MethodPatch,
		"/api/v1/auth/shops/vehicles/"+fixture.vehicleID+"/usage",
		map[string]interface{}{
			"operation":          "add",
			"mileage_adjustment": 10,
			"hours_adjustment":   5,
		},
		fixture.memberID,
	)

	require.Equal(t, http.StatusOK, resp.Code)
	standard := decodeStandardResponse(t, resp.Body)
	require.Equal(t, http.StatusOK, standard.Status)
	require.Equal(t, "Equipment usage adjusted successfully", standard.Message)
	vehicle := decodeMap(t, standard.Data)
	require.Equal(t, fixture.vehicleID, vehicle["id"])
	require.Equal(t, float64(110), vehicle["tracked_mileage"])
	require.Equal(t, float64(55), vehicle["tracked_hours"])
}

func TestAdjustVehicleUsageSubtractsForOrdinaryMember(t *testing.T) {
	fixture := newVehicleUsageFixture(t, 100, 50)
	seedTrackedUsage(t, testDB, fixture.vehicleID, 120, 70)

	resp := doJSONRequest(
		t,
		fixture.router,
		http.MethodPatch,
		"/api/v1/auth/shops/vehicles/"+fixture.vehicleID+"/usage",
		map[string]interface{}{
			"operation":          "subtract",
			"mileage_adjustment": 20,
			"hours_adjustment":   15,
		},
		fixture.memberID,
	)

	require.Equal(t, http.StatusOK, resp.Code)
	vehicle := decodeMap(t, decodeStandardResponse(t, resp.Body).Data)
	require.Equal(t, float64(100), vehicle["tracked_mileage"])
	require.Equal(t, float64(55), vehicle["tracked_hours"])
}

func TestAdjustVehicleUsageValidation(t *testing.T) {
	cases := []struct {
		name        string
		body        map[string]interface{}
		userID      string
		wantStatus  int
		wantMileage float64
		wantHours   float64
		assertUsage bool
	}{
		{name: "missing operation defaults add", body: map[string]interface{}{"mileage_adjustment": 1}, userID: "member", wantStatus: http.StatusOK, wantMileage: 11, wantHours: 10, assertUsage: true},
		{name: "blank operation defaults add", body: map[string]interface{}{"operation": "  ", "hours_adjustment": 1}, userID: "member", wantStatus: http.StatusOK, wantMileage: 10, wantHours: 11, assertUsage: true},
		{name: "hours only", body: map[string]interface{}{"operation": "subtract", "hours_adjustment": 1}, userID: "member", wantStatus: http.StatusOK, wantMileage: 10, wantHours: 9, assertUsage: true},
		{name: "both omitted", body: map[string]interface{}{}, userID: "member", wantStatus: http.StatusBadRequest},
		{name: "both zero", body: map[string]interface{}{"mileage_adjustment": 0, "hours_adjustment": 0}, userID: "member", wantStatus: http.StatusBadRequest},
		{name: "negative magnitude", body: map[string]interface{}{"mileage_adjustment": -1}, userID: "member", wantStatus: http.StatusBadRequest},
		{name: "magnitude too large", body: map[string]interface{}{"hours_adjustment": 10001}, userID: "member", wantStatus: http.StatusBadRequest},
		{name: "unknown operation", body: map[string]interface{}{"operation": "negative", "hours_adjustment": 1}, userID: "member", wantStatus: http.StatusBadRequest},
		{name: "outsider", body: map[string]interface{}{"mileage_adjustment": 1}, userID: "outsider", wantStatus: http.StatusForbidden},
		{name: "missing user", body: map[string]interface{}{"mileage_adjustment": 1}, userID: "", wantStatus: http.StatusUnauthorized},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			fixture := newVehicleUsageFixture(t, 10, 10)
			if testCase.assertUsage {
				seedTrackedUsage(t, testDB, fixture.vehicleID, 10, 10)
			}

			resp := doJSONRequest(
				t,
				fixture.router,
				http.MethodPatch,
				"/api/v1/auth/shops/vehicles/"+fixture.vehicleID+"/usage",
				testCase.body,
				testCase.userID,
			)

			require.Equal(t, testCase.wantStatus, resp.Code)
			if testCase.assertUsage {
				vehicle := decodeMap(t, decodeStandardResponse(t, resp.Body).Data)
				require.Equal(t, testCase.wantMileage, vehicle["tracked_mileage"])
				require.Equal(t, testCase.wantHours, vehicle["tracked_hours"])
			}
		})
	}

	t.Run("nonexistent vehicle", func(t *testing.T) {
		fixture := newVehicleUsageFixture(t, 10, 10)
		resp := doJSONRequest(
			t,
			fixture.router,
			http.MethodPatch,
			"/api/v1/auth/shops/vehicles/00000000-0000-4000-8000-000000000000/usage",
			map[string]interface{}{"mileage_adjustment": 1},
			fixture.memberID,
		)
		require.Equal(t, http.StatusNotFound, resp.Code)
	})
}

func TestAdjustVehicleUsageRejectsUnknownFields(t *testing.T) {
	fixture := newVehicleUsageFixture(t, 10, 10)

	req, err := http.NewRequest(
		http.MethodPatch,
		"/api/v1/auth/shops/vehicles/"+fixture.vehicleID+"/usage",
		bytes.NewBufferString(`{"mileage_adjustment":1,"comment":"not allowed"}`),
	)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", fixture.memberID)

	resp := httptest.NewRecorder()
	fixture.router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusBadRequest, resp.Code)
}

func TestAdjustVehicleUsageRejectsMalformedJSON(t *testing.T) {
	fixture := newVehicleUsageFixture(t, 10, 10)

	req, err := http.NewRequest(
		http.MethodPatch,
		"/api/v1/auth/shops/vehicles/"+fixture.vehicleID+"/usage",
		bytes.NewBufferString(`{"mileage_adjustment":`),
	)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", fixture.memberID)

	resp := httptest.NewRecorder()
	fixture.router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusBadRequest, resp.Code)
}

func TestAdjustVehicleUsageRejectsTrailingJSON(t *testing.T) {
	fixture := newVehicleUsageFixture(t, 10, 10)

	req, err := http.NewRequest(
		http.MethodPatch,
		"/api/v1/auth/shops/vehicles/"+fixture.vehicleID+"/usage",
		bytes.NewBufferString(`{"mileage_adjustment":1}{"hours_adjustment":1}`),
	)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", fixture.memberID)

	resp := httptest.NewRecorder()
	fixture.router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusBadRequest, resp.Code)
}

func TestAdjustVehicleUsageAllowsExactZero(t *testing.T) {
	fixture := newVehicleUsageFixture(t, 10, 10)

	resp := doJSONRequest(
		t,
		fixture.router,
		http.MethodPatch,
		"/api/v1/auth/shops/vehicles/"+fixture.vehicleID+"/usage",
		map[string]interface{}{
			"operation":          "subtract",
			"mileage_adjustment": 10,
			"hours_adjustment":   10,
		},
		fixture.memberID,
	)
	require.Equal(t, http.StatusOK, resp.Code)
	vehicle := decodeMap(t, decodeStandardResponse(t, resp.Body).Data)
	require.Equal(t, float64(0), vehicle["tracked_mileage"])
	require.Equal(t, float64(0), vehicle["tracked_hours"])
}

func TestAdjustVehicleUsageRejectsBelowZeroAtomically(t *testing.T) {
	fixture := newVehicleUsageFixture(t, 10, 10)

	exactZeroResp := doJSONRequest(
		t,
		fixture.router,
		http.MethodPatch,
		"/api/v1/auth/shops/vehicles/"+fixture.vehicleID+"/usage",
		map[string]interface{}{
			"operation":          "subtract",
			"mileage_adjustment": 10,
			"hours_adjustment":   10,
		},
		fixture.memberID,
	)
	require.Equal(t, http.StatusOK, exactZeroResp.Code)

	belowZeroResp := doJSONRequest(
		t,
		fixture.router,
		http.MethodPatch,
		"/api/v1/auth/shops/vehicles/"+fixture.vehicleID+"/usage",
		map[string]interface{}{
			"operation":          "subtract",
			"mileage_adjustment": 1,
		},
		fixture.memberID,
	)
	require.Equal(t, http.StatusConflict, belowZeroResp.Code)

	getResp := doJSONRequest(
		t,
		fixture.router,
		http.MethodGet,
		"/api/v1/auth/shops/vehicles/"+fixture.vehicleID,
		nil,
		fixture.memberID,
	)
	require.Equal(t, http.StatusOK, getResp.Code)
	vehicle := decodeMap(t, decodeStandardResponse(t, getResp.Body).Data)
	require.Equal(t, float64(0), vehicle["tracked_mileage"])
	require.Equal(t, float64(0), vehicle["tracked_hours"])
}

func TestAdjustVehicleUsageRejectsInt32OverflowAtomically(t *testing.T) {
	fixture := newVehicleUsageFixture(t, 10, 10)
	seedTrackedUsage(t, testDB, fixture.vehicleID, math.MaxInt32, 10)

	overflowResp := doJSONRequest(
		t,
		fixture.router,
		http.MethodPatch,
		"/api/v1/auth/shops/vehicles/"+fixture.vehicleID+"/usage",
		map[string]interface{}{"mileage_adjustment": 1},
		fixture.memberID,
	)
	require.Equal(t, http.StatusConflict, overflowResp.Code)

	getResp := doJSONRequest(
		t,
		fixture.router,
		http.MethodGet,
		"/api/v1/auth/shops/vehicles/"+fixture.vehicleID,
		nil,
		fixture.memberID,
	)
	require.Equal(t, http.StatusOK, getResp.Code)
	vehicle := decodeMap(t, decodeStandardResponse(t, getResp.Body).Data)
	require.Equal(t, float64(math.MaxInt32), vehicle["tracked_mileage"])
	require.Equal(t, float64(10), vehicle["tracked_hours"])
}

func TestAdjustVehicleUsageConcurrentAddsAreCumulative(t *testing.T) {
	fixture := newVehicleUsageFixture(t, 100, 50)
	seedTrackedUsage(t, testDB, fixture.vehicleID, 100, 50)

	const adjustmentCount = 10
	statuses := make(chan int, adjustmentCount)
	requestErrors := make(chan error, adjustmentCount)
	var waitGroup sync.WaitGroup

	for index := 0; index < adjustmentCount; index++ {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()

			payload, err := json.Marshal(map[string]interface{}{
				"operation":          "add",
				"mileage_adjustment": 1,
			})
			if err != nil {
				requestErrors <- err
				return
			}

			req, err := http.NewRequest(
				http.MethodPatch,
				"/api/v1/auth/shops/vehicles/"+fixture.vehicleID+"/usage",
				bytes.NewReader(payload),
			)
			if err != nil {
				requestErrors <- err
				return
			}
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-User-ID", fixture.memberID)

			resp := httptest.NewRecorder()
			fixture.router.ServeHTTP(resp, req)
			statuses <- resp.Code
		}()
	}

	waitGroup.Wait()
	close(statuses)
	close(requestErrors)
	require.Empty(t, requestErrors)
	for status := range statuses {
		require.Equal(t, http.StatusOK, status)
	}

	getResp := doJSONRequest(
		t,
		fixture.router,
		http.MethodGet,
		"/api/v1/auth/shops/vehicles/"+fixture.vehicleID,
		nil,
		fixture.memberID,
	)
	require.Equal(t, http.StatusOK, getResp.Code)
	vehicle := decodeMap(t, decodeStandardResponse(t, getResp.Body).Data)
	require.Equal(t, float64(110), vehicle["tracked_mileage"])
}

func TestLegacyVehicleUpdateRejectsNegativeTrackedUsage(t *testing.T) {
	fixture := newVehicleUsageFixture(t, 100, 50)

	validUpdate := map[string]interface{}{
		"vehicle_id":      fixture.vehicleID,
		"admin":           "ADMIN-001",
		"mileage":         100,
		"hours":           50,
		"tracked_mileage": 25,
		"tracked_hours":   15,
	}
	validResp := doJSONRequest(
		t,
		fixture.router,
		http.MethodPut,
		"/api/v1/auth/shops/vehicles",
		validUpdate,
		"owner",
	)
	require.Equal(t, http.StatusOK, validResp.Code)

	negativeUpdate := map[string]interface{}{
		"vehicle_id":      fixture.vehicleID,
		"admin":           "ADMIN-001",
		"mileage":         100,
		"hours":           50,
		"tracked_mileage": -1,
		"tracked_hours":   15,
	}
	negativeResp := doJSONRequest(
		t,
		fixture.router,
		http.MethodPut,
		"/api/v1/auth/shops/vehicles",
		negativeUpdate,
		"owner",
	)
	require.Equal(t, http.StatusBadRequest, negativeResp.Code)

	getResp := doJSONRequest(
		t,
		fixture.router,
		http.MethodGet,
		"/api/v1/auth/shops/vehicles/"+fixture.vehicleID,
		nil,
		fixture.memberID,
	)
	require.Equal(t, http.StatusOK, getResp.Code)
	vehicle := decodeMap(t, decodeStandardResponse(t, getResp.Body).Data)
	require.Equal(t, float64(25), vehicle["tracked_mileage"])
	require.Equal(t, float64(15), vehicle["tracked_hours"])
}

func newVehicleUsageFixture(t *testing.T, mileage, hours int32) vehicleUsageFixture {
	t.Helper()

	clearShopTables(t, testDB)
	ensureUser(t, testDB, "owner")
	ensureUser(t, testDB, "member")
	ensureUser(t, testDB, "outsider")

	router := newTestRouter(t)
	shopID := createShop(t, router, "owner", "Usage Adjustment Shop")
	createResp := doJSONRequest(
		t,
		router,
		http.MethodPost,
		"/api/v1/auth/shops/vehicles",
		map[string]interface{}{
			"shop_id": shopID,
			"admin":   "ADMIN-001",
			"mileage": mileage,
			"hours":   hours,
		},
		"owner",
	)
	require.Equal(t, http.StatusCreated, createResp.Code)
	vehicle := decodeMap(t, decodeStandardResponse(t, createResp.Body).Data)
	vehicleID, ok := vehicle["id"].(string)
	require.True(t, ok)
	require.NotEmpty(t, vehicleID)

	_, inviteCode := createInviteCode(t, router, "owner", shopID)
	joinResp := doJSONRequest(
		t,
		router,
		http.MethodPost,
		"/api/v1/auth/shops/join",
		map[string]interface{}{"invite_code": inviteCode},
		"member",
	)
	require.Equal(t, http.StatusOK, joinResp.Code)

	return vehicleUsageFixture{
		router:    router,
		vehicleID: vehicleID,
		memberID:  "member",
	}
}

func seedTrackedUsage(t *testing.T, db *sql.DB, vehicleID string, mileage, hours int32) {
	t.Helper()

	_, err := db.Exec(
		`UPDATE shop_vehicle SET tracked_mileage = $1, tracked_hours = $2 WHERE id = $3`,
		mileage,
		hours,
		vehicleID,
	)
	require.NoError(t, err)
}

func TestVehicleCRUD(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "user-1")

	router := newTestRouter(t)

	shopID := createShop(t, router, "user-1", "Vehicle Shop")
	vehicleID := createVehicle(t, router, "user-1", shopID)

	getListResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/"+shopID+"/vehicles", nil, "user-1")
	require.Equal(t, http.StatusOK, getListResp.Code)

	list := decodeStandardResponse(t, getListResp.Body)
	vehicles := decodeSlice(t, list.Data)
	require.Len(t, vehicles, 1)

	getByIDResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/vehicles/"+vehicleID, nil, "user-1")
	require.Equal(t, http.StatusOK, getByIDResp.Code)

	updateBody := map[string]interface{}{
		"vehicle_id": vehicleID,
		"admin":      "updated-admin",
		"niin":       "2222-22-222-2222",
		"model":      "Model X",
		"serial":     "SERIAL-1",
		"uoc":        "UOC",
		"mileage":    5,
		"hours":      10,
		"comment":    "updated",
	}

	updateResp := doJSONRequest(t, router, http.MethodPut, "/api/v1/auth/shops/vehicles", updateBody, "user-1")
	require.Equal(t, http.StatusOK, updateResp.Code)

	deleteResp := doJSONRequest(t, router, http.MethodDelete, "/api/v1/auth/shops/vehicles/"+vehicleID, nil, "user-1")
	require.Equal(t, http.StatusOK, deleteResp.Code)

	getListAfterDelete := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/"+shopID+"/vehicles", nil, "user-1")
	require.Equal(t, http.StatusOK, getListAfterDelete.Code)

	listAfterDelete := decodeStandardResponse(t, getListAfterDelete.Body)
	vehiclesAfterDelete := decodeSlice(t, listAfterDelete.Data)
	require.Len(t, vehiclesAfterDelete, 0)
}

func TestVehicleUsageUpdateRequiresShopMembership(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "vehicle-owner")
	ensureUser(t, testDB, "shop-member")
	ensureUser(t, testDB, "shop-outsider")

	router := newTestRouter(t)
	shopID := createShop(t, router, "vehicle-owner", "Usage Shop")

	createBody := map[string]interface{}{
		"shop_id": shopID,
		"admin":   "ADMIN-001",
		"niin":    "1234-00-123-4567",
		"model":   "M1123",
		"serial":  "SERIAL-001",
		"uoc":     "ABC",
		"mileage": 100,
		"hours":   50,
		"comment": "owner-managed metadata",
	}
	createResp := doJSONRequest(t, router, http.MethodPost, "/api/v1/auth/shops/vehicles", createBody, "vehicle-owner")
	require.Equal(t, http.StatusCreated, createResp.Code)
	createdVehicle := decodeMap(t, decodeStandardResponse(t, createResp.Body).Data)
	vehicleID, ok := createdVehicle["id"].(string)
	require.True(t, ok)

	_, inviteCode := createInviteCode(t, router, "vehicle-owner", shopID)
	joinResp := doJSONRequest(
		t,
		router,
		http.MethodPost,
		"/api/v1/auth/shops/join",
		map[string]interface{}{"invite_code": inviteCode},
		"shop-member",
	)
	require.Equal(t, http.StatusOK, joinResp.Code)

	ownerUpdate := map[string]interface{}{
		"vehicle_id":      vehicleID,
		"admin":           "ADMIN-001",
		"niin":            "1234-00-123-4567",
		"model":           "M1123",
		"serial":          "SERIAL-001",
		"uoc":             "ABC",
		"mileage":         110,
		"hours":           55,
		"tracked_mileage": 110,
		"tracked_hours":   55,
		"comment":         "owner-managed metadata",
	}
	ownerUpdateResp := doJSONRequest(t, router, http.MethodPut, "/api/v1/auth/shops/vehicles", ownerUpdate, "vehicle-owner")
	require.Equal(t, http.StatusOK, ownerUpdateResp.Code)

	memberUpdate := map[string]interface{}{
		"vehicle_id":      vehicleID,
		"admin":           "ADMIN-001",
		"mileage":         100,
		"hours":           50,
		"tracked_mileage": 125,
		"tracked_hours":   60,
	}
	memberUpdateResp := doJSONRequest(t, router, http.MethodPut, "/api/v1/auth/shops/vehicles", memberUpdate, "shop-member")
	require.Equal(t, http.StatusOK, memberUpdateResp.Code)

	getAfterMemberUpdate := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/vehicles/"+vehicleID, nil, "shop-member")
	require.Equal(t, http.StatusOK, getAfterMemberUpdate.Code)
	updatedVehicle := decodeMap(t, decodeStandardResponse(t, getAfterMemberUpdate.Body).Data)
	require.Equal(t, float64(125), updatedVehicle["tracked_mileage"])
	require.Equal(t, float64(60), updatedVehicle["tracked_hours"])
	require.Equal(t, "ADMIN-001", updatedVehicle["admin"])
	require.Equal(t, "1234-00-123-4567", updatedVehicle["niin"])
	require.Equal(t, "M1123", updatedVehicle["model"])
	require.Equal(t, "SERIAL-001", updatedVehicle["serial"])
	require.Equal(t, "ABC", updatedVehicle["uoc"])
	require.Equal(t, float64(110), updatedVehicle["mileage"])
	require.Equal(t, float64(55), updatedVehicle["hours"])
	require.Equal(t, "owner-managed metadata", updatedVehicle["comment"])

	memberMetadataUpdate := map[string]interface{}{
		"vehicle_id":      vehicleID,
		"admin":           "ADMIN-001",
		"model":           "UNAUTHORIZED-METADATA-CHANGE",
		"mileage":         100,
		"hours":           50,
		"tracked_mileage": 500,
		"tracked_hours":   500,
	}
	memberMetadataUpdateResp := doJSONRequest(t, router, http.MethodPut, "/api/v1/auth/shops/vehicles", memberMetadataUpdate, "shop-member")
	require.Equal(t, http.StatusInternalServerError, memberMetadataUpdateResp.Code)

	outsiderUpdate := map[string]interface{}{
		"vehicle_id":      vehicleID,
		"admin":           "ADMIN-001",
		"tracked_mileage": 500,
		"tracked_hours":   500,
	}
	outsiderUpdateResp := doJSONRequest(t, router, http.MethodPut, "/api/v1/auth/shops/vehicles", outsiderUpdate, "shop-outsider")
	require.Equal(t, http.StatusInternalServerError, outsiderUpdateResp.Code)

	getAfterOutsiderUpdate := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/vehicles/"+vehicleID, nil, "shop-member")
	require.Equal(t, http.StatusOK, getAfterOutsiderUpdate.Code)
	vehicleAfterOutsiderUpdate := decodeMap(t, decodeStandardResponse(t, getAfterOutsiderUpdate.Body).Data)
	require.Equal(t, float64(125), vehicleAfterOutsiderUpdate["tracked_mileage"])
	require.Equal(t, float64(60), vehicleAfterOutsiderUpdate["tracked_hours"])
}
