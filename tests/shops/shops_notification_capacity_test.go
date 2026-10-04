package shops_test

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"miltechserver/api/middleware"
	"miltechserver/api/response"
	"miltechserver/api/shops"
	"miltechserver/api/shops/aggregates"
	"miltechserver/bootstrap"
	"miltechserver/tests/testutil"
	"net/http"
	"testing"
	"time"
)

func TestNotificationUnboundedParameterCapacity(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "capacity-reader")
	ensureUser(t, testDB, "capacity-outsider")
	router := newTestRouter(t)
	shopID := createShop(t, router, "capacity-reader", "capacity")
	vehicleID := createVehicle(t, router, "capacity-reader", shopID)
	foreignShop := createShop(t, router, "capacity-outsider", "foreign")
	foreignVehicle := createVehicle(t, router, "capacity-outsider", foreignShop)
	createNotificationRow(t, foreignShop, foreignVehicle, "private", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	// The same route handler stack runs through a counting physical driver.
	probe := &aggregateSnapshotProbe{}
	database := aggregateSnapshotDatabase(t, probe)
	measured := gin.New()
	measured.Use(middleware.ErrorHandler, testutil.FakeAuthMiddleware())
	shops.RegisterRoutes(shops.Dependencies{DB: database, Env: &bootstrap.Env{BlobAccountName: "test-account"}}, measured.Group("/api/v1/auth"))
	for _, count := range []int{65534, 65535, 65536, 67000} {
		_, err := testDB.Exec(`INSERT INTO shop_vehicle_notifications(id,shop_id,vehicle_id,title,description,type,completed,save_time,last_updated) SELECT 'capacity-'||lpad(n::text,6,'0'),$1,$2,'capacity','description','PM',false,'2026-01-01'::timestamp,'2026-01-01'::timestamp FROM generate_series(1,$3) n ON CONFLICT DO NOTHING`, shopID, vehicleID, count)
		require.NoError(t, err)
		_, err = testDB.Exec(`INSERT INTO shop_notification_items(id,shop_id,notification_id,niin,nomenclature,quantity,save_time) SELECT 'item-'||n.id,$1,n.id,'123456789','capacity',1,'2026-01-01'::timestamp FROM shop_vehicle_notifications n WHERE n.shop_id=$1 ON CONFLICT DO NOTHING`, shopID)
		require.NoError(t, err)
		for _, kind := range []string{"legacy", "vehicle", "shop"} {
			t.Run(fmt.Sprintf("%s/%d", kind, count), func(t *testing.T) {
				path := "/api/v1/auth/shops/vehicles/" + vehicleID + "/notifications-with-items"
				if kind == "vehicle" {
					path = "/api/v1/auth/shops/vehicles/" + vehicleID + "/maintenance-snapshot"
				}
				if kind == "shop" {
					path = "/api/v1/auth/shops/" + shopID + "/snapshot?include=notifications"
				}
				result := doJSONRequest(t, measured, http.MethodGet, path, nil, "capacity-reader")
				require.Equal(t, http.StatusOK, result.Code)
				raw := decodeStandardResponse(t, result.Body).Data
				var rows []response.VehicleNotificationWithItems
				if kind == "legacy" {
					require.NoError(t, json.Unmarshal(raw, &rows))
				} else {
					var payload struct {
						Notifications []response.VehicleNotificationWithItems `json:"notifications"`
					}
					require.NoError(t, json.Unmarshal(raw, &payload))
					rows = payload.Notifications
				}
				require.Len(t, rows, count)
				seen := map[string]bool{}
				for _, row := range rows {
					require.Equal(t, shopID, row.Notification.ShopID)
					require.False(t, seen[row.Notification.ID])
					seen[row.Notification.ID] = true
					require.Len(t, row.Items, 1)
					require.Equal(t, row.Notification.ID, row.Items[0].NotificationID)
				}
				denied := doJSONRequest(t, measured, http.MethodGet, path, nil, "capacity-outsider")
				require.NotEqual(t, http.StatusOK, denied.Code)
			})
		}
	}
	require.LessOrEqual(t, probe.maxParameters, 2, "selected notification sets must bind as one array plus optional item limit")
	t.Logf("complete notification parents/items=67000; max query bind count=%d", probe.maxParameters)
}

// Keep the fixture timestamp stable so deterministic ID tie ordering is exercised.

func TestBootstrapEquipmentBoundedParameters(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "bootstrap-reader")
	ensureUser(t, testDB, "bootstrap-other")
	router := newTestRouter(t)
	expected := map[string]string{}
	for i := 0; i < 4; i++ {
		shop := createShop(t, router, "bootstrap-reader", fmt.Sprint(i))
		old := createVehicle(t, router, "bootstrap-reader", shop)
		_, err := testDB.Exec(`UPDATE shop_vehicle SET save_time='2026-01-01',admin='older' WHERE id=$1`, old)
		require.NoError(t, err)
		latest := createVehicle(t, router, "bootstrap-reader", shop)
		expected[shop] = latest
	}
	foreign := createShop(t, router, "bootstrap-other", "private")
	createVehicle(t, router, "bootstrap-other", foreign)
	probe := &aggregateSnapshotProbe{}
	repo := aggregates.NewRepository(aggregateSnapshotDatabase(t, probe))
	result, err := repo.GetBootstrap(context.Background(), &bootstrap.User{UserID: "bootstrap-reader"}, aggregates.BootstrapOptions{EquipmentLimitPerShop: 1})
	require.NoError(t, err)
	require.Len(t, result, 4)
	for _, row := range result {
		require.Contains(t, expected, row.ID)
		require.Len(t, row.Equipment, 1)
		require.Equal(t, expected[row.ID], row.Equipment[0].ID)
	}
	require.Equal(t, 2, probe.maxParameters)
	requireAggregateSnapshot(t, probe)
	result, err = repo.GetBootstrap(context.Background(), &bootstrap.User{UserID: "bootstrap-reader"}, aggregates.BootstrapOptions{})
	require.NoError(t, err)
	for _, row := range result {
		require.Len(t, row.Equipment, 2)
	}
}
