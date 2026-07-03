package shops_test

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

const (
	aggregatePerfUserID      = "aggregate-perf-user"
	aggregatePerfSnapshotID  = "agg-perf-shop-01"
	aggregatePerfRequestRuns = 50
)

func TestShopsAggregatePerformance(t *testing.T) {
	if os.Getenv("SHOP_AGGREGATE_PERF") != "1" {
		t.Skip("set SHOP_AGGREGATE_PERF=1 to run the representative Shops aggregate performance test")
	}

	clearShopTables(t, testDB)
	t.Cleanup(func() {
		clearShopTables(t, testDB)
		deleteAggregatePerformanceUser(t)
	})
	deleteAggregatePerformanceUser(t)
	ensureUser(t, testDB, aggregatePerfUserID)
	createShopsAggregatePerformanceFixture(t)
	logBootstrapCoreQueryPlan(t)

	router := newTestRouter(t)

	bootstrapPath := "/api/v1/auth/shops/bootstrap"
	bootstrapStats := measureAggregateEndpoint(t, router, bootstrapPath, aggregatePerfUserID, assertBootstrapWarmupPayload)
	t.Logf(
		"/shops/bootstrap p50=%s p95=%s uncompressed_bytes=%d gzip_bytes=%d",
		bootstrapStats.p50,
		bootstrapStats.p95,
		bootstrapStats.uncompressedBytes,
		bootstrapStats.gzipBytes,
	)

	snapshotPath := "/api/v1/auth/shops/" + aggregatePerfSnapshotID + "/snapshot"
	snapshotStats := measureAggregateEndpoint(t, router, snapshotPath, aggregatePerfUserID, assertShopSnapshotWarmupPayload)
	t.Logf(
		"/shops/:shop_id/snapshot p50=%s p95=%s uncompressed_bytes=%d gzip_bytes=%d",
		snapshotStats.p50,
		snapshotStats.p95,
		snapshotStats.uncompressedBytes,
		snapshotStats.gzipBytes,
	)
}

func deleteAggregatePerformanceUser(t *testing.T) {
	t.Helper()

	_, err := testDB.Exec(`DELETE FROM users WHERE uid=$1`, aggregatePerfUserID)
	require.NoError(t, err)
}

func createShopsAggregatePerformanceFixture(t *testing.T) {
	t.Helper()

	execAggregatePerformanceSQL(t, `
		INSERT INTO shops (id, name, details, created_by, created_at, updated_at)
		SELECT
			'agg-perf-shop-' || lpad(n::text, 2, '0'),
			'Aggregate Performance Shop ' || n,
			'Representative aggregate fixture',
			$1,
			now() - (n * interval '1 second'),
			now()
		FROM generate_series(1, 25) AS n;
	`, aggregatePerfUserID)

	execAggregatePerformanceSQL(t, `
		INSERT INTO shop_members (id, shop_id, user_id, role, joined_at)
		SELECT
			'agg-perf-member-' || lpad(n::text, 2, '0'),
			'agg-perf-shop-' || lpad(n::text, 2, '0'),
			$1,
			'member',
			now()
		FROM generate_series(1, 25) AS n;
	`, aggregatePerfUserID)

	execAggregatePerformanceSQL(t, `
		INSERT INTO shop_vehicle (
			id, creator_id, niin, admin, model, serial, uoc,
			mileage, hours, comment, save_time, last_updated, shop_id
		)
		SELECT
			'agg-perf-equipment-' || lpad(n::text, 3, '0'),
			$1,
			lpad(n::text, 9, '0'),
			'ADMIN-' || n,
			'MODEL-' || ((n - 1) % 50 + 1),
			'SERIAL-' || n,
			'UNK',
			0,
			0,
			'',
			now() - (n * interval '1 millisecond'),
			now(),
			'agg-perf-shop-' || lpad((((n - 1) % 25) + 1)::text, 2, '0')
		FROM generate_series(1, 250) AS n;
	`, aggregatePerfUserID)

	execAggregatePerformanceSQL(t, `
		INSERT INTO shop_lists (id, shop_id, created_by, description, created_at, updated_at)
		SELECT
			'agg-perf-list-' || lpad(n::text, 3, '0'),
			'agg-perf-shop-' || lpad((((n - 1) % 25) + 1)::text, 2, '0'),
			$1,
			'Aggregate performance list ' || n,
			now() - (n * interval '1 millisecond'),
			now()
		FROM generate_series(1, 250) AS n;
	`, aggregatePerfUserID)

	execAggregatePerformanceSQL(t, `
		INSERT INTO shop_list_items (
			id, list_id, niin, nomenclature, quantity, added_by,
			created_at, updated_at, unit_of_measure
		)
		SELECT
			'agg-perf-list-item-' || lpad(n::text, 4, '0'),
			'agg-perf-list-' || lpad((((n - 1) % 250) + 1)::text, 3, '0'),
			lpad(n::text, 9, '0'),
			'Aggregate Performance Item ' || n,
			1,
			$1,
			now() - (n * interval '1 millisecond'),
			now(),
			'ea'
		FROM generate_series(1, 2500) AS n;
	`, aggregatePerfUserID)

	execAggregatePerformanceSQL(t, `
		INSERT INTO shop_vehicle_notifications (
			id, shop_id, vehicle_id, title, description, type,
			completed, save_time, last_updated
		)
		SELECT
			'agg-perf-notification-' || lpad(n::text, 3, '0'),
			'agg-perf-shop-' || lpad((((n - 1) % 25) + 1)::text, 2, '0'),
			'agg-perf-equipment-' || lpad(n::text, 3, '0'),
			'Aggregate PM ' || n,
			'Representative notification',
			'PM',
			false,
			now() - (n * interval '1 millisecond'),
			now()
		FROM generate_series(1, 250) AS n;
	`)

	execAggregatePerformanceSQL(t, `
		INSERT INTO shop_notification_items (
			id, shop_id, notification_id, niin, nomenclature, quantity, save_time
		)
		SELECT
			'agg-perf-notif-item-' || lpad(n::text, 3, '0'),
			'agg-perf-shop-' || lpad(((((n - 1) / 2) % 25) + 1)::text, 2, '0'),
			'agg-perf-notification-' || lpad((((n - 1) / 2) + 1)::text, 3, '0'),
			lpad(n::text, 9, '0'),
			'Aggregate Notification Item ' || n,
			1,
			now() - (n * interval '1 millisecond')
		FROM generate_series(1, 500) AS n;
	`)

	execAggregatePerformanceSQL(t, `
		INSERT INTO equipment_services (
			id, shop_id, equipment_id, list_id, description, service_type,
			created_by, is_completed, created_at, updated_at, service_date
		)
		SELECT
			'agg-perf-service-' || lpad(n::text, 3, '0'),
			'agg-perf-shop-' || lpad((((((n - 1) % 250) + 1) - 1) % 25 + 1)::text, 2, '0'),
			'agg-perf-equipment-' || lpad((((n - 1) % 250) + 1)::text, 3, '0'),
			'agg-perf-list-' || lpad((((n - 1) % 250) + 1)::text, 3, '0'),
			'Aggregate service ' || n,
			'inspection',
			$1,
			(n % 4 = 0),
			now() - (n * interval '1 millisecond'),
			now(),
			now() + (n * interval '1 hour')
		FROM generate_series(1, 500) AS n;
	`, aggregatePerfUserID)

	for _, tableName := range []string{
		"shops",
		"shop_members",
		"shop_vehicle",
		"shop_lists",
		"shop_list_items",
		"shop_vehicle_notifications",
		"shop_notification_items",
		"equipment_services",
	} {
		execAggregatePerformanceSQL(t, "ANALYZE "+tableName)
	}

	assertAggregatePerformanceFixtureCounts(t)
}

func execAggregatePerformanceSQL(t *testing.T, query string, args ...any) {
	t.Helper()

	_, err := testDB.Exec(query, args...)
	require.NoError(t, err)
}

func assertAggregatePerformanceFixtureCounts(t *testing.T) {
	t.Helper()

	expectedCounts := map[string]int{
		"shops":                      25,
		"shop_vehicle":               250,
		"shop_lists":                 250,
		"shop_list_items":            2500,
		"shop_vehicle_notifications": 250,
		"shop_notification_items":    500,
		"equipment_services":         500,
	}
	for tableName, expectedCount := range expectedCounts {
		var actualCount int
		err := testDB.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)).Scan(&actualCount)
		require.NoError(t, err)
		require.Equal(t, expectedCount, actualCount, tableName)
	}
}

func logBootstrapCoreQueryPlan(t *testing.T) {
	t.Helper()

	// Keep this SQL in sync with the core summary query in api/shops/aggregates.RepositoryImpl.GetBootstrap.
	planRows, err := testDB.Query(`
EXPLAIN (ANALYZE, BUFFERS, FORMAT TEXT)
SELECT
	s.id,
	s.name,
	s.details,
	sm.role,
	(sm.role = 'admin') AS is_admin,
	s.admin_only_lists,
	(SELECT COUNT(*) FROM shop_members m WHERE m.shop_id = s.id) AS member_count,
	(SELECT COUNT(*) FROM shop_vehicle v WHERE v.shop_id = s.id) AS vehicle_count,
	(SELECT COUNT(*) FROM shop_lists l WHERE l.shop_id = s.id) AS list_count,
	(SELECT COUNT(*) FROM shop_messages msg WHERE msg.shop_id = s.id) AS message_count,
	(SELECT COUNT(*) FROM shop_vehicle_notifications n WHERE n.shop_id = s.id) AS notification_count,
	(SELECT COUNT(*) FROM shop_notification_items ni WHERE ni.shop_id = s.id) AS notification_item_count,
	(SELECT COUNT(*) FROM equipment_services es WHERE es.shop_id = s.id AND es.is_completed = false) AS open_service_count,
	(SELECT COUNT(*) FROM equipment_services es WHERE es.shop_id = s.id) AS service_count,
	(SELECT COUNT(*) FROM shop_vehicle_notification_changes c WHERE c.shop_id = s.id) AS recent_change_count
FROM shop_members sm
INNER JOIN shops s ON s.id = sm.shop_id
WHERE sm.user_id = $1
ORDER BY s.created_at DESC NULLS LAST, s.id DESC`, aggregatePerfUserID)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, planRows.Close())
	}()

	t.Log("EXPLAIN (ANALYZE, BUFFERS) for /shops/bootstrap core query:")
	for planRows.Next() {
		var line string
		require.NoError(t, planRows.Scan(&line))
		t.Log(line)
	}
	require.NoError(t, planRows.Err())
}

type aggregateEndpointStats struct {
	p50               time.Duration
	p95               time.Duration
	uncompressedBytes int
	gzipBytes         int
}

func measureAggregateEndpoint(
	t *testing.T,
	router *gin.Engine,
	path string,
	userID string,
	assertWarmupPayload func(*testing.T, []byte),
) aggregateEndpointStats {
	t.Helper()

	uncompressedDurations, uncompressedBytes, uncompressedWarmupBody := measureAggregateEndpointVariant(
		t,
		router,
		path,
		userID,
		false,
		assertWarmupPayload,
	)
	_, gzipBytes, gzipWarmupBody := measureAggregateEndpointVariant(
		t,
		router,
		path,
		userID,
		true,
		assertWarmupPayload,
	)
	require.JSONEq(t, string(uncompressedWarmupBody), string(gzipWarmupBody))

	return aggregateEndpointStats{
		p50:               percentileDuration(uncompressedDurations, 0.50),
		p95:               percentileDuration(uncompressedDurations, 0.95),
		uncompressedBytes: uncompressedBytes,
		gzipBytes:         gzipBytes,
	}
}

func measureAggregateEndpointVariant(
	t *testing.T,
	router *gin.Engine,
	path string,
	userID string,
	requestGzip bool,
	assertWarmupPayload func(*testing.T, []byte),
) ([]time.Duration, int, []byte) {
	t.Helper()

	durations := make([]time.Duration, 0, aggregatePerfRequestRuns)
	responseBytes := 0
	var warmupBody []byte
	for requestNumber := 0; requestNumber <= aggregatePerfRequestRuns; requestNumber++ {
		req, err := http.NewRequest(http.MethodGet, path, nil)
		require.NoError(t, err)
		req.Header.Set("X-User-ID", userID)
		if requestGzip {
			req.Header.Set("Accept-Encoding", "gzip")
		}

		resp := httptest.NewRecorder()
		startedAt := time.Now()
		router.ServeHTTP(resp, req)
		duration := time.Since(startedAt)

		require.Equal(t, http.StatusOK, resp.Code)
		if requestGzip {
			require.Equal(t, "gzip", resp.Header().Get("Content-Encoding"))
		}
		if requestNumber == 0 {
			warmupBody = decodeAggregateWarmupBody(t, resp, requestGzip)
			assertWarmupPayload(t, warmupBody)
			continue
		}

		durations = append(durations, duration)
		responseBytes = resp.Body.Len()
	}

	return durations, responseBytes, warmupBody
}

func decodeAggregateWarmupBody(t *testing.T, resp *httptest.ResponseRecorder, isGzip bool) []byte {
	t.Helper()

	if !isGzip {
		body := append([]byte(nil), resp.Body.Bytes()...)
		require.True(t, json.Valid(body))
		return body
	}

	reader, err := gzip.NewReader(bytes.NewReader(resp.Body.Bytes()))
	require.NoError(t, err)
	body, err := io.ReadAll(reader)
	require.NoError(t, err)
	require.NoError(t, reader.Close())
	require.True(t, json.Valid(body))
	return body
}

func assertBootstrapWarmupPayload(t *testing.T, body []byte) {
	t.Helper()

	payload := decodeMap(t, decodeStandardResponse(t, bytes.NewBuffer(body)).Data)
	shops := payload["shops"].([]interface{})
	require.Len(t, shops, 25)

	shopsByID := shopsByID(shops)
	shop := shopsByID[aggregatePerfSnapshotID]
	require.NotNil(t, shop)
	require.Equal(t, "Aggregate Performance Shop 1", shop["name"])
	require.Equal(t, "member", shop["role"])

	counts := shop["counts"].(map[string]interface{})
	require.Equal(t, float64(1), counts["members"])
	require.Equal(t, float64(10), counts["vehicles"])
	require.Equal(t, float64(10), counts["lists"])
	require.Equal(t, float64(0), counts["messages"])
	require.Equal(t, float64(10), counts["notifications"])
	require.Equal(t, float64(20), counts["notification_items"])
	require.Equal(t, float64(15), counts["open_services"])
	require.Equal(t, float64(20), counts["services"])
	require.NotContains(t, counts, "recent_changes")

	equipment := shop["equipment"].([]interface{})
	require.Len(t, equipment, 10)
	firstEquipment := equipment[0].(map[string]interface{})
	require.Equal(t, "agg-perf-equipment-001", firstEquipment["id"])
	require.Equal(t, "ADMIN-1", firstEquipment["admin"])
	require.Equal(t, "MODEL-1", firstEquipment["model"])
}

func assertShopSnapshotWarmupPayload(t *testing.T, body []byte) {
	t.Helper()

	payload := decodeMap(t, decodeStandardResponse(t, bytes.NewBuffer(body)).Data)
	shop := payload["shop"].(map[string]interface{})
	require.Equal(t, aggregatePerfSnapshotID, shop["id"])
	require.Equal(t, "Aggregate Performance Shop 1", shop["name"])
	require.Equal(t, "member", shop["role"])

	counts := shop["counts"].(map[string]interface{})
	require.Equal(t, float64(1), counts["members"])
	require.Equal(t, float64(10), counts["vehicles"])
	require.Equal(t, float64(10), counts["lists"])
	require.Equal(t, float64(0), counts["messages"])
	require.Equal(t, float64(10), counts["notifications"])
	require.Equal(t, float64(15), counts["open_services"])

	vehicles := payload["vehicles"].([]interface{})
	require.Len(t, vehicles, 10)
	require.Equal(t, "agg-perf-equipment-001", vehicles[0].(map[string]interface{})["id"])

	lists := payload["lists"].([]interface{})
	require.Len(t, lists, 10)
	firstList := lists[0].(map[string]interface{})
	require.Equal(t, "agg-perf-list-001", firstList["id"])
	require.Len(t, firstList["items"].([]interface{}), 10)

	notifications := payload["notifications"].([]interface{})
	require.Len(t, notifications, 10)
	firstNotification := notifications[0].(map[string]interface{})
	notification := firstNotification["notification"].(map[string]interface{})
	require.Equal(t, "agg-perf-notification-001", notification["id"])
	require.Len(t, firstNotification["items"].([]interface{}), 2)

	services := payload["services"].([]interface{})
	require.Len(t, services, 20)
	require.NotEmpty(t, services[0].(map[string]interface{})["id"])
	require.Empty(t, payload["messages"].([]interface{}))
	require.Empty(t, payload["recent_changes"].([]interface{}))
}

func shopsByID(shops []interface{}) map[string]map[string]interface{} {
	result := make(map[string]map[string]interface{}, len(shops))
	for _, rawShop := range shops {
		shop := rawShop.(map[string]interface{})
		result[shop["id"].(string)] = shop
	}
	return result
}

func percentileDuration(values []time.Duration, percentile float64) time.Duration {
	if len(values) == 0 {
		return 0
	}

	sortedValues := append([]time.Duration(nil), values...)
	sort.Slice(sortedValues, func(i, j int) bool { return sortedValues[i] < sortedValues[j] })

	index := int(math.Ceil(percentile*float64(len(sortedValues)))) - 1
	if index < 0 {
		index = 0
	}
	if index >= len(sortedValues) {
		index = len(sortedValues) - 1
	}
	return sortedValues[index]
}
