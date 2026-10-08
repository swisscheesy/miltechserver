package shops_test

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

// Removing the Shop-before-counter bridge makes these two physical sessions
// deadlock: the legacy insert holds the counter while waiting on the Shop FK.
func TestMessageAllocatorMixedWriters(t *testing.T) {
	shopID := seedShop(t, "Mixed allocator")
	_, err := insertAndCommitInNewTx(context.Background(), shopID)
	require.NoError(t, err) // Exercise an existing counter, as on a populated target.
	current, currentPID := allocatorSession(t)
	legacy, legacyPID := allocatorSession(t)
	require.NotEqual(t, currentPID, legacyPID)
	_, err = current.Exec(`SELECT id FROM public.shops WHERE id=$1 FOR UPDATE`, shopID)
	require.NoError(t, err)
	legacyResult := make(chan insertOutcome, 1)
	go func() {
		n, err := insertMessageInTx(legacy, shopID)
		if err == nil {
			err = legacy.Commit()
		} else {
			_ = legacy.Rollback()
		}
		legacyResult <- insertOutcome{number: n, err: err}
	}()
	waitForSessionBlock(t, legacyPID, currentPID)
	firstCommittedNumber, currentWriterError := insertMessageInTx(current, shopID)
	if currentWriterError == nil {
		currentWriterError = current.Commit()
	} else {
		_ = current.Rollback()
	}
	outcome := <-legacyResult
	logAllocatorError(t, currentWriterError)
	logAllocatorError(t, outcome.err)
	require.NoError(t, currentWriterError)
	require.NoError(t, outcome.err)
	require.Less(t, firstCommittedNumber, outcome.number)
	require.Equal(t, int64(2), firstCommittedNumber)
	require.Equal(t, int64(3), outcome.number)
}

func TestMessageAllocatorLegacyFirst(t *testing.T) {
	shopID := seedShop(t, "Legacy first")
	legacy, legacyPID := allocatorSession(t)
	current, currentPID := allocatorSession(t)
	first := txInsert(t, legacy, shopID)
	result := make(chan insertOutcome, 1)
	go func() {
		_, err := current.Exec(`SELECT id FROM public.shops WHERE id=$1 FOR UPDATE`, shopID)
		var number int64
		if err == nil {
			number, err = insertMessageInTx(current, shopID)
		}
		if err == nil {
			err = current.Commit()
		} else {
			_ = current.Rollback()
		}
		result <- insertOutcome{number: number, err: err}
	}()
	waitForSessionBlock(t, currentPID, legacyPID)
	require.NoError(t, legacy.Commit())
	outcome := <-result
	require.NoError(t, outcome.err)
	require.Equal(t, int64(1), first)
	require.Equal(t, int64(2), outcome.number)
}

func allocatorSession(t *testing.T) (*sql.Tx, int) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	t.Cleanup(cancel)
	conn, err := testDB.Conn(ctx)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, conn.Close()) })
	tx, err := conn.BeginTx(ctx, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = tx.Rollback() })
	// Bound an in-flight query even if a failing barrier leaves its blocker
	// open while test cleanup unwinds reserved sessions in reverse order.
	_, err = tx.Exec(`SET LOCAL statement_timeout = '15s'`)
	require.NoError(t, err)
	var pid int
	require.NoError(t, tx.QueryRow(`SELECT pg_backend_pid()`).Scan(&pid))
	return tx, pid
}

func waitForSessionBlock(t *testing.T, waitingPID, blockingPID int) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		var waiting bool
		require.NoError(t, testDB.QueryRow(`SELECT $2 = ANY(pg_blocking_pids($1)) AND EXISTS (
            SELECT 1 FROM pg_stat_activity WHERE pid=$1 AND wait_event_type='Lock')`, waitingPID, blockingPID).Scan(&waiting))
		if waiting {
			t.Logf("physical lock wait: backend %d blocked by backend %d", waitingPID, blockingPID)
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("expected physical session %d to wait on %d", waitingPID, blockingPID)
}

func logAllocatorError(t *testing.T, err error) {
	t.Helper()
	if pgError, ok := err.(*pq.Error); ok {
		t.Logf("allocator SQLSTATE=%s message=%s detail=%s", pgError.Code, pgError.Message, pgError.Detail)
	}
}

// A rolled-back current writer must release the Shop and counter together;
// the waiting legacy writer reuses the aborted number.
func TestMessageAllocatorMixedRollback(t *testing.T) {
	shopID := seedShop(t, "Mixed rollback")
	current, currentPID := allocatorSession(t)
	legacy, legacyPID := allocatorSession(t)
	_, err := current.Exec(`SELECT id FROM public.shops WHERE id=$1 FOR UPDATE`, shopID)
	require.NoError(t, err)
	require.Equal(t, int64(1), txInsert(t, current, shopID))
	result := make(chan insertOutcome, 1)
	go func() {
		n, err := insertMessageInTx(legacy, shopID)
		if err == nil {
			err = legacy.Commit()
		} else {
			_ = legacy.Rollback()
		}
		result <- insertOutcome{number: n, err: err}
	}()
	waitForSessionBlock(t, legacyPID, currentPID)
	require.NoError(t, current.Rollback())
	outcome := <-result
	require.NoError(t, outcome.err)
	require.Equal(t, int64(1), outcome.number)
	var count, last int
	require.NoError(t, testDB.QueryRow(`SELECT count(*) FROM public.shop_messages WHERE shop_id=$1`, shopID).Scan(&count))
	require.Equal(t, 1, count)
	require.NoError(t, testDB.QueryRow(`SELECT last_number FROM public.shop_message_counters WHERE shop_id=$1`, shopID).Scan(&last))
	require.Equal(t, 1, last)
}

func TestMessageAllocatorDifferentShops(t *testing.T) {
	shopA := seedShop(t, "Current writer")
	shopB := createShop(t, newTestRouter(t), "user-1", "Legacy writer")
	current, _ := allocatorSession(t)
	legacy, _ := allocatorSession(t)
	_, err := current.Exec(`SELECT id FROM public.shops WHERE id=$1 FOR UPDATE`, shopA)
	require.NoError(t, err)
	require.Equal(t, int64(1), txInsert(t, current, shopA))
	require.Equal(t, int64(1), txInsert(t, legacy, shopB))
	require.NoError(t, legacy.Commit())
	require.NoError(t, current.Commit())
}

func TestMessageAllocatorDeletion(t *testing.T) {
	t.Run("delete waits for legacy commit then cascades", func(t *testing.T) {
		shopID := seedShop(t, "Delete waits")
		legacy, legacyPID := allocatorSession(t)
		deleter, deletePID := allocatorSession(t)
		require.Equal(t, int64(1), txInsert(t, legacy, shopID))
		result := make(chan error, 1)
		go func() {
			_, err := deleter.Exec(`DELETE FROM public.shops WHERE id=$1`, shopID)
			if err == nil {
				err = deleter.Commit()
			} else {
				_ = deleter.Rollback()
			}
			result <- err
		}()
		waitForSessionBlock(t, deletePID, legacyPID)
		require.NoError(t, legacy.Commit())
		require.NoError(t, <-result)
		requireAllocatorShopEmpty(t, shopID)
	})
	t.Run("legacy waits for deletion then refuses missing parent", func(t *testing.T) {
		shopID := seedShop(t, "Insert waits")
		deleter, deletePID := allocatorSession(t)
		legacy, legacyPID := allocatorSession(t)
		_, err := deleter.Exec(`DELETE FROM public.shops WHERE id=$1`, shopID)
		require.NoError(t, err)
		result := make(chan error, 1)
		go func() {
			_, err := insertMessageInTx(legacy, shopID)
			_ = legacy.Rollback()
			result <- err
		}()
		waitForSessionBlock(t, legacyPID, deletePID)
		require.NoError(t, deleter.Commit())
		var pgError *pq.Error
		require.ErrorAs(t, <-result, &pgError)
		require.Equal(t, pq.ErrorCode("23503"), pgError.Code)
		requireAllocatorShopEmpty(t, shopID)
	})
}

func requireAllocatorShopEmpty(t *testing.T, shopID string) {
	t.Helper()
	var count int
	require.NoError(t, testDB.QueryRow(`SELECT
        (SELECT count(*) FROM public.shop_messages WHERE shop_id=$1) +
        (SELECT count(*) FROM public.shop_message_counters WHERE shop_id=$1)`, shopID).Scan(&count))
	require.Zero(t, count)
}

// Each omitted privilege must cause a real 42501 under the effective role,
// rather than relying on any-of has_table_privilege checks or owner execution.
func TestMessageAllocatorRestrictedRoles(t *testing.T) {
	cases := []struct {
		name                                                              string
		shopSelect, shopLock, counterSelect, counterInsert, counterUpdate bool
		wantTable                                                         string
	}{
		{name: "no Shop SELECT", shopLock: true, counterSelect: true, counterInsert: true, counterUpdate: true, wantTable: "shops"},
		{name: "Shop SELECT only", shopSelect: true, counterSelect: true, counterInsert: true, counterUpdate: true, wantTable: "shops"},
		{name: "counter SELECT only", shopSelect: true, shopLock: true, counterSelect: true, wantTable: "shop_message_counters"},
		{name: "counter INSERT only", shopSelect: true, shopLock: true, counterInsert: true, wantTable: "shop_message_counters"},
		{name: "counter UPDATE only", shopSelect: true, shopLock: true, counterUpdate: true, wantTable: "shop_message_counters"},
		{name: "no counter SELECT", shopSelect: true, shopLock: true, counterInsert: true, counterUpdate: true, wantTable: "shop_message_counters"},
		{name: "no counter INSERT", shopSelect: true, shopLock: true, counterSelect: true, counterUpdate: true, wantTable: "shop_message_counters"},
		{name: "no counter UPDATE", shopSelect: true, shopLock: true, counterSelect: true, counterInsert: true, wantTable: "shop_message_counters"},
		{name: "minimal authorized", shopSelect: true, shopLock: true, counterSelect: true, counterInsert: true, counterUpdate: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			shopID := seedShop(t, tc.name)
			role := "allocator_" + uuid.NewString()
			quoted := pq.QuoteIdentifier(role)
			_, err := testDB.Exec(`CREATE ROLE ` + quoted + ` NOLOGIN NOSUPERUSER NOBYPASSRLS NOINHERIT`)
			require.NoError(t, err)
			t.Cleanup(func() {
				_, err := testDB.Exec(`DROP OWNED BY ` + quoted + `; DROP ROLE ` + quoted)
				require.NoError(t, err)
			})
			grants := []string{
				`USAGE ON SCHEMA public`,
				`INSERT (id, shop_id, user_id, message, created_at), SELECT (insertion_number) ON public.shop_messages`,
			}
			if tc.shopSelect {
				grants = append(grants, `SELECT (id) ON public.shops`)
			}
			if tc.shopLock {
				grants = append(grants, `UPDATE (name) ON public.shops`)
			}
			if tc.counterSelect {
				grants = append(grants, `SELECT (shop_id, last_number) ON public.shop_message_counters`)
			}
			if tc.counterInsert {
				grants = append(grants, `INSERT (shop_id, last_number) ON public.shop_message_counters`)
			}
			if tc.counterUpdate {
				grants = append(grants, `UPDATE (last_number) ON public.shop_message_counters`)
			}
			for _, grant := range grants {
				_, err = testDB.Exec(`GRANT ` + grant + ` TO ` + quoted)
				require.NoError(t, err)
			}
			tx, pid := allocatorSession(t)
			_, err = tx.Exec(`SET LOCAL ROLE ` + quoted)
			require.NoError(t, err)
			var actualRole string
			var isSuper bool
			require.NoError(t, tx.QueryRow(`SELECT current_user, rolsuper FROM pg_roles WHERE rolname=current_user`).Scan(&actualRole, &isSuper))
			require.Equal(t, role, actualRole)
			require.False(t, isSuper)
			number, insertErr := insertMessageInTx(tx, shopID)
			if tc.wantTable != "" {
				var pgError *pq.Error
				require.ErrorAs(t, insertErr, &pgError)
				require.Equal(t, pq.ErrorCode("42501"), pgError.Code)
				require.Equal(t, fmt.Sprintf("permission denied for table %s", tc.wantTable), pgError.Message)
				t.Logf("backend %d effective_role=%s SQLSTATE=%s %s", pid, actualRole, pgError.Code, pgError.Message)
				require.NoError(t, tx.Rollback())
				requireAllocatorShopEmpty(t, shopID)
			} else {
				require.NoError(t, insertErr)
				require.Equal(t, int64(1), number)
				require.Equal(t, int64(2), txInsert(t, tx, shopID))
				require.NoError(t, tx.Commit())
				t.Logf("backend %d effective_role=%s inserted fresh and existing counter paths with minimal column grants", pid, actualRole)
			}
		})
	}
}
