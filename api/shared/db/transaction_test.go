package db

import (
	"database/sql"
	"errors"
	"testing"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

const testDSN = "postgres://postgres:potato123@192.168.20.70/miltech_ng_test?sslmode=disable"

func TestWithTx_CommitsOnSuccess(t *testing.T) {
	conn, err := sql.Open("postgres", testDSN)
	require.NoError(t, err)
	defer conn.Close()

	result, err := WithTx(conn, func(tx *sql.Tx) (int, error) {
		return 42, nil
	})

	require.NoError(t, err)
	require.Equal(t, 42, result)
}

func TestWithTx_RollsBackOnError(t *testing.T) {
	conn, err := sql.Open("postgres", testDSN)
	require.NoError(t, err)
	defer conn.Close()

	// Create a temp table, attempt a write inside a callback that then
	// errors, and confirm the write did not persist.
	_, err = conn.Exec("CREATE TEMP TABLE IF NOT EXISTS tx_rollback_probe (id INT)")
	require.NoError(t, err)

	wantErr := errors.New("forced failure")
	_, err = WithTx(conn, func(tx *sql.Tx) (int, error) {
		_, execErr := tx.Exec("INSERT INTO tx_rollback_probe (id) VALUES (1)")
		require.NoError(t, execErr)
		return 0, wantErr
	})
	require.ErrorIs(t, err, wantErr)

	var count int
	require.NoError(t, conn.QueryRow("SELECT COUNT(*) FROM tx_rollback_probe").Scan(&count))
	require.Equal(t, 0, count, "rollback should have discarded the insert")
}
