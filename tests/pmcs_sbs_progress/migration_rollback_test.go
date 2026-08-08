package pmcs_sbs_progress_test

import (
	"context"
	"database/sql"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestInspectionSourceRollbackWaitsForConcurrentCustomInsertAndRefusesExplicitly(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	requirePmcsSbsTestDatabase(t, ctx, testDB)

	user := testUser("prl-" + uuid.NewString()[:8])
	ensureUser(t, testDB, user)
	shopID := createShopWithMember(t, testDB, user, "member")
	vehicleID := createShopVehicle(t, testDB, shopID, user, "rollback-lock")
	inspectionID := uuid.New()
	t.Cleanup(func() {
		cleanupContext, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		_, cleanupErr := testDB.ExecContext(cleanupContext, `DELETE FROM pmcs_sbs_inspections WHERE id = $1`, inspectionID)
		require.NoError(t, cleanupErr)
		_, cleanupErr = testDB.ExecContext(cleanupContext, `DELETE FROM shops WHERE id = $1`, shopID)
		require.NoError(t, cleanupErr)
		_, cleanupErr = testDB.ExecContext(cleanupContext, `DELETE FROM users WHERE uid = $1`, user.UserID)
		require.NoError(t, cleanupErr)
	})

	rollbackSQL, err := os.ReadFile("../../migrations/011_rollback_extend_pmcs_inspection_sources.sql")
	require.NoError(t, err)

	writerConnection, err := testDB.Conn(ctx)
	require.NoError(t, err)
	defer writerConnection.Close()
	rollbackConnection, err := testDB.Conn(ctx)
	require.NoError(t, err)
	defer rollbackConnection.Close()
	defer func() {
		_, _ = rollbackConnection.ExecContext(context.Background(), `ROLLBACK`)
	}()
	requirePmcsSbsTestDatabase(t, ctx, writerConnection)
	requirePmcsSbsTestDatabase(t, ctx, rollbackConnection)

	var writerPID int
	var rollbackPID int
	require.NoError(t, writerConnection.QueryRowContext(ctx, `SELECT pg_backend_pid()`).Scan(&writerPID))
	require.NoError(t, rollbackConnection.QueryRowContext(ctx, `SELECT pg_backend_pid()`).Scan(&rollbackPID))
	require.NotEqual(t, writerPID, rollbackPID)

	writerTransaction, err := writerConnection.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer writerTransaction.Rollback()
	_, err = writerTransaction.ExecContext(ctx, `
		INSERT INTO pmcs_sbs_inspections (
			id, equipment_id, source_type, guide_manual,
			custom_checklist_id, custom_revision_id,
			custom_revision_number, custom_checklist_name,
			performed_date, performed_by
		) VALUES ($1, $2, 'custom', NULL, $3, $4, 0, 'Rollback Lock Fixture', $5, $6)`,
		inspectionID,
		vehicleID,
		uuid.New(),
		uuid.New(),
		time.Now().UTC(),
		user.UserID,
	)
	require.NoError(t, err)

	rollbackResult := make(chan error, 1)
	go func() {
		_, rollbackErr := rollbackConnection.ExecContext(ctx, string(rollbackSQL))
		rollbackResult <- rollbackErr
	}()

	requireConnectionWaitingForTableLock(t, ctx, testDB, rollbackPID)
	require.NoError(t, writerTransaction.Commit())

	rollbackErr := <-rollbackResult
	_, cleanupErr := rollbackConnection.ExecContext(ctx, `ROLLBACK`)
	require.NoError(t, cleanupErr)
	require.Error(t, rollbackErr)
	require.Contains(t, rollbackErr.Error(), "cannot roll back PMCS inspection source union while custom inspections exist")
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

func requireConnectionWaitingForTableLock(t *testing.T, ctx context.Context, database *sql.DB, backendPID int) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		var waitEventType sql.NullString
		err := database.QueryRowContext(ctx, `
			SELECT wait_event_type
			FROM pg_stat_activity
			WHERE pid = $1`, backendPID).Scan(&waitEventType)
		require.NoError(t, err)
		if waitEventType.Valid && strings.EqualFold(waitEventType.String, "Lock") {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("rollback connection did not wait for the inspection table lock")
}
