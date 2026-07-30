package persistence

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/lib/pq"
)

type TxFunc[T any] func(*sql.Tx) (T, error)

func WithWriteTx[T any](
	ctx context.Context,
	database *sql.DB,
	maxAttempts int,
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
		result, err := runWriteAttempt(ctx, database, fn)
		if err == nil {
			return result, nil
		}

		if !IsRetryable(err) || attempt == maxAttempts {
			return zero, err
		}
		if err := waitForRetry(ctx, retryJitter()); err != nil {
			return zero, err
		}
	}

	return zero, errors.New("write transaction attempts exhausted")
}

func runWriteAttempt[T any](
	ctx context.Context,
	database *sql.DB,
	fn TxFunc[T],
) (result T, err error) {
	tx, err := database.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelReadCommitted,
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
