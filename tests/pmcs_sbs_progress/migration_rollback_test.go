package pmcs_sbs_progress_test

import (
	"context"
	"database/sql"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestTableRenameRollbackTimesOutAtomically(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	requirePmcsSbsTestDatabase(t, ctx, testDB)

	rollbackSQL, err := os.ReadFile("../../migrations/014_rollback_rename_pmcs_tables.sql")
	require.NoError(t, err)

	holderConnection, err := testDB.Conn(ctx)
	require.NoError(t, err)
	defer holderConnection.Close()
	rollbackConnection, err := testDB.Conn(ctx)
	require.NoError(t, err)
	defer rollbackConnection.Close()
	defer func() {
		_, _ = rollbackConnection.ExecContext(context.Background(), `ROLLBACK`)
	}()
	requirePmcsSbsTestDatabase(t, ctx, holderConnection)
	requirePmcsSbsTestDatabase(t, ctx, rollbackConnection)

	holderTransaction, err := holderConnection.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer holderTransaction.Rollback()
	_, err = holderTransaction.ExecContext(ctx, `
		LOCK TABLE public.user_pmcs_inspections IN ACCESS SHARE MODE`)
	require.NoError(t, err)

	rollbackStarted := time.Now()
	_, rollbackErr := rollbackConnection.ExecContext(ctx, string(rollbackSQL))
	require.Error(t, rollbackErr)
	require.Contains(t, strings.ToLower(rollbackErr.Error()), "lock timeout")
	require.Less(t, time.Since(rollbackStarted), 5*time.Second)
	_, cleanupErr := rollbackConnection.ExecContext(ctx, `ROLLBACK`)
	require.NoError(t, cleanupErr)

	requireTargetRelations(t, ctx, testDB)
}

type databaseQueryer interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func requirePmcsSbsTestDatabase(t *testing.T, ctx context.Context, database databaseQueryer) {
	t.Helper()
	var databaseName string
	require.NoError(t, database.QueryRowContext(ctx, `SELECT current_database()`).Scan(&databaseName))
	require.Equal(t, "miltech_ng_test", databaseName)
}
