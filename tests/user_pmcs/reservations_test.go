package user_pmcs_test

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"miltechserver/api/user_pmcs/persistence"
	"miltechserver/api/user_pmcs/shared"
)

func TestContentUUIDReservationLifecycle(t *testing.T) {
	requireUserPmcsTestDatabase(t, testDB)
	requireReservationRowCount(t, 0)

	t.Run("success", func(t *testing.T) {
		tx, err := testDB.BeginTx(context.Background(), nil)
		require.NoError(t, err)
		reservations, err := persistence.AcquireContentUUIDReservations(
			context.Background(),
			tx,
			preparedTree(t, uuid.New()).Input,
		)
		require.NoError(t, err)
		requireReservationRowCountInTx(t, tx, 5)
		require.NoError(t, reservations.Release(context.Background(), tx))
		require.NoError(t, tx.Commit())
		requireReservationRowCount(t, 0)
	})

	t.Run("injected failure", func(t *testing.T) {
		injectedFailure := errors.New("injected reservation failure")
		_, err := persistence.WithWriteTx(
			context.Background(),
			testDB,
			1,
			func(tx *sql.Tx) (struct{}, error) {
				_, acquireErr := persistence.AcquireContentUUIDReservations(
					context.Background(),
					tx,
					preparedTree(t, uuid.New()).Input,
				)
				if acquireErr != nil {
					return struct{}{}, acquireErr
				}
				return struct{}{}, injectedFailure
			},
		)
		require.ErrorIs(t, err, injectedFailure)
		requireReservationRowCount(t, 0)
	})

	t.Run("explicit rollback", func(t *testing.T) {
		tx, err := testDB.BeginTx(context.Background(), nil)
		require.NoError(t, err)
		_, err = persistence.AcquireContentUUIDReservations(
			context.Background(),
			tx,
			preparedTree(t, uuid.New()).Input,
		)
		require.NoError(t, err)
		requireReservationRowCountInTx(t, tx, 5)
		require.NoError(t, tx.Rollback())
		requireReservationRowCount(t, 0)
	})

	t.Run("context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		tx, err := testDB.BeginTx(ctx, nil)
		require.NoError(t, err)
		_, err = persistence.AcquireContentUUIDReservations(
			ctx,
			tx,
			preparedTree(t, uuid.New()).Input,
		)
		require.NoError(t, err)
		cancel()
		require.Eventually(t, func() bool {
			return tx.Rollback() != nil
		}, 5*time.Second, 10*time.Millisecond)
		requireReservationRowCount(t, 0)
	})

	t.Run("broken connection", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		connection, err := testDB.Conn(ctx)
		require.NoError(t, err)
		defer connection.Close()
		tx, err := connection.BeginTx(ctx, nil)
		require.NoError(t, err)
		_, err = persistence.AcquireContentUUIDReservations(
			ctx,
			tx,
			preparedTree(t, uuid.New()).Input,
		)
		require.NoError(t, err)
		var backendPID int
		require.NoError(t, tx.QueryRowContext(
			ctx,
			`SELECT pg_backend_pid()`,
		).Scan(&backendPID))
		var terminated bool
		require.NoError(t, testDB.QueryRowContext(
			ctx,
			`SELECT pg_terminate_backend($1)`,
			backendPID,
		).Scan(&terminated))
		require.True(t, terminated)
		require.Eventually(t, func() bool {
			return reservationRowCount(t) == 0
		}, 5*time.Second, 10*time.Millisecond)
	})
}

func TestContentUUIDReservationsUseOneGlobalOrder(t *testing.T) {
	requireUserPmcsTestDatabase(t, testDB)
	requireReservationRowCount(t, 0)

	firstRevision, secondRevision := reverseOrderedOverlappingRevisions(t)
	type acquisition struct {
		tx           *sql.Tx
		reservations persistence.ContentUUIDReservations
		err          error
	}
	start := make(chan struct{})
	acquired := make(chan acquisition, 2)
	var workers sync.WaitGroup
	for _, revision := range []shared.RevisionInput{firstRevision, secondRevision} {
		revision := revision
		workers.Add(1)
		go func() {
			defer workers.Done()
			tx, err := testDB.BeginTx(context.Background(), nil)
			if err != nil {
				acquired <- acquisition{err: err}
				return
			}
			<-start
			reservations, err :=
				persistence.AcquireContentUUIDReservations(
					context.Background(),
					tx,
					revision,
				)
			acquired <- acquisition{
				tx:           tx,
				reservations: reservations,
				err:          err,
			}
		}()
	}
	close(start)

	first := <-acquired
	require.NoError(t, first.err)
	require.NoError(t, first.reservations.Release(context.Background(), first.tx))
	require.NoError(t, first.tx.Commit())

	second := <-acquired
	require.NoError(t, second.err)
	require.NoError(t, second.reservations.Release(context.Background(), second.tx))
	require.NoError(t, second.tx.Commit())
	workers.Wait()
	requireReservationRowCount(t, 0)
}

func TestTreeMutationsWaitForSubmittedUUIDReservations(t *testing.T) {
	t.Run("Create", func(t *testing.T) {
		ownerUID := newUserPmcsTestUser(t)
		draft := preparedTree(t, uuid.New())
		assertMutationWaitsForReservation(
			t,
			draft.Input.ID,
			func(ctx context.Context) error {
				_, err := newOwnedRepository().Create(
					ctx,
					ownerUID,
					uuid.New(),
					draft,
					shared.Precondition{Mode: shared.PreconditionCreate},
				)
				return err
			},
		)
	})

	t.Run("PutDraft", func(t *testing.T) {
		ownerUID := newUserPmcsTestUser(t)
		repository := newOwnedRepository()
		checklistID := uuid.New()
		created, err := repository.Create(
			context.Background(),
			ownerUID,
			checklistID,
			preparedTree(t, uuid.New()),
			shared.Precondition{Mode: shared.PreconditionCreate},
		)
		require.NoError(t, err)
		replacement := preparedTree(t, uuid.New())
		assertMutationWaitsForReservation(
			t,
			replacement.Input.ID,
			func(ctx context.Context) error {
				_, err := repository.PutDraft(
					ctx,
					ownerUID,
					checklistID,
					replacement,
					checklistPrecondition(
						checklistID,
						created.Aggregate.SyncVersion,
					),
				)
				return err
			},
		)
	})

	t.Run("Publish", func(t *testing.T) {
		ownerUID := newUserPmcsTestUser(t)
		repository := newOwnedRepository()
		checklistID := uuid.New()
		draft := preparedTree(t, uuid.New())
		created, err := repository.Create(
			context.Background(),
			ownerUID,
			checklistID,
			draft,
			shared.Precondition{Mode: shared.PreconditionCreate},
		)
		require.NoError(t, err)
		publication := preparePublication(t, draft.Input, 1)
		assertMutationWaitsForReservation(
			t,
			publication.Input.ID,
			func(ctx context.Context) error {
				_, err := repository.Publish(
					ctx,
					ownerUID,
					checklistID,
					publication,
					checklistPrecondition(
						checklistID,
						created.Aggregate.SyncVersion,
					),
				)
				return err
			},
		)
	})
}

func assertMutationWaitsForReservation(
	t *testing.T,
	reservedID uuid.UUID,
	mutate func(context.Context) error,
) {
	t.Helper()
	requireReservationRowCount(t, 0)
	blocker, err := testDB.BeginTx(context.Background(), nil)
	require.NoError(t, err)
	_, err = blocker.ExecContext(
		context.Background(),
		`INSERT INTO user_pmcs_content_uuid_reservations (id) VALUES ($1)`,
		reservedID,
	)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	result := make(chan error, 1)
	go func() {
		result <- mutate(ctx)
	}()
	select {
	case err := <-result:
		_ = blocker.Rollback()
		require.Failf(
			t,
			"mutation ignored reservation",
			"mutation completed before reservation released: %v",
			err,
		)
	case <-time.After(100 * time.Millisecond):
	}

	require.NoError(t, blocker.Rollback())
	require.NoError(t, <-result)
	requireReservationRowCount(t, 0)
}

func reverseOrderedOverlappingRevisions(
	t *testing.T,
) (shared.RevisionInput, shared.RevisionInput) {
	t.Helper()
	first := preparedTree(t, uuid.New()).Input
	second := preparedTree(t, uuid.New()).Input
	firstSharedID := uuid.New()
	secondSharedID := uuid.New()
	first.Sections[0].ID = firstSharedID
	first.Sections[0].Items[0].ID = secondSharedID
	second.Sections[0].ID = secondSharedID
	second.Sections[0].Items[0].ID = firstSharedID
	return first, second
}

func requireReservationRowCount(t *testing.T, expected int) {
	t.Helper()
	require.Equal(t, expected, reservationRowCount(t))
}

func reservationRowCount(t *testing.T) int {
	t.Helper()
	var count int
	require.NoError(t, testDB.QueryRowContext(
		context.Background(),
		`SELECT count(*) FROM user_pmcs_content_uuid_reservations`,
	).Scan(&count))
	return count
}

func requireReservationRowCountInTx(
	t *testing.T,
	tx *sql.Tx,
	expected int,
) {
	t.Helper()
	var count int
	require.NoError(t, tx.QueryRowContext(
		context.Background(),
		`SELECT count(*) FROM user_pmcs_content_uuid_reservations`,
	).Scan(&count))
	require.Equal(t, expected, count)
}
