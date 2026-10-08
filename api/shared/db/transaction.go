package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// WithTx runs fn inside a transaction on conn, committing if fn returns
// a nil error and rolling back otherwise. The rollback error (if any) is
// joined with fn's original error rather than replacing it.
func WithTx[T any](conn *sql.DB, fn func(tx *sql.Tx) (T, error)) (T, error) {
	var zero T

	tx, err := conn.Begin()
	if err != nil {
		return zero, fmt.Errorf("begin transaction: %w", err)
	}

	result, err := fn(tx)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
			return zero, errors.Join(err, fmt.Errorf("rollback: %w", rbErr))
		}
		return zero, err
	}

	if err := tx.Commit(); err != nil {
		return zero, fmt.Errorf("commit transaction: %w", err)
	}

	return result, nil
}

// WithTxContext binds the transaction lifetime to ctx at READ COMMITTED.
// Callbacks should also pass ctx to statement methods for query cancellation.
func WithTxContext(ctx context.Context, conn *sql.DB, fn func(*sql.Tx) error) error {
	return WithTxOptions(ctx, conn, &sql.TxOptions{Isolation: sql.LevelReadCommitted}, fn)
}

// WithTxOptions runs fn in a transaction with the supplied context and options.
// A commit error is ambiguous: callers must resolve it with the same operation ID.
func WithTxOptions(ctx context.Context, conn *sql.DB, opts *sql.TxOptions, fn func(*sql.Tx) error) (err error) {
	tx, err := conn.BeginTx(ctx, opts)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		if rollbackErr := tx.Rollback(); rollbackErr != nil && !errors.Is(rollbackErr, sql.ErrTxDone) {
			err = errors.Join(err, fmt.Errorf("rollback: %w", rollbackErr))
		}
	}()
	if err := fn(tx); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		// database/sql may finish its cancellation rollback before Commit is called.
		if errors.Is(err, sql.ErrTxDone) && ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}
