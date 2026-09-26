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

// WithTxContext binds every statement and the transaction lifetime to ctx.
// A commit error is ambiguous: callers must resolve it with the same operation ID.
func WithTxContext(ctx context.Context, conn *sql.DB, fn func(*sql.Tx) error) error {
	tx, err := conn.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit()
}
