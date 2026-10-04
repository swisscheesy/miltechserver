package testsql_test

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"miltechserver/internal/testsql"
)

func TestDriverRowsAndExecution(t *testing.T) {
	behavior := &testsql.Behavior{Columns: []string{"id", "name"}, Rows: [][]driver.Value{{int64(7), "first"}, {int64(8), nil}}}
	db := testsql.Open(t, behavior)
	rows, err := db.QueryContext(context.Background(), "SELECT id, name")
	require.NoError(t, err)
	defer rows.Close()
	columns, err := rows.Columns()
	require.NoError(t, err)
	require.Equal(t, []string{"id", "name"}, columns)
	var got [][]any
	for rows.Next() {
		var id int64
		var name sql.NullString
		require.NoError(t, rows.Scan(&id, &name))
		got = append(got, []any{id, name})
	}
	require.NoError(t, rows.Err())
	require.Equal(t, [][]any{{int64(7), sql.NullString{String: "first", Valid: true}}, {int64(8), sql.NullString{}}}, got)
	result, err := db.ExecContext(context.Background(), "UPDATE anything")
	require.NoError(t, err)
	affected, err := result.RowsAffected()
	require.NoError(t, err)
	require.Equal(t, int64(1), affected)
	require.Equal(t, int64(1), behavior.Queries.Load())
	require.Equal(t, int64(1), behavior.Execs.Load())
}

func TestDriverConfiguredErrors(t *testing.T) {
	sentinel := errors.New("configured failure")
	t.Run("begin", func(t *testing.T) {
		behavior := &testsql.Behavior{BeginErr: sentinel}
		db := testsql.Open(t, behavior)
		_, err := db.BeginTx(context.Background(), nil)
		require.ErrorIs(t, err, sentinel)
		require.Equal(t, int64(1), behavior.Begins.Load())
	})
	t.Run("query", func(t *testing.T) {
		behavior := &testsql.Behavior{QueryErr: sentinel}
		db := testsql.Open(t, behavior)
		_, err := db.QueryContext(context.Background(), "SELECT anything")
		require.ErrorIs(t, err, sentinel)
	})
	t.Run("exec", func(t *testing.T) {
		behavior := &testsql.Behavior{ExecErr: sentinel}
		db := testsql.Open(t, behavior)
		_, err := db.ExecContext(context.Background(), "UPDATE anything")
		require.ErrorIs(t, err, sentinel)
	})
	t.Run("commit", func(t *testing.T) {
		behavior := &testsql.Behavior{CommitErr: sentinel}
		db := testsql.Open(t, behavior)
		tx, err := db.Begin()
		require.NoError(t, err)
		require.ErrorIs(t, tx.Commit(), sentinel)
		require.Equal(t, int64(1), behavior.Commits.Load())
	})
	t.Run("rollback", func(t *testing.T) {
		behavior := &testsql.Behavior{RollbackErr: sentinel}
		db := testsql.Open(t, behavior)
		tx, err := db.Begin()
		require.NoError(t, err)
		require.ErrorIs(t, tx.Rollback(), sentinel)
		require.Equal(t, int64(1), behavior.Rollbacks.Load())
	})
}

func TestDriverPreparedStatements(t *testing.T) {
	db := testsql.Open(t, &testsql.Behavior{Columns: []string{"id"}, Rows: [][]driver.Value{{int64(9)}}})
	stmt, err := db.Prepare("SELECT id WHERE id = ?")
	require.NoError(t, err)
	defer stmt.Close()
	var id int64
	require.NoError(t, stmt.QueryRowContext(context.Background(), 9).Scan(&id))
	require.Equal(t, int64(9), id)
	_, err = stmt.ExecContext(context.Background(), 9)
	require.NoError(t, err)
}

func TestDriverCancellation(t *testing.T) {
	behavior := &testsql.Behavior{}
	db := testsql.Open(t, behavior)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := db.ExecContext(ctx, "UPDATE anything")
	require.ErrorIs(t, err, context.Canceled)
	_, err = db.QueryContext(ctx, "SELECT anything")
	require.ErrorIs(t, err, context.Canceled)
	_, err = db.BeginTx(ctx, nil)
	require.ErrorIs(t, err, context.Canceled)
	require.Zero(t, behavior.Execs.Load())
	require.Zero(t, behavior.Queries.Load())
	require.Zero(t, behavior.Begins.Load())
}
