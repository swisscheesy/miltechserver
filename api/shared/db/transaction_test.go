package db

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"miltechserver/internal/testsql"
)

func TestWithTx(t *testing.T) {
	t.Run("returns result after commit", func(t *testing.T) {
		behavior := &testsql.Behavior{}
		got, err := WithTx(testsql.Open(t, behavior), func(tx *sql.Tx) (string, error) { _, err := tx.Exec("UPDATE anything"); return "saved", err })
		require.NoError(t, err)
		require.Equal(t, "saved", got)
		require.Equal(t, int64(1), behavior.Commits.Load())
		require.Zero(t, behavior.Rollbacks.Load())
	})
	t.Run("begin failure skips callback", func(t *testing.T) {
		beginError := errors.New("begin failure")
		behavior := &testsql.Behavior{BeginErr: beginError}
		got, err := WithTx(testsql.Open(t, behavior), func(*sql.Tx) (string, error) { t.Fatal("callback must not run"); return "", nil })
		require.ErrorIs(t, err, beginError)
		require.Empty(t, got)
		require.Zero(t, behavior.Commits.Load())
	})
	t.Run("callback and rollback causes survive", func(t *testing.T) {
		callbackError := errors.New("callback failure")
		rollbackError := errors.New("rollback failure")
		behavior := &testsql.Behavior{RollbackErr: rollbackError}
		got, err := WithTx(testsql.Open(t, behavior), func(*sql.Tx) (string, error) { return "discarded", callbackError })
		require.ErrorIs(t, err, callbackError)
		require.ErrorIs(t, err, rollbackError)
		require.Empty(t, got)
		require.Equal(t, int64(1), behavior.Rollbacks.Load())
		require.Zero(t, behavior.Commits.Load())
	})
	t.Run("commit failure returns zero result", func(t *testing.T) {
		commitError := errors.New("commit failure")
		behavior := &testsql.Behavior{CommitErr: commitError}
		got, err := WithTx(testsql.Open(t, behavior), func(*sql.Tx) (string, error) { return "uncertain", nil })
		require.ErrorIs(t, err, commitError)
		require.Empty(t, got)
		require.Equal(t, int64(1), behavior.Begins.Load())
		require.Equal(t, int64(1), behavior.Commits.Load())
	})
}
