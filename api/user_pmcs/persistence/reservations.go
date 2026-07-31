package persistence

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"sort"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"miltechserver/api/user_pmcs/shared"
)

type ContentUUIDReservations struct {
	ids []uuid.UUID
}

func AcquireContentUUIDReservations(
	ctx context.Context,
	tx *sql.Tx,
	revision shared.RevisionInput,
) (ContentUUIDReservations, error) {
	if tx == nil {
		return ContentUUIDReservations{}, fmt.Errorf(
			"content UUID reservation transaction is nil",
		)
	}

	ids := contentUUIDs(revision)
	sort.Slice(ids, func(left, right int) bool {
		return bytes.Compare(ids[left][:], ids[right][:]) < 0
	})

	// ORDER BY makes unique-index acquisition deterministic in PostgreSQL;
	// sorted Go input alone does not define a multi-row INSERT executor order.
	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO user_pmcs_content_uuid_reservations (id)
		 SELECT submitted.id
		 FROM unnest($1::uuid[]) AS submitted(id)
		 ORDER BY submitted.id`,
		pq.Array(ids),
	); err != nil {
		return ContentUUIDReservations{}, fmt.Errorf(
			"acquire content UUID reservations: %w",
			err,
		)
	}
	return ContentUUIDReservations{ids: ids}, nil
}

func (reservations ContentUUIDReservations) Release(
	ctx context.Context,
	tx *sql.Tx,
) error {
	if tx == nil {
		return fmt.Errorf("content UUID reservation transaction is nil")
	}
	if len(reservations.ids) == 0 {
		return nil
	}
	if _, err := tx.ExecContext(
		ctx,
		`DELETE FROM user_pmcs_content_uuid_reservations
		 WHERE id = ANY($1::uuid[])`,
		pq.Array(reservations.ids),
	); err != nil {
		return fmt.Errorf("release content UUID reservations: %w", err)
	}
	return nil
}

func contentUUIDs(revision shared.RevisionInput) []uuid.UUID {
	ids := []uuid.UUID{revision.ID}
	for _, section := range revision.Sections {
		ids = append(ids, section.ID)
		for _, item := range section.Items {
			ids = append(ids, item.ID)
			for _, notice := range item.Notices {
				ids = append(ids, notice.ID)
			}
			for _, step := range item.ProcedureSteps {
				ids = append(ids, step.ID)
			}
		}
	}
	return ids
}
