package shops_test

import (
	"context"
	"database/sql"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// waitForLockWait polls until some backend is blocked on a heavyweight/row lock,
// so ordering assertions never depend on wall-clock sleeps.
func waitForLockWait(t *testing.T) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		var waiting int
		require.NoError(t, testDB.QueryRow(
			`SELECT count(*) FROM pg_stat_activity WHERE datname = current_database() AND wait_event_type = 'Lock'`).Scan(&waiting))
		if waiting > 0 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("no backend ever waited on a lock")
}

// insertMessageInTx returns an error instead of failing the test so it is safe
// to call from goroutines (FailNow must only run on the test goroutine).
func insertMessageInTx(tx *sql.Tx, shopID string) (int64, error) {
	var n int64
	err := tx.QueryRow(
		`INSERT INTO public.shop_messages (id, shop_id, user_id, message, created_at)
		 VALUES ($1, $2, 'user-1', 'concurrent', now()) RETURNING insertion_number`,
		uuid.NewString(), shopID).Scan(&n)
	return n, err
}

// insertAndCommitInNewTx runs a full begin/insert/commit cycle and is safe to
// call from goroutines.
func insertAndCommitInNewTx(ctx context.Context, shopID string) (int64, error) {
	tx, err := testDB.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	n, err := insertMessageInTx(tx, shopID)
	if err != nil {
		_ = tx.Rollback()
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return n, nil
}

// txInsert is for the test goroutine only; it fails the test on error.
func txInsert(t *testing.T, tx *sql.Tx, shopID string) int64 {
	t.Helper()
	n, err := insertMessageInTx(tx, shopID)
	require.NoError(t, err)
	return n
}

func seedShop(t *testing.T, name string) string {
	t.Helper()
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "user-1")
	return createShop(t, newTestRouter(t), "user-1", name)
}

type insertOutcome struct {
	number int64
	err    error
}

func TestMessageSyncConcurrencySecondWriterWaitsForFirstCommit(t *testing.T) {
	shopID := seedShop(t, "Barrier")
	ctx := context.Background()

	txA, err := testDB.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer func() { _ = txA.Rollback() }() // no-op after Commit; unblocks writer B if an assertion fails
	require.Equal(t, int64(1), txInsert(t, txA, shopID))

	result := make(chan insertOutcome, 1)
	go func() {
		n, err := insertAndCommitInNewTx(ctx, shopID)
		result <- insertOutcome{number: n, err: err}
	}()

	waitForLockWait(t)
	select {
	case outcome := <-result:
		t.Fatalf("second writer finished (%d, err=%v) before the first committed", outcome.number, outcome.err)
	default:
	}
	require.NoError(t, txA.Commit())
	select {
	case outcome := <-result:
		require.NoError(t, outcome.err)
		require.Equal(t, int64(2), outcome.number)
	case <-time.After(10 * time.Second):
		t.Fatal("second writer never completed after first commit")
	}
}

func TestMessageSyncConcurrencyRollbackLeavesNoGap(t *testing.T) {
	shopID := seedShop(t, "Rollback")
	ctx := context.Background()
	committed, err := testDB.BeginTx(ctx, nil)
	require.NoError(t, err)
	require.Equal(t, int64(1), txInsert(t, committed, shopID))
	require.NoError(t, committed.Commit())

	aborted, err := testDB.BeginTx(ctx, nil)
	require.NoError(t, err)
	require.Equal(t, int64(2), txInsert(t, aborted, shopID))
	require.NoError(t, aborted.Rollback())

	var last int64
	require.NoError(t, testDB.QueryRow(`SELECT last_number FROM public.shop_message_counters WHERE shop_id=$1`, shopID).Scan(&last))
	require.Equal(t, int64(1), last, "rolled-back increment must not persist")

	next, err := testDB.BeginTx(ctx, nil)
	require.NoError(t, err)
	require.Equal(t, int64(2), txInsert(t, next, shopID))
	require.NoError(t, next.Commit())
}

func TestMessageSyncConcurrencyDifferentShopsDoNotBlock(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "user-1")
	router := newTestRouter(t)
	shopA := createShop(t, router, "user-1", "A")
	shopB := createShop(t, router, "user-1", "B")
	ctx := context.Background()

	txA, err := testDB.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer func() { _ = txA.Rollback() }()
	txInsert(t, txA, shopA)

	done := make(chan insertOutcome, 1)
	go func() {
		n, err := insertAndCommitInNewTx(ctx, shopB)
		done <- insertOutcome{number: n, err: err}
	}()
	select {
	case outcome := <-done:
		require.NoError(t, outcome.err)
		require.Equal(t, int64(1), outcome.number)
	case <-time.After(10 * time.Second):
		t.Fatal("insert into another shop was blocked by an open transaction")
	}
}

func TestMessageSyncConcurrencyFiftyWritersProduceContiguousNumbers(t *testing.T) {
	shopID := seedShop(t, "Burst")
	const writers = 50
	outcomes := make([]insertOutcome, writers)
	var wg sync.WaitGroup
	for i := 0; i < writers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			n, err := insertAndCommitInNewTx(context.Background(), shopID)
			outcomes[i] = insertOutcome{number: n, err: err}
		}(i)
	}
	wg.Wait()

	numbers := make([]int64, 0, writers)
	for _, outcome := range outcomes {
		require.NoError(t, outcome.err)
		numbers = append(numbers, outcome.number)
	}
	sort.Slice(numbers, func(a, b int) bool { return numbers[a] < numbers[b] })
	for i, n := range numbers {
		require.Equal(t, int64(i+1), n)
	}
}
