package persistence

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

type accountCleaner struct{}

func NewAccountCleaner() *accountCleaner {
	return &accountCleaner{}
}

func (cleaner *accountCleaner) CleanupAccount(
	ctx context.Context,
	tx *sql.Tx,
	uid string,
) error {
	if err := lockOptionalRow(
		ctx,
		tx,
		`SELECT user_uid FROM user_pmcs_sync_state
		  WHERE user_uid = $1
		  FOR UPDATE`,
		uid,
	); err != nil {
		return fmt.Errorf("lock account sync state: %w", err)
	}

	ownedChecklistIDs, err := lockUUIDRows(
		ctx,
		tx,
		`SELECT id
		   FROM user_pmcs_checklists
		  WHERE owner_uid = $1
		  ORDER BY id
		  FOR UPDATE`,
		uid,
	)
	if err != nil {
		return fmt.Errorf("lock owned checklist roots: %w", err)
	}

	checklistValues := uuidStrings(ownedChecklistIDs)
	sourceIDs := make(map[uuid.UUID]struct{})
	if len(checklistValues) > 0 {
		lockedSources, err := lockUUIDRows(
			ctx,
			tx,
			`SELECT checklist_id
			   FROM user_pmcs_community_sources
			  WHERE checklist_id = ANY($1)
			  ORDER BY checklist_id
			  FOR UPDATE`,
			pq.Array(checklistValues),
		)
		if err != nil {
			return fmt.Errorf("lock community sources: %w", err)
		}
		for _, checklistID := range lockedSources {
			sourceIDs[checklistID] = struct{}{}
		}
	}

	if _, err := lockUUIDRows(
		ctx,
		tx,
		`SELECT checklist_id
		   FROM user_pmcs_subscriptions
		  WHERE subscriber_uid = $1
		  ORDER BY checklist_id
		  FOR UPDATE`,
		uid,
	); err != nil {
		return fmt.Errorf("lock account subscriptions: %w", err)
	}

	if len(checklistValues) > 0 {
		if _, err := lockUUIDRows(
			ctx,
			tx,
			`SELECT id
			   FROM user_pmcs_revisions
			  WHERE checklist_id = ANY($1)
			  ORDER BY checklist_id, id
			  FOR UPDATE`,
			pq.Array(checklistValues),
		); err != nil {
			return fmt.Errorf("lock owned revisions: %w", err)
		}
		if _, err := lockUUIDRows(
			ctx,
			tx,
			`SELECT revision_id
			   FROM user_pmcs_community_releases
			  WHERE checklist_id = ANY($1)
			  ORDER BY checklist_id, revision_id
			  FOR UPDATE`,
			pq.Array(checklistValues),
		); err != nil {
			return fmt.Errorf("lock owned releases: %w", err)
		}
	}

	if _, err := tx.ExecContext(
		ctx,
		`DELETE FROM user_pmcs_subscriptions WHERE subscriber_uid = $1`,
		uid,
	); err != nil {
		return fmt.Errorf("delete account subscriptions: %w", err)
	}

	for _, checklistID := range ownedChecklistIDs {
		if _, released := sourceIDs[checklistID]; !released {
			if err := deletePrivateChecklist(ctx, tx, checklistID); err != nil {
				return err
			}
			continue
		}
		if err := cleanReleasedChecklist(ctx, tx, checklistID); err != nil {
			return err
		}
	}

	if _, err := tx.ExecContext(
		ctx,
		`DELETE FROM user_pmcs_sync_state WHERE user_uid = $1`,
		uid,
	); err != nil {
		return fmt.Errorf("delete account sync state: %w", err)
	}
	return nil
}

func cleanReleasedChecklist(
	ctx context.Context,
	tx *sql.Tx,
	checklistID uuid.UUID,
) error {
	if _, err := tx.ExecContext(
		ctx,
		`UPDATE user_pmcs_community_sources
		    SET status = 'retired',
		        current_release_revision_id = NULL,
		        retired_at = COALESCE(retired_at, now()),
		        updated_at = now()
		  WHERE checklist_id = $1`,
		checklistID,
	); err != nil {
		return fmt.Errorf("retire community source %s: %w", checklistID, err)
	}

	var activePins int
	if err := tx.QueryRowContext(
		ctx,
		`SELECT count(*)
		   FROM user_pmcs_subscriptions
		  WHERE checklist_id = $1
		    AND deleted_at IS NULL`,
		checklistID,
	).Scan(&activePins); err != nil {
		return fmt.Errorf("count active pins for %s: %w", checklistID, err)
	}

	if activePins == 0 {
		if _, err := tx.ExecContext(
			ctx,
			`DELETE FROM user_pmcs_community_sources
			  WHERE checklist_id = $1`,
			checklistID,
		); err != nil {
			return fmt.Errorf("delete unpinned source %s: %w", checklistID, err)
		}
		if _, err := tx.ExecContext(
			ctx,
			`DELETE FROM user_pmcs_community_releases
			  WHERE checklist_id = $1`,
			checklistID,
		); err != nil {
			return fmt.Errorf("delete unpinned releases %s: %w", checklistID, err)
		}
		return deletePrivateChecklist(ctx, tx, checklistID)
	}

	if _, err := tx.ExecContext(
		ctx,
		`DELETE FROM user_pmcs_community_releases AS release
		  WHERE release.checklist_id = $1
		    AND NOT EXISTS (
		        SELECT 1
		          FROM user_pmcs_subscriptions AS subscription
		         WHERE subscription.checklist_id = release.checklist_id
		           AND subscription.installed_revision_id = release.revision_id
		           AND subscription.deleted_at IS NULL
		    )`,
		checklistID,
	); err != nil {
		return fmt.Errorf("delete unpinned releases %s: %w", checklistID, err)
	}
	if _, err := tx.ExecContext(
		ctx,
		`DELETE FROM user_pmcs_revisions AS revision
		  WHERE revision.checklist_id = $1
		    AND NOT EXISTS (
		        SELECT 1
		          FROM user_pmcs_subscriptions AS subscription
		         WHERE subscription.checklist_id = revision.checklist_id
		           AND subscription.installed_revision_id = revision.id
		           AND subscription.deleted_at IS NULL
		    )`,
		checklistID,
	); err != nil {
		return fmt.Errorf("delete unpinned revisions %s: %w", checklistID, err)
	}
	if _, err := tx.ExecContext(
		ctx,
		`UPDATE user_pmcs_checklists
		    SET owner_uid = NULL,
		        deleted_at = COALESCE(deleted_at, now()),
		        updated_at = now()
		  WHERE id = $1`,
		checklistID,
	); err != nil {
		return fmt.Errorf("anonymize retained checklist %s: %w", checklistID, err)
	}
	return nil
}

func deletePrivateChecklist(
	ctx context.Context,
	tx *sql.Tx,
	checklistID uuid.UUID,
) error {
	if _, err := tx.ExecContext(
		ctx,
		`DELETE FROM user_pmcs_revisions WHERE checklist_id = $1`,
		checklistID,
	); err != nil {
		return fmt.Errorf("delete checklist revisions %s: %w", checklistID, err)
	}
	if _, err := tx.ExecContext(
		ctx,
		`DELETE FROM user_pmcs_checklists WHERE id = $1`,
		checklistID,
	); err != nil {
		return fmt.Errorf("delete checklist root %s: %w", checklistID, err)
	}
	return nil
}

func lockOptionalRow(
	ctx context.Context,
	tx *sql.Tx,
	query string,
	argument any,
) error {
	var value string
	err := tx.QueryRowContext(ctx, query, argument).Scan(&value)
	if err == sql.ErrNoRows {
		return nil
	}
	return err
}

func lockUUIDRows(
	ctx context.Context,
	tx *sql.Tx,
	query string,
	arguments ...any,
) ([]uuid.UUID, error) {
	rows, err := tx.QueryContext(ctx, query, arguments...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	values := make([]uuid.UUID, 0)
	for rows.Next() {
		var value uuid.UUID
		if err := rows.Scan(&value); err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return values, nil
}

func uuidStrings(values []uuid.UUID) []string {
	result := make([]string, len(values))
	for index, value := range values {
		result[index] = value.String()
	}
	return result
}
