package shops_test

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/shops/messages"
	"miltechserver/bootstrap"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const planRowsPerShop = 10000

// The projection and predicates below are copied from
// api/shops/messages/sync_repository.go with concrete literals, so the planner
// sees what the reader sends (minus parameter binding, which would let a generic
// plan hide the literal-dependent choice). Keep them in step with the reader.
const planProjection = `SELECT m.id,m.shop_id,m.user_id,m.message,m.created_at,m.updated_at,m.is_edited,m.parent_id,NULLIF(BTRIM(u.username),''),m.insertion_number
 FROM shop_messages m LEFT JOIN users u ON u.uid = m.user_id WHERE `

func explainAnalyze(t *testing.T, query string) string {
	t.Helper()
	rows, err := testDB.Query("EXPLAIN (ANALYZE, BUFFERS, FORMAT TEXT) " + query)
	require.NoError(t, err)
	defer rows.Close()
	var lines []string
	for rows.Next() {
		var line string
		require.NoError(t, rows.Scan(&line))
		lines = append(lines, line)
	}
	require.NoError(t, rows.Err())
	return strings.Join(lines, "\n")
}

// shortenLongLines keeps the reconcile ID list from flooding the log.
func shortenLongLines(text string) string {
	const maxLineLength = 220
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if len(line) > maxLineLength {
			lines[i] = line[:maxLineLength] + " ..."
		}
	}
	return strings.Join(lines, "\n")
}

// TestMessageSyncPlan seeds two shops with 10,000 messages each so the planner
// cannot pick a sequential scan simply because the table is tiny, then checks
// each reader query is served by the intended index. Skipped by -short.
func TestMessageSyncPlan(t *testing.T) {
	if testing.Short() {
		t.Skip("plan measurement seeds 20,000 rows; skipped with -short")
	}
	clearShopTables(t, testDB)
	t.Cleanup(func() { clearShopTables(t, testDB) })
	ensureUser(t, testDB, "user-1")
	_, err := testDB.Exec(`INSERT INTO public.shops (id, name, created_by, created_at) VALUES ('plan-a','A','user-1',now()), ('plan-b','B','user-1',now())`)
	require.NoError(t, err)
	// Trigger numbers each row. created_at is spread one second apart, and ids
	// are random-looking text like the server's UUIDs.
	for _, shopID := range []string{"plan-a", "plan-b"} {
		_, err = testDB.Exec(`
			INSERT INTO public.shop_messages (id, shop_id, user_id, message, created_at, updated_at, is_edited)
			SELECT gen_random_uuid()::text, $1, 'user-1', 'Vehicle 12 needs the PMCS follow-up before Thursday, parts are on order (' || g || ')',
			       timestamptz '2026-01-01 00:00:00+00' + g * interval '1 second',
			       timestamptz '2026-01-01 00:00:00+00' + g * interval '1 second', false
			FROM generate_series(1, $2) AS g`, shopID, planRowsPerShop)
		require.NoError(t, err)
	}
	_, err = testDB.Exec(`ANALYZE public.shop_messages`)
	require.NoError(t, err)

	var anchorID, anchorCreatedAt string
	require.NoError(t, testDB.QueryRow(`
		SELECT id, to_char(created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"') FROM public.shop_messages
		WHERE shop_id='plan-a' AND insertion_number = 5000`).Scan(&anchorID, &anchorCreatedAt))
	var idList []string
	idRows, err := testDB.Query(`SELECT quote_literal(id) FROM public.shop_messages WHERE shop_id='plan-a' AND insertion_number BETWEEN 100 AND 199`)
	require.NoError(t, err)
	for idRows.Next() {
		var quoted string
		require.NoError(t, idRows.Scan(&quoted))
		idList = append(idList, quoted)
	}
	require.NoError(t, idRows.Err())
	require.NoError(t, idRows.Close())
	require.Len(t, idList, 100)

	cases := []struct {
		name  string
		query string
		// Any one of these indexes satisfies the case (see the reconcile note below).
		anyOfIndexes []string
	}{
		{"initial (newest 51 of one shop)",
			planProjection + `m.shop_id = 'plan-a' ORDER BY m.created_at DESC,m.id DESC LIMIT 51`,
			[]string{"idx_shop_messages_shop_created_id"}},
		{"history (older than the middle anchor)",
			planProjection + fmt.Sprintf(`m.shop_id = 'plan-a' AND (m.created_at,m.id) < ('%s','%s') ORDER BY m.created_at DESC,m.id DESC LIMIT 51`, anchorCreatedAt, anchorID),
			[]string{"idx_shop_messages_shop_created_id"}},
		{"catch-up (a few new rows at the tail)",
			planProjection + `m.shop_id = 'plan-a' AND m.insertion_number > 9950 AND m.insertion_number <= 10000 ORDER BY m.insertion_number ASC LIMIT 101`,
			[]string{"shop_messages_shop_insertion_number_key"}},
		{"catch-up (first cycle from zero, full backlog)",
			planProjection + `m.shop_id = 'plan-a' AND m.insertion_number > 0 AND m.insertion_number <= 10000 ORDER BY m.insertion_number ASC LIMIT 101`,
			[]string{"shop_messages_shop_insertion_number_key"}},
		{"reconcile (100 ids)",
			planProjection + `m.shop_id = 'plan-a' AND m.id IN (` + strings.Join(idList, ",") + `) ORDER BY m.created_at DESC,m.id DESC`,
			// Measured: the planner switches between shop_messages_pkey (bitmap,
			// ~0.2 ms) and the pre-existing idx_shop_messages_shop_id (index scan
			// filtering by id, <1 ms at 10,000 rows in the shop) depending on shop
			// size and table statistics. Both are index-bounded; a Seq Scan is
			// the regression this test guards against.
			[]string{"shop_messages_pkey", "idx_shop_messages_shop_id"}},
	}
	for _, tc := range cases {
		plan := explainAnalyze(t, tc.query)
		t.Logf("PLAN %s\n%s", tc.name, shortenLongLines(plan))
		require.NotContains(t, plan, "Seq Scan on shop_messages", tc.name)
		usesExpectedIndex := false
		for _, index := range tc.anyOfIndexes {
			usesExpectedIndex = usesExpectedIndex || strings.Contains(plan, index)
		}
		require.True(t, usesExpectedIndex, "%s must use one of %v", tc.name, tc.anyOfIndexes)
	}
}

// Eight concurrent current-writer streams exercise the actual persisted Shop
// authority/allocator path. All errors and committed row counts are asserted.
func TestMessageWritePerformance(t *testing.T) {
	if os.Getenv("SHOP_MESSAGE_WRITE_PERF") != "1" {
		t.Skip("set SHOP_MESSAGE_WRITE_PERF=1 for write capacity measurements")
	}
	for _, shopCount := range []int{1, 8} {
		t.Run(fmt.Sprintf("shops_%d", shopCount), func(t *testing.T) {
			clearShopTables(t, testDB)
			t.Cleanup(func() { clearShopTables(t, testDB) })
			ensureUser(t, testDB, "write-perf-user")
			router, database, counter := newPerformanceRouter(t)
			shopIDs := make([]string, shopCount)
			for i := range shopIDs {
				shopIDs[i] = createShop(t, router, "write-perf-user", fmt.Sprintf("Write performance %d", i))
			}
			repository := messages.NewRepository(database, nil, &bootstrap.Env{BlobAccountName: "test-account"})
			user := &bootstrap.User{UserID: "write-perf-user"}
			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()
			body := strings.Repeat("x", 256)
			for _, shopID := range shopIDs {
				now := time.Now().UTC()
				edited := false
				_, err := repository.CreateShopMessage(ctx, user, model.ShopMessages{ID: uuid.NewString(), ShopID: shopID, UserID: user.UserID, Message: body, CreatedAt: &now, UpdatedAt: &now, IsEdited: &edited})
				require.NoError(t, err)
			}
			type outcome struct {
				duration time.Duration
				bytes    int
				err      error
			}
			const workers, writesPerWorker = 8, 25
			outcomes := make(chan outcome, workers*writesPerWorker)
			start := make(chan struct{})
			var workersDone sync.WaitGroup
			finish := measurePerformanceResources(t, database, counter)
			for worker := 0; worker < workers; worker++ {
				workersDone.Add(1)
				go func(worker int) {
					defer workersDone.Done()
					<-start
					for i := 0; i < writesPerWorker; i++ {
						now := time.Now().UTC()
						edited := false
						began := time.Now()
						message, err := repository.CreateShopMessage(ctx, user, model.ShopMessages{ID: uuid.NewString(), ShopID: shopIDs[worker%shopCount], UserID: user.UserID, Message: body, CreatedAt: &now, UpdatedAt: &now, IsEdited: &edited})
						duration := time.Since(began)
						var payload []byte
						if err == nil {
							payload, err = json.Marshal(message)
						}
						outcomes <- outcome{duration: duration, bytes: len(payload), err: err}
					}
				}(worker)
			}
			close(start)
			workersDone.Wait()
			close(outcomes)
			finish(fmt.Sprintf("current_message_writes_shops_%d_workers_8", shopCount))
			durations := make([]time.Duration, 0, workers*writesPerWorker)
			payloadBytes := 0
			for result := range outcomes {
				require.NoError(t, result.err)
				durations = append(durations, result.duration)
				payloadBytes += result.bytes
			}
			var count int
			require.NoError(t, testDB.QueryRow(`SELECT count(*) FROM shop_messages WHERE user_id=$1`, user.UserID).Scan(&count))
			require.Equal(t, workers*writesPerWorker+shopCount, count)
			t.Logf("current_message_writes shops=%d workers=%d writes=%d p50=%s p95=%s p99=%s request_message_bytes=%d response_bytes_total=%d", shopCount, workers, len(durations), percentileDuration(durations, .5), percentileDuration(durations, .95), percentileDuration(durations, .99), len(body), payloadBytes)
		})
	}
}
