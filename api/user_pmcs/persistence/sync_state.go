package persistence

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"miltechserver/api/user_pmcs/shared"
)

func LockAccountVersion(
	ctx context.Context,
	tx *sql.Tx,
	userUID string,
) (int64, error) {
	var userExists bool
	if err := tx.QueryRowContext(
		ctx,
		`SELECT EXISTS (SELECT 1 FROM users WHERE uid = $1)`,
		userUID,
	).Scan(&userExists); err != nil {
		return 0, fmt.Errorf("verify user account: %w", err)
	}
	if !userExists {
		return 0, shared.NewAccountNotInitialized(
			"account is not initialized",
			nil,
		)
	}

	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO user_pmcs_sync_state (user_uid, current_version)
		 VALUES ($1, 0)
		 ON CONFLICT (user_uid) DO NOTHING`,
		userUID,
	); err != nil {
		return 0, fmt.Errorf("initialize user PMCS sync state: %w", err)
	}

	var currentVersion int64
	if err := tx.QueryRowContext(
		ctx,
		`SELECT current_version
		 FROM user_pmcs_sync_state
		 WHERE user_uid = $1
		 FOR UPDATE`,
		userUID,
	).Scan(&currentVersion); err != nil {
		return 0, fmt.Errorf("lock user PMCS sync state: %w", err)
	}
	return currentVersion, nil
}

func AdvanceAccountVersion(
	ctx context.Context,
	tx *sql.Tx,
	userUID string,
) (int64, error) {
	var currentVersion int64
	err := tx.QueryRowContext(
		ctx,
		`UPDATE user_pmcs_sync_state
		 SET current_version = current_version + 1,
		     updated_at = now()
		 WHERE user_uid = $1
		 RETURNING current_version`,
		userUID,
	).Scan(&currentVersion)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, shared.NewAccountNotInitialized(
			"account is not initialized",
			nil,
		)
	}
	if err != nil {
		return 0, fmt.Errorf("advance user PMCS account version: %w", err)
	}
	return currentVersion, nil
}
