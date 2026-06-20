package shops_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"testing"
	"time"

	"miltechserver/api/response"

	"github.com/stretchr/testify/require"
)

func TestShopEquipmentOverviewPerformance(t *testing.T) {
	if os.Getenv("SHOP_OVERVIEW_PERF") != "1" {
		t.Skip("set SHOP_OVERVIEW_PERF=1 to run the representative performance test")
	}
	clearShopTables(t, testDB)
	t.Cleanup(func() { clearShopTables(t, testDB) })
	ensureUser(t, testDB, "overview-perf-user")
	_, err := testDB.Exec(`
		INSERT INTO shops (id, name, details, created_by, created_at, updated_at)
		SELECT 'overview-perf-shop-' || n, 'Performance Shop ' || n,
			   'Performance fixture', 'overview-perf-user',
			   now() - (n * interval '1 second'), now()
		FROM generate_series(1, 100) AS n;

		INSERT INTO shop_members (id, shop_id, user_id, role, joined_at)
		SELECT 'overview-perf-member-' || n, 'overview-perf-shop-' || n,
			   'overview-perf-user', 'member', now()
		FROM generate_series(1, 100) AS n;

		INSERT INTO shop_vehicle (
			id, creator_id, niin, admin, model, serial, uoc,
			mileage, hours, comment, save_time, last_updated, shop_id
		)
		SELECT 'overview-perf-equipment-' || n, 'overview-perf-user',
			   lpad(n::text, 9, '0'), 'ADMIN-' || n, 'MODEL-' || (n % 50),
			   'SERIAL-' || n, 'UNK', 0, 0, '',
			   now() - (n * interval '1 millisecond'), now(),
			   'overview-perf-shop-' || (((n - 1) % 100) + 1)
		FROM generate_series(1, 25000) AS n;

		ANALYZE shops;
		ANALYZE shop_members;
		ANALYZE shop_vehicle;
	`)
	require.NoError(t, err)

	// Keep this EXPLAIN SQL in sync with the production overview repository query so plan evidence tracks real endpoint behavior.
	planRows, err := testDB.Query(`
		EXPLAIN (ANALYZE, BUFFERS, FORMAT TEXT)
		SELECT s.id, s.name, s.details, sm.role,
			   v.id, v.admin, v.model, v.serial, v.niin
		FROM shops s
		INNER JOIN shop_members sm ON sm.shop_id = s.id
		LEFT JOIN shop_vehicle v ON v.shop_id = s.id
		WHERE sm.user_id = 'overview-perf-user'
		ORDER BY s.created_at DESC, v.save_time DESC, v.id DESC
	`)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, planRows.Close())
	}()
	for planRows.Next() {
		var line string
		require.NoError(t, planRows.Scan(&line))
		t.Log(line)
	}
	require.NoError(t, planRows.Err())

	router := newTestRouter(t)
	const measuredRequests = 100
	durations := make([]time.Duration, 0, measuredRequests)
	responseSize := 0
	for requestNumber := 0; requestNumber <= measuredRequests; requestNumber++ {
		startedAt := time.Now()
		resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/equipment/overview", nil, "overview-perf-user")
		duration := time.Since(startedAt)
		require.Equal(t, http.StatusOK, resp.Code)
		if requestNumber == 0 {
			assertShopEquipmentOverviewPayload(t, resp)
			continue
		}
		durations = append(durations, duration)
		responseSize = resp.Body.Len()
	}
	sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })
	p50 := durations[49]
	p95 := durations[94]
	compressedDurations := make([]time.Duration, 0, measuredRequests)
	compressedSize := 0
	for requestNumber := 0; requestNumber <= measuredRequests; requestNumber++ {
		compressedReq, requestErr := http.NewRequest(http.MethodGet, "/api/v1/auth/shops/equipment/overview", nil)
		require.NoError(t, requestErr)
		compressedReq.Header.Set("X-User-ID", "overview-perf-user")
		compressedReq.Header.Set("Accept-Encoding", "gzip")
		compressedResp := httptest.NewRecorder()
		startedAt := time.Now()
		router.ServeHTTP(compressedResp, compressedReq)
		duration := time.Since(startedAt)
		require.Equal(t, http.StatusOK, compressedResp.Code)
		require.Equal(t, "gzip", compressedResp.Header().Get("Content-Encoding"))
		if requestNumber == 0 {
			continue
		}
		compressedDurations = append(compressedDurations, duration)
		compressedSize = compressedResp.Body.Len()
	}
	sort.Slice(compressedDurations, func(i, j int) bool { return compressedDurations[i] < compressedDurations[j] })
	compressedP95 := compressedDurations[94]
	t.Logf("warm-cache p50=%s p95=%s compressed_p95=%s uncompressed_bytes=%d compressed_bytes=%d", p50, p95, compressedP95, responseSize, compressedSize)
	require.Less(t, p95, time.Second, fmt.Sprintf("p95 target missed: %s", p95))
	require.Less(t, compressedP95, time.Second, fmt.Sprintf("compressed p95 target missed: %s", compressedP95))
}

func assertShopEquipmentOverviewPayload(t *testing.T, resp *httptest.ResponseRecorder) {
	t.Helper()

	decoded := decodeStandardResponse(t, resp.Body)
	var overview response.ShopEquipmentOverviewResponse
	require.NoError(t, json.Unmarshal(decoded.Data, &overview))
	require.Len(t, overview.Shops, 100)

	equipmentCount := 0
	equipmentLength := 0
	for _, shop := range overview.Shops {
		require.NotNil(t, shop.Equipment)
		equipmentCount += shop.EquipmentCount
		equipmentLength += len(shop.Equipment)
	}
	require.Equal(t, 25000, equipmentCount)
	require.Equal(t, 25000, equipmentLength)
}
