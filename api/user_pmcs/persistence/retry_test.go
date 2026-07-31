package persistence

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

func TestIsRetryableTransactionError(t *testing.T) {
	t.Parallel()

	require.True(t, IsRetryable(&pq.Error{Code: pq.ErrorCode("40P01")}))
	require.True(t, IsRetryable(&pq.Error{Code: pq.ErrorCode("40001")}))
	require.False(t, IsRetryable(&pq.Error{Code: pq.ErrorCode("23505")}))
	require.False(t, IsRetryable(errors.New("validation failed")))
}

func TestTreeWriteTransactionRetriesOnlyRetryableFailures(t *testing.T) {
	t.Parallel()

	t.Run("retryable callback error succeeds on the next attempt", func(t *testing.T) {
		database, state := openScriptedDatabase(t, nil)
		attempts := 0

		result, err := WithWriteTx(
			context.Background(),
			database,
			3,
			func(_ *sql.Tx) (string, error) {
				attempts++
				if attempts == 1 {
					return "", &pq.Error{Code: pq.ErrorCode("40P01")}
				}
				return "committed", nil
			},
		)

		require.NoError(t, err)
		require.Equal(t, "committed", result)
		require.Equal(t, 2, attempts)
		require.Equal(t, 2, state.beginCount())
		require.Equal(t, 1, state.rollbackCount())
		require.Equal(t, 1, state.commitCount())
		require.Equal(t, driver.IsolationLevel(sql.LevelReadCommitted), state.isolationLevel())
	})

	t.Run("constraint error is not retried", func(t *testing.T) {
		database, state := openScriptedDatabase(t, nil)
		attempts := 0

		_, err := WithWriteTx(
			context.Background(),
			database,
			3,
			func(_ *sql.Tx) (struct{}, error) {
				attempts++
				return struct{}{}, &pq.Error{Code: pq.ErrorCode("23505")}
			},
		)

		require.Error(t, err)
		require.Equal(t, 1, attempts)
		require.Equal(t, 1, state.beginCount())
		require.Equal(t, 1, state.rollbackCount())
		require.Zero(t, state.commitCount())
	})
}

func TestSerializableTreeWriteTransactionUsesSerializableIsolation(t *testing.T) {
	t.Parallel()
	database, state := openScriptedDatabase(t, nil)

	result, err := WithSerializableWriteTx(
		context.Background(),
		database,
		3,
		func(_ *sql.Tx) (string, error) {
			return "committed", nil
		},
	)

	require.NoError(t, err)
	require.Equal(t, "committed", result)
	require.Equal(
		t,
		driver.IsolationLevel(sql.LevelSerializable),
		state.isolationLevel(),
	)
}

func TestTreeWriteTransactionBoundsAttemptsAndDiscardsResults(t *testing.T) {
	t.Parallel()

	database, state := openScriptedDatabase(t, nil)
	attempts := 0

	result, err := WithWriteTx(
		context.Background(),
		database,
		3,
		func(_ *sql.Tx) (string, error) {
			attempts++
			return fmt.Sprintf("attempt-%d", attempts),
				&pq.Error{Code: pq.ErrorCode("40001")}
		},
	)

	require.Error(t, err)
	require.Empty(t, result)
	require.Equal(t, 3, attempts)
	require.Equal(t, 3, state.beginCount())
	require.Equal(t, 3, state.rollbackCount())
	require.Zero(t, state.commitCount())
}

func TestTreeWriteTransactionTreatsCommitErrorAsFailure(t *testing.T) {
	t.Parallel()

	commitFailure := &pq.Error{Code: pq.ErrorCode("40001")}
	database, state := openScriptedDatabase(t, []error{commitFailure, nil})
	attempts := 0

	result, err := WithWriteTx(
		context.Background(),
		database,
		3,
		func(_ *sql.Tx) (string, error) {
			attempts++
			return fmt.Sprintf("result-%d", attempts), nil
		},
	)

	require.NoError(t, err)
	require.Equal(t, "result-2", result)
	require.Equal(t, 2, attempts)
	require.Equal(t, 2, state.beginCount())
	require.Equal(t, 2, state.commitCount())
}

func TestTreeWriteTransactionObservesCanceledContextBeforeRetry(t *testing.T) {
	t.Parallel()

	database, state := openScriptedDatabase(t, nil)
	ctx, cancel := context.WithCancel(context.Background())
	attempts := 0

	_, err := WithWriteTx(ctx, database, 3, func(_ *sql.Tx) (struct{}, error) {
		attempts++
		cancel()
		return struct{}{}, &pq.Error{Code: pq.ErrorCode("40P01")}
	})

	require.ErrorIs(t, err, context.Canceled)
	require.Equal(t, 1, attempts)
	require.Equal(t, 1, state.rollbackCount())
}

func TestTreeWriteTransactionRejectsNonPositiveAttemptLimit(t *testing.T) {
	t.Parallel()

	database, state := openScriptedDatabase(t, nil)
	_, err := WithWriteTx(
		context.Background(),
		database,
		0,
		func(_ *sql.Tx) (struct{}, error) {
			t.Fatal("callback must not run")
			return struct{}{}, nil
		},
	)

	require.Error(t, err)
	require.Zero(t, state.beginCount())
}

func TestTreeWriteTransactionRollsBackBeforePropagatingCallbackPanic(
	t *testing.T,
) {
	t.Parallel()

	database, state := openScriptedDatabase(t, nil)
	require.PanicsWithValue(t, "callback panic", func() {
		_, _ = WithWriteTx(
			context.Background(),
			database,
			3,
			func(_ *sql.Tx) (struct{}, error) {
				panic("callback panic")
			},
		)
	})
	require.Equal(t, 1, state.beginCount())
	require.Equal(t, 1, state.rollbackCount())
	require.Zero(t, state.commitCount())
}

type scriptedDriver struct {
	state *scriptedDriverState
}

type scriptedDriverState struct {
	mu             sync.Mutex
	begins         int
	commits        int
	rollbacks      int
	isolation      driver.IsolationLevel
	commitFailures []error
}

func (state *scriptedDriverState) beginCount() int {
	state.mu.Lock()
	defer state.mu.Unlock()
	return state.begins
}

func (state *scriptedDriverState) commitCount() int {
	state.mu.Lock()
	defer state.mu.Unlock()
	return state.commits
}

func (state *scriptedDriverState) rollbackCount() int {
	state.mu.Lock()
	defer state.mu.Unlock()
	return state.rollbacks
}

func (state *scriptedDriverState) isolationLevel() driver.IsolationLevel {
	state.mu.Lock()
	defer state.mu.Unlock()
	return state.isolation
}

func (databaseDriver *scriptedDriver) Open(string) (driver.Conn, error) {
	return &scriptedConnection{state: databaseDriver.state}, nil
}

type scriptedConnection struct {
	state *scriptedDriverState
}

func (connection *scriptedConnection) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("not implemented")
}

func (connection *scriptedConnection) Close() error {
	return nil
}

func (connection *scriptedConnection) Begin() (driver.Tx, error) {
	return connection.BeginTx(context.Background(), driver.TxOptions{})
}

func (connection *scriptedConnection) BeginTx(
	_ context.Context,
	options driver.TxOptions,
) (driver.Tx, error) {
	connection.state.mu.Lock()
	defer connection.state.mu.Unlock()
	connection.state.begins++
	connection.state.isolation = options.Isolation
	attempt := connection.state.begins - 1
	var commitError error
	if attempt < len(connection.state.commitFailures) {
		commitError = connection.state.commitFailures[attempt]
	}
	return &scriptedTransaction{
		state:       connection.state,
		commitError: commitError,
	}, nil
}

type scriptedTransaction struct {
	state       *scriptedDriverState
	commitError error
}

func (transaction *scriptedTransaction) Commit() error {
	transaction.state.mu.Lock()
	defer transaction.state.mu.Unlock()
	transaction.state.commits++
	return transaction.commitError
}

func (transaction *scriptedTransaction) Rollback() error {
	transaction.state.mu.Lock()
	defer transaction.state.mu.Unlock()
	transaction.state.rollbacks++
	return nil
}

var scriptedDriverCounter atomic.Uint64

func openScriptedDatabase(
	t *testing.T,
	commitFailures []error,
) (*sql.DB, *scriptedDriverState) {
	t.Helper()

	state := &scriptedDriverState{commitFailures: commitFailures}
	driverName := fmt.Sprintf(
		"user-pmcs-scripted-%d",
		scriptedDriverCounter.Add(1),
	)
	sql.Register(driverName, &scriptedDriver{state: state})
	database, err := sql.Open(driverName, "")
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, database.Close())
	})
	return database, state
}

var _ driver.Driver = (*scriptedDriver)(nil)
var _ driver.Conn = (*scriptedConnection)(nil)
var _ driver.ConnBeginTx = (*scriptedConnection)(nil)
var _ driver.Tx = (*scriptedTransaction)(nil)
var _ io.Closer = (*scriptedConnection)(nil)
