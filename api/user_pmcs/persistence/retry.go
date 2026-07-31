package persistence

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/lib/pq"

	"miltechserver/api/user_pmcs/shared"
)

type TxFunc[T any] func(*sql.Tx) (T, error)

func WithWriteTx[T any](
	ctx context.Context,
	database *sql.DB,
	maxAttempts int,
	fn TxFunc[T],
) (T, error) {
	return withWriteTx(
		ctx,
		database,
		maxAttempts,
		sql.LevelReadCommitted,
		fn,
	)
}

func WithSerializableWriteTx[T any](
	ctx context.Context,
	database *sql.DB,
	maxAttempts int,
	fn TxFunc[T],
) (T, error) {
	// Serializable retries let callers protect read-then-write invariants
	// without introducing one process-wide blocking lock.
	return withWriteTx(
		ctx,
		database,
		maxAttempts,
		sql.LevelSerializable,
		fn,
	)
}

func withWriteTx[T any](
	ctx context.Context,
	database *sql.DB,
	maxAttempts int,
	isolation sql.IsolationLevel,
	fn TxFunc[T],
) (T, error) {
	var zero T
	if database == nil {
		return zero, errors.New("write transaction database is nil")
	}
	if fn == nil {
		return zero, errors.New("write transaction callback is nil")
	}
	if maxAttempts <= 0 {
		return zero, fmt.Errorf(
			"write transaction max attempts must be positive: %d",
			maxAttempts,
		)
	}

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		startedAt := time.Now()
		result, err := runWriteAttempt(ctx, database, isolation, fn)
		shared.RecordDBDuration(ctx, time.Since(startedAt))
		if err == nil {
			return result, nil
		}

		if !IsRetryable(err) || attempt == maxAttempts {
			return zero, err
		}
		shared.RecordRetry(ctx)
		if err := waitForRetry(ctx, retryJitter()); err != nil {
			return zero, err
		}
	}

	return zero, errors.New("write transaction attempts exhausted")
}

func runWriteAttempt[T any](
	ctx context.Context,
	database *sql.DB,
	isolation sql.IsolationLevel,
	fn TxFunc[T],
) (result T, err error) {
	tx, err := database.BeginTx(ctx, &sql.TxOptions{
		Isolation: isolation,
	})
	if err != nil {
		return result, err
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			_ = tx.Rollback()
			panic(recovered)
		}
	}()

	result, err = fn(tx)
	if err != nil {
		_ = tx.Rollback()
		return result, err
	}
	if err := tx.Commit(); err != nil {
		_ = tx.Rollback()
		return result, err
	}
	return result, nil
}

func IsRetryable(err error) bool {
	var postgresError *pq.Error
	if !errors.As(err, &postgresError) {
		return false
	}
	return postgresError.Code == "40P01" || postgresError.Code == "40001"
}

func retryJitter() time.Duration {
	var randomByte [1]byte
	if _, err := rand.Read(randomByte[:]); err != nil {
		return 10 * time.Millisecond
	}
	return time.Duration(5+randomByte[0]%16) * time.Millisecond
}

func waitForRetry(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
