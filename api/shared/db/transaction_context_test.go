package db

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"miltechserver/internal/testsql"
)

func TestWithTxContextCallbackAndRollbackErrors(t *testing.T) {
	callbackError := errors.New("callback failure")
	rollbackError := errors.New("rollback failure")
	behavior := &testsql.Behavior{RollbackErr: rollbackError}
	joined := WithTxContext(context.Background(), testsql.Open(t, behavior), func(*sql.Tx) error { return callbackError })
	require.ErrorIs(t, joined, callbackError)
	require.ErrorIs(t, joined, rollbackError)
	require.Zero(t, behavior.Commits.Load())
}

func TestWithTxContextReadCommitted(t *testing.T) {
	behavior := &testsql.Behavior{}
	err := WithTxContext(context.Background(), testsql.Open(t, behavior), func(*sql.Tx) error { return nil })
	require.NoError(t, err)
	require.Equal(t, int64(sql.LevelReadCommitted), behavior.BeginIsolation.Load())
	require.False(t, behavior.BeginReadOnly.Load())
	require.Equal(t, int64(1), behavior.Commits.Load())
}

func TestWithTxOptionsIsolationAndReadOnly(t *testing.T) {
	behavior := &testsql.Behavior{}
	opts := &sql.TxOptions{Isolation: sql.LevelSerializable, ReadOnly: true}
	err := WithTxOptions(context.Background(), testsql.Open(t, behavior), opts, func(*sql.Tx) error { return nil })
	require.NoError(t, err)
	require.Equal(t, int64(sql.LevelSerializable), behavior.BeginIsolation.Load())
	require.True(t, behavior.BeginReadOnly.Load())
	require.Equal(t, int64(1), behavior.Commits.Load())
	require.Zero(t, behavior.Rollbacks.Load())
}

func TestWithTxOptionsCancellation(t *testing.T) {
	t.Run("before begin", func(t *testing.T) {
		behavior := &testsql.Behavior{}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		err := WithTxOptions(ctx, testsql.Open(t, behavior), nil, func(*sql.Tx) error { t.Fatal("callback must not run"); return nil })
		require.ErrorIs(t, err, context.Canceled)
		require.Zero(t, behavior.Begins.Load())
		require.Zero(t, behavior.Commits.Load())
	})
	t.Run("during callback", func(t *testing.T) {
		behavior := &testsql.Behavior{}
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		err := WithTxOptions(ctx, testsql.Open(t, behavior), nil, func(tx *sql.Tx) error { cancel(); return nil })
		require.ErrorIs(t, err, context.Canceled)
		require.Zero(t, behavior.Commits.Load())
		require.Eventually(t, func() bool { return behavior.Rollbacks.Load() == 1 }, time.Second, time.Millisecond)
	})
}

func TestWithTxOptionsPoolWaitCancellation(t *testing.T) {
	behavior := &testsql.Behavior{}
	db := testsql.Open(t, behavior)
	db.SetMaxOpenConns(1)
	occupied, err := db.Conn(context.Background())
	require.NoError(t, err)
	defer occupied.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	result := make(chan error, 1)
	callbackRan := make(chan struct{}, 1)
	go func() {
		result <- WithTxOptions(ctx, db, nil, func(*sql.Tx) error { callbackRan <- struct{}{}; return nil })
	}()
	require.Eventually(t, func() bool { return db.Stats().WaitCount == 1 }, time.Second, time.Millisecond)
	cancel()
	select {
	case err := <-result:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(time.Second):
		t.Fatal("transaction did not stop waiting for a connection")
	}
	require.Empty(t, callbackRan)
	require.Zero(t, behavior.Begins.Load())
	require.Zero(t, behavior.Commits.Load())
}

func TestWithTxOptionsCallbackAndRollbackErrors(t *testing.T) {
	callbackError := errors.New("callback failure")
	rollbackError := errors.New("rollback failure")
	behavior := &testsql.Behavior{RollbackErr: rollbackError}
	joined := WithTxOptions(context.Background(), testsql.Open(t, behavior), nil, func(*sql.Tx) error { return callbackError })
	require.ErrorIs(t, joined, callbackError)
	require.ErrorIs(t, joined, rollbackError)
	require.Equal(t, int64(1), behavior.Rollbacks.Load())
	require.Zero(t, behavior.Commits.Load())
}

func TestWithTxOptionsCommitError(t *testing.T) {
	commitError := errors.New("commit failure")
	behavior := &testsql.Behavior{CommitErr: commitError}
	callbacks := 0
	err := WithTxOptions(context.Background(), testsql.Open(t, behavior), nil, func(*sql.Tx) error { callbacks++; return nil })
	require.ErrorIs(t, err, commitError)
	require.Equal(t, 1, callbacks)
	require.Equal(t, int64(1), behavior.Begins.Load())
	require.Equal(t, int64(1), behavior.Commits.Load())
}

func TestWithTxOptionsBeginError(t *testing.T) {
	beginError := errors.New("begin failure")
	behavior := &testsql.Behavior{BeginErr: beginError}
	err := WithTxOptions(context.Background(), testsql.Open(t, behavior), nil, func(*sql.Tx) error { t.Fatal("callback must not run"); return nil })
	require.ErrorIs(t, err, beginError)
	require.Zero(t, behavior.Commits.Load())
	require.Zero(t, behavior.Rollbacks.Load())
}

func TestWithTxOptionsPanicRollsBack(t *testing.T) {
	behavior := &testsql.Behavior{}
	require.PanicsWithValue(t, "callback panic", func() {
		_ = WithTxOptions(context.Background(), testsql.Open(t, behavior), nil, func(*sql.Tx) error { panic("callback panic") })
	})
	require.Equal(t, int64(1), behavior.Rollbacks.Load())
	require.Zero(t, behavior.Commits.Load())
}
