package shops_test

import (
	"context"
	"encoding/json"
	"runtime"
	"strings"
	"testing"
	"time"

	"miltechserver/api/response"
	"miltechserver/api/shops/aggregates"
	"miltechserver/bootstrap"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

const pmcsCapacityUserID = "pmcs-capacity-member"
const pmcsCapacityEquipmentID = "ABCDEFAB-1234-4567-89AB-ABCDEFABCDEF"
const pmcsCapacityGuide = "pmcs_sbs/hmmwv/hmmwv_up_armor_pmcs.json"
const pmcsCapacityChecklistID = "10000000-0000-4000-8000-000000000001"
const pmcsCapacityRevisionID = "20000000-0000-4000-8000-000000000002"

// Growing bind lists, response truncation, incorrect source joins, and foreign
// equipment access must all fail this populated physical PostgreSQL regression.
func TestPmcsHistoryBeyondParameterLimit(t *testing.T) {
	seedPmcsHistoryCapacity(t, 65536)
	database, counter := newShopHistoryQueryCountingDatabase(t)
	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	started := time.Now()
	equipment, err := aggregates.NewRepository(database).GetEquipmentPmcsHistory(context.Background(), &bootstrap.User{UserID: pmcsCapacityUserID})
	duration := time.Since(started)
	runtime.ReadMemStats(&after)
	require.NoError(t, err)
	assertPmcsHistoryCapacity(t, equipment, 65536)
	executions := pmcsHistoryExecutions(counter)
	require.Len(t, executions, 4)
	for i, execution := range executions {
		require.Less(t, len(execution.arguments), 65535)
		t.Logf("query %d bound_parameters=%d", i+1, len(execution.arguments))
	}
	payload, err := json.Marshal(response.EquipmentPmcsHistoryResponse{Equipment: equipment, Count: len(equipment)})
	require.NoError(t, err)
	require.NotContains(t, string(payload), "foreign private checklist")
	require.NotContains(t, string(payload), "foreign secret fault")
	require.NotContains(t, string(payload), "foreign secret comment")
	require.NotContains(t, string(payload), "item_to_be_checked_or_serviced")
	t.Logf("full_history inspections=65536 equipment=%d payload_bytes=%d runtime=%s total_alloc_bytes=%d heap_before_bytes=%d heap_after_bytes=%d", len(equipment), len(payload), duration, after.TotalAlloc-before.TotalAlloc, before.HeapAlloc, after.HeapAlloc)
	var serverVersion, workMemory string
	require.NoError(t, testDB.QueryRow(`SHOW server_version`).Scan(&serverVersion))
	require.NoError(t, testDB.QueryRow(`SHOW work_mem`).Scan(&workMemory))
	t.Logf("measurement_environment postgres=%s go=%s os=%s arch=%s work_mem=%s local_single_request=true", serverVersion, runtime.Version(), runtime.GOOS, runtime.GOARCH, workMemory)
	for i, execution := range executions {
		rows, err := testDB.Query("EXPLAIN (ANALYZE, BUFFERS) "+execution.query, execution.arguments...)
		require.NoError(t, err)
		t.Logf("EXPLAIN query %d", i+1)
		for rows.Next() {
			var line string
			require.NoError(t, rows.Scan(&line))
			t.Log(line)
		}
		require.NoError(t, rows.Err())
		require.NoError(t, rows.Close())
	}
	runtime.KeepAlive(equipment)
	_, err = testDB.Exec(`DELETE FROM shop_members WHERE user_id=$1`, pmcsCapacityUserID)
	require.NoError(t, err)
	equipment, err = aggregates.NewRepository(database).GetEquipmentPmcsHistory(context.Background(), &bootstrap.User{UserID: pmcsCapacityUserID})
	require.NoError(t, err)
	require.Empty(t, equipment, "the next snapshot must recheck current membership")
}

func TestPmcsHistoryQueryParameterCount(t *testing.T) {
	for _, size := range []int{2, 128} {
		seedPmcsHistoryCapacity(t, size)
		database, counter := newShopHistoryQueryCountingDatabase(t)
		equipment, err := aggregates.NewRepository(database).GetEquipmentPmcsHistory(context.Background(), &bootstrap.User{UserID: pmcsCapacityUserID})
		require.NoError(t, err)
		assertPmcsHistoryCapacity(t, equipment, size)
		executions := pmcsHistoryExecutions(counter)
		require.Len(t, executions, 4)
		for i, execution := range executions {
			require.Len(t, execution.arguments, 1, "query %d must bind only membership, independent of row count", i+1)
			require.Equal(t, pmcsCapacityUserID, execution.arguments[0])
			t.Logf("inspections=%d query=%d bound_parameters=%d", size, i+1, len(execution.arguments))
		}
	}
}

func pmcsHistoryExecutions(counter *shopHistoryQueryCounter) []shopHistoryQueryExecution {
	counter.mutex.Lock()
	defer counter.mutex.Unlock()
	return append([]shopHistoryQueryExecution(nil), counter.executions...)
}

func seedPmcsHistoryCapacity(t *testing.T, count int) {
	t.Helper()
	const foreignUserID = "pmcs-capacity-foreign"
	clearShopTables(t, testDB)
	t.Cleanup(func() {
		clearShopTables(t, testDB)
		_, err := testDB.Exec(`DELETE FROM users WHERE uid IN ($1, $2)`, pmcsCapacityUserID, foreignUserID)
		require.NoError(t, err)
	})
	ensureUser(t, testDB, pmcsCapacityUserID)
	ensureUser(t, testDB, foreignUserID)
	_, err := testDB.Exec(`UPDATE users SET username='Capacity Tech' WHERE uid=$1`, pmcsCapacityUserID)
	require.NoError(t, err)
	router := newTestRouter(t)
	shopID := createShop(t, router, pmcsCapacityUserID, "PMCS capacity")
	foreignShopID := createShop(t, router, foreignUserID, "Foreign PMCS")
	vehicleID := createVehicle(t, router, pmcsCapacityUserID, shopID)
	foreignVehicleID := createVehicle(t, router, foreignUserID, foreignShopID)
	_, err = testDB.Exec(`UPDATE shop_vehicle SET id=$1 WHERE id=$2`, pmcsCapacityEquipmentID, vehicleID)
	require.NoError(t, err)
	foreignVehicleIDLower := strings.ToLower(pmcsCapacityEquipmentID)
	_, err = testDB.Exec(`UPDATE shop_vehicle SET id=$1 WHERE id=$2`, foreignVehicleIDLower, foreignVehicleID)
	require.NoError(t, err)
	// A case-different TEXT identifier belongs to a foreign Shop; UUID casting or
	// canonicalization would wrongly combine these two equipment histories.
	foreignGuideID := createPmcsInspection(t, testDB, foreignVehicleIDLower, pmcsCapacityGuide, time.Now().UTC(), foreignUserID)
	foreignCustom := createCustomPmcsHistoryFixture(t, testDB, foreignVehicleIDLower, time.Now().UTC(), foreignUserID)
	_, err = testDB.Exec(`UPDATE user_pmcs_inspections SET custom_checklist_name='foreign private checklist' WHERE id=$1`, foreignCustom.ID)
	require.NoError(t, err)
	for _, id := range []string{foreignGuideID, foreignCustom.ID} {
		createPmcsFault(t, testDB, id, "foreign", 0)
		createPmcsComment(t, testDB, id, foreignUserID, "foreign secret comment")
		_, err = testDB.Exec(`UPDATE user_pmcs_faults SET fault_text='foreign secret fault' WHERE pmcs_id=$1`, id)
		require.NoError(t, err)
	}
	// Bounded fixture parameters exercise real source-shape constraints without
	// requiring live authored checklist trees or building another giant bind list.
	_, err = testDB.Exec(`INSERT INTO user_pmcs_inspections
		(id, equipment_id, source_type, guide_manual, custom_checklist_id,
		 custom_revision_id, custom_revision_number, custom_checklist_name,
		 performed_date, performed_by, created_at)
		SELECT md5('capacity-' || n)::uuid, $1,
		 CASE WHEN n % 2 = 0 THEN 'custom' ELSE 'guide' END,
		 CASE WHEN n % 2 = 1 THEN $2::text END,
		 CASE WHEN n % 2 = 0 THEN $3::uuid END,
		 CASE WHEN n % 2 = 0 THEN $4::uuid END,
		 CASE WHEN n % 2 = 0 THEN 7 END,
		 CASE WHEN n % 2 = 0 THEN 'Retired private checklist' END,
		 '2026-10-01T00:00:00Z'::timestamptz + n * interval '1 second', $5,
		 '2026-10-01T00:00:00Z'::timestamptz
		FROM generate_series(1, $6::integer) AS n`, pmcsCapacityEquipmentID, pmcsCapacityGuide, pmcsCapacityChecklistID, pmcsCapacityRevisionID, pmcsCapacityUserID, count)
	require.NoError(t, err)
	_, err = testDB.Exec(`INSERT INTO user_pmcs_faults
		(pmcs_id, section_id, item_index, item_no, status, fault_text)
		SELECT i.id, 'capacity', n, '1', 'x', 'capacity fault'
		FROM user_pmcs_inspections i
		CROSS JOIN LATERAL generate_series(1, CASE WHEN i.source_type='custom' THEN 2 ELSE 1 END) n
		WHERE i.equipment_id=$1`, pmcsCapacityEquipmentID)
	require.NoError(t, err)
	_, err = testDB.Exec(`INSERT INTO user_pmcs_inspection_comments (pmcs_id, author_id, text)
		SELECT i.id, $2, CASE WHEN n=2 THEN 'Deleted by user' ELSE 'capacity comment' END
		FROM user_pmcs_inspections i
		CROSS JOIN LATERAL generate_series(1, CASE WHEN i.source_type='guide' THEN 2 ELSE 1 END) n
		WHERE i.equipment_id=$1`, pmcsCapacityEquipmentID, pmcsCapacityUserID)
	require.NoError(t, err)
	var actual int
	require.NoError(t, testDB.QueryRow(`SELECT count(*) FROM user_pmcs_inspections WHERE equipment_id=$1`, pmcsCapacityEquipmentID).Scan(&actual))
	require.Equal(t, count, actual)
	_, err = testDB.Exec(`ANALYZE shop_vehicle; ANALYZE shop_members; ANALYZE users; ANALYZE user_pmcs_inspections; ANALYZE user_pmcs_faults; ANALYZE user_pmcs_inspection_comments`)
	require.NoError(t, err)
}

func assertPmcsHistoryCapacity(t *testing.T, equipment []response.EquipmentWithPmcsHistory, count int) {
	t.Helper()
	require.Len(t, equipment, 1)
	require.Equal(t, pmcsCapacityEquipmentID, equipment[0].ID)
	accessibleInspections := equipment[0].HistoricalPmcs
	require.Len(t, accessibleInspections, count)
	seen := make(map[uuid.UUID]bool, count)
	var faults, comments int
	for i, inspection := range accessibleInspections {
		require.False(t, seen[inspection.ID], "duplicate inspection")
		seen[inspection.ID] = true
		require.Equal(t, time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC).Add(time.Duration(count-i)*time.Second), inspection.PerformedDate.UTC())
		require.NotNil(t, inspection.PerformedBy)
		require.Equal(t, pmcsCapacityUserID, *inspection.PerformedBy)
		require.NotNil(t, inspection.PerformedByUsername)
		require.Equal(t, "Capacity Tech", *inspection.PerformedByUsername)
		if (count-i)%2 == 0 {
			require.Equal(t, "custom", inspection.SourceType)
			require.Nil(t, inspection.GuideManual)
			require.NotNil(t, inspection.CustomChecklistID)
			require.NotNil(t, inspection.CustomRevisionID)
			require.NotNil(t, inspection.CustomRevisionNumber)
			require.NotNil(t, inspection.CustomChecklistName)
			require.Equal(t, uuid.MustParse(pmcsCapacityChecklistID), *inspection.CustomChecklistID)
			require.Equal(t, uuid.MustParse(pmcsCapacityRevisionID), *inspection.CustomRevisionID)
			require.EqualValues(t, 7, *inspection.CustomRevisionNumber)
			require.Equal(t, "Retired private checklist", *inspection.CustomChecklistName)
			require.Equal(t, 2, inspection.FaultCount)
			require.Equal(t, 1, inspection.CommentCount)
		} else {
			require.Equal(t, "guide", inspection.SourceType)
			require.NotNil(t, inspection.GuideManual)
			require.Equal(t, pmcsCapacityGuide, *inspection.GuideManual)
			require.Nil(t, inspection.CustomChecklistID)
			require.Nil(t, inspection.CustomRevisionID)
			require.Nil(t, inspection.CustomRevisionNumber)
			require.Nil(t, inspection.CustomChecklistName)
			require.Equal(t, 1, inspection.FaultCount)
			require.Equal(t, 2, inspection.CommentCount)
		}
		faults += inspection.FaultCount
		comments += inspection.CommentCount
	}
	require.Equal(t, count/2*3, faults)
	require.Equal(t, count/2*3, comments, "soft-deleted comments still count as persisted rows")
}
