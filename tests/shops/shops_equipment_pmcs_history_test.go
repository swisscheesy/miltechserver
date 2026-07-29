package shops_test

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"miltechserver/api/shops/aggregates"
	"miltechserver/bootstrap"

	"github.com/stretchr/testify/require"
)

func TestGetEquipmentPmcsHistoryRepository(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "history-user")
	ensureUser(t, testDB, "other-user")
	router := newTestRouter(t)

	shopID := createShop(t, router, "history-user", "History Shop")
	hiddenShopID := createShop(t, router, "other-user", "Hidden Shop")
	vehicleWithHistoryID := createVehicle(t, router, "history-user", shopID)
	hiddenVehicleID := createVehicle(t, router, "other-user", hiddenShopID)

	// Create second vehicle in same shop by creating another shop first, then inserting
	shop2ID := createShop(t, router, "history-user", "History Shop 2")
	vehicleWithoutHistoryID := createVehicle(t, router, "history-user", shop2ID)

	// Update vehicles with unique admin/serial
	_, err := testDB.Exec(`UPDATE shop_vehicle SET admin='A1', serial='S1' WHERE id=$1`, vehicleWithHistoryID)
	require.NoError(t, err)
	_, err = testDB.Exec(`UPDATE shop_vehicle SET admin='A2', serial='S2' WHERE id=$1`, vehicleWithoutHistoryID)
	require.NoError(t, err)
	_, err = testDB.Exec(`UPDATE shop_vehicle SET admin='A3', serial='S3' WHERE id=$1`, hiddenVehicleID)
	require.NoError(t, err)

	newerTime := time.Date(2026, time.July, 16, 14, 30, 0, 0, time.UTC)
	olderTime := newerTime.Add(-7 * 24 * time.Hour)
	newerInspectionID := createPmcsInspection(t, testDB, vehicleWithHistoryID, "pmcs_sbs/hmmwv/hmmwv_up_armor_pmcs.json", newerTime, "history-user")
	createPmcsFault(t, testDB, newerInspectionID, "before", 0)
	createPmcsFault(t, testDB, newerInspectionID, "during", 1)
	createPmcsComment(t, testDB, newerInspectionID, "history-user", "first comment")
	createPmcsComment(t, testDB, newerInspectionID, "history-user", "second comment")
	olderInspectionID := createPmcsInspection(t, testDB, vehicleWithHistoryID, "pmcs_sbs/hmmwv/hmmwv_up_armor_pmcs.json", olderTime, "history-user")

	repository := aggregates.NewRepository(testDB)
	equipment, err := repository.GetEquipmentPmcsHistory(context.Background(), &bootstrap.User{UserID: "history-user"})

	require.NoError(t, err)
	require.Len(t, equipment, 2)

	byID := make(map[string]int, len(equipment))
	for i, e := range equipment {
		byID[e.ID] = i
	}
	require.Contains(t, byID, vehicleWithHistoryID)
	require.Contains(t, byID, vehicleWithoutHistoryID)

	withHistory := equipment[byID[vehicleWithHistoryID]]
	require.Equal(t, shopID, withHistory.ShopID)
	require.Len(t, withHistory.HistoricalPmcs, 2)
	require.Equal(t, newerInspectionID, withHistory.HistoricalPmcs[0].ID.String())
	require.Equal(t, 2, withHistory.HistoricalPmcs[0].FaultCount)
	require.Equal(t, 2, withHistory.HistoricalPmcs[0].CommentCount)
	require.Equal(t, olderInspectionID, withHistory.HistoricalPmcs[1].ID.String())
	require.Equal(t, 0, withHistory.HistoricalPmcs[1].FaultCount)
	require.Equal(t, 0, withHistory.HistoricalPmcs[1].CommentCount)

	withoutHistory := equipment[byID[vehicleWithoutHistoryID]]
	require.Equal(t, shop2ID, withoutHistory.ShopID)
	require.Empty(t, withoutHistory.HistoricalPmcs)

	for _, e := range equipment {
		require.NotEqual(t, hiddenShopID, e.ShopID)
	}
}

func TestGetEquipmentPmcsHistoryRepositoryReturnsEmptyForUserWithNoShops(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "lonely-user")

	repository := aggregates.NewRepository(testDB)
	equipment, err := repository.GetEquipmentPmcsHistory(context.Background(), &bootstrap.User{UserID: "lonely-user"})

	require.NoError(t, err)
	require.Empty(t, equipment)
}

func TestGetEquipmentPmcsHistoryRepositoryHonorsCanceledContext(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "history-user")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	repository := aggregates.NewRepository(testDB)
	_, err := repository.GetEquipmentPmcsHistory(ctx, &bootstrap.User{UserID: "history-user"})

	require.Error(t, err)
	require.True(t, errors.Is(err, context.Canceled), err.Error())
}

func TestEquipmentPmcsHistoryEndpoint(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "history-user")
	ensureUser(t, testDB, "other-user")
	router := newTestRouter(t)

	shopID := createShop(t, router, "history-user", "History Shop")
	hiddenShopID := createShop(t, router, "other-user", "Hidden Shop")
	vehicleID := createVehicle(t, router, "history-user", shopID)
	_ = createVehicle(t, router, "other-user", hiddenShopID)
	_, err := testDB.Exec(`UPDATE shop_vehicle SET admin='A1', model='M1152A1', serial='SER-1', niin='000000001' WHERE id=$1`, vehicleID)
	require.NoError(t, err)

	performedDate := time.Date(2026, time.July, 16, 14, 30, 0, 0, time.UTC)
	inspectionID := createPmcsInspection(t, testDB, vehicleID, "pmcs_sbs/hmmwv/hmmwv_up_armor_pmcs.json", performedDate, "history-user")
	createPmcsFault(t, testDB, inspectionID, "before", 0)
	createPmcsComment(t, testDB, inspectionID, "history-user", "a comment")

	resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/equipment-pmcs-history", nil, "history-user")
	require.Equal(t, http.StatusOK, resp.Code)

	payload := decodeMap(t, decodeStandardResponse(t, resp.Body).Data)
	equipment := payload["equipment"].([]interface{})
	require.Len(t, equipment, 1)

	entry := equipment[0].(map[string]interface{})
	require.Equal(t, vehicleID, entry["id"])
	require.Equal(t, shopID, entry["shop_id"])
	require.Equal(t, "A1", entry["admin"])

	history := entry["historical_pmcs"].([]interface{})
	require.Len(t, history, 1)
	inspection := history[0].(map[string]interface{})
	require.Equal(t, inspectionID, inspection["id"])
	require.Equal(t, "pmcs_sbs/hmmwv/hmmwv_up_armor_pmcs.json", inspection["guide_manual"])
	require.Equal(t, float64(1), inspection["fault_count"])
	require.Equal(t, float64(1), inspection["comment_count"])
	require.Equal(t, "history-user", inspection["performed_by"])
	require.Equal(t, "test-user", inspection["performed_by_username"])
}

func TestGetEquipmentPmcsHistoryRepositoryIncludesPerformedByUsername(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "history-user")
	router := newTestRouter(t)

	shopID := createShop(t, router, "history-user", "History Shop")
	vehicleID := createVehicle(t, router, "history-user", shopID)

	performedDate := time.Date(2026, time.July, 16, 14, 30, 0, 0, time.UTC)
	inspectionID := createPmcsInspection(t, testDB, vehicleID, "pmcs_sbs/hmmwv/hmmwv_up_armor_pmcs.json", performedDate, "history-user")

	repository := aggregates.NewRepository(testDB)
	equipment, err := repository.GetEquipmentPmcsHistory(context.Background(), &bootstrap.User{UserID: "history-user"})

	require.NoError(t, err)
	require.Len(t, equipment, 1)
	require.Len(t, equipment[0].HistoricalPmcs, 1)
	entry := equipment[0].HistoricalPmcs[0]
	require.Equal(t, inspectionID, entry.ID.String())
	require.NotNil(t, entry.PerformedBy)
	require.Equal(t, "history-user", *entry.PerformedBy)
	require.NotNil(t, entry.PerformedByUsername)
	require.Equal(t, "test-user", *entry.PerformedByUsername)
}

func TestEquipmentPmcsHistoryEndpointRequiresAuthentication(t *testing.T) {
	router := newTestRouter(t)
	resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/equipment-pmcs-history", nil, "")
	require.Equal(t, http.StatusUnauthorized, resp.Code)
}
