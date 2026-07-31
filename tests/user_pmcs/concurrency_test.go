package user_pmcs_test

import (
	"context"
	"database/sql"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/stretchr/testify/require"

	"miltechserver/api/user_pmcs/persistence"
	"miltechserver/api/user_pmcs/shared"
)

func TestConcurrencyDatabaseSafetyGate(t *testing.T) {
	requireUserPmcsTestDatabase(t, testDB)
}

func TestConcurrencySimultaneousSavesWithOneETagMutateExactlyOnce(
	t *testing.T,
) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	requireUserPmcsTestDatabase(t, testDB)

	ownerUID := newUserPmcsTestUser(t)
	repository := newOwnedRepository()
	checklistID := uuid.New()
	created, err := repository.Create(
		ctx,
		ownerUID,
		checklistID,
		preparedTree(t, uuid.New()),
		shared.Precondition{Mode: shared.PreconditionCreate},
	)
	require.NoError(t, err)

	blocker, observer := dedicatedConnections(t, ctx)
	lockTx, err := blocker.BeginTx(ctx, nil)
	require.NoError(t, err)
	_, err = lockTx.ExecContext(
		ctx,
		`SELECT id FROM user_pmcs_checklists WHERE id = $1 FOR UPDATE`,
		checklistID,
	)
	require.NoError(t, err)

	drafts := []shared.PreparedRevision{
		preparedTree(t, uuid.New()),
		preparedTree(t, uuid.New()),
	}
	start := make(chan struct{})
	results := make(chan error, len(drafts))
	var workers sync.WaitGroup
	for _, draft := range drafts {
		draft := draft
		workers.Add(1)
		go func() {
			defer workers.Done()
			<-start
			_, putErr := repository.PutDraft(
				ctx,
				ownerUID,
				checklistID,
				draft,
				checklistPrecondition(
					checklistID,
					created.Aggregate.SyncVersion,
				),
			)
			results <- putErr
		}()
	}
	close(start)
	waitForBlockedUserPmcsQuery(t, ctx, observer)
	require.NoError(t, lockTx.Rollback())
	waitForWorkers(t, &workers, 10*time.Second)
	close(results)

	requireOneSuccessAndOneStale(t, results)
	require.Equal(t, int64(2), checklistVersion(t, checklistID))
	require.Equal(t, int64(2), accountVersion(t, ownerUID))
}

func TestConcurrencySameNextPublicationHasNoDuplicateOrSkip(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	ownerUID := newUserPmcsTestUser(t)
	repository := newOwnedRepository()
	checklistID := uuid.New()
	initial := preparedTree(t, uuid.New())
	created, err := repository.Create(
		ctx,
		ownerUID,
		checklistID,
		initial,
		shared.Precondition{Mode: shared.PreconditionCreate},
	)
	require.NoError(t, err)
	first, err := repository.Publish(
		ctx,
		ownerUID,
		checklistID,
		preparePublication(t, initial.Input, 1),
		checklistPrecondition(checklistID, created.Aggregate.SyncVersion),
	)
	require.NoError(t, err)

	publications := []shared.PreparedRevision{
		preparePublication(t, preparedTree(t, uuid.New()).Input, 2),
		preparePublication(t, preparedTree(t, uuid.New()).Input, 2),
	}
	results := runConcurrentOwnedMutations(
		t,
		ctx,
		publications,
		func(publication shared.PreparedRevision) error {
			_, publishErr := repository.Publish(
				ctx,
				ownerUID,
				checklistID,
				publication,
				checklistPrecondition(
					checklistID,
					first.Aggregate.SyncVersion,
				),
			)
			return publishErr
		},
	)
	requireOneSuccessAndOneStale(t, results)

	var numbers []int32
	rows, err := testDB.QueryContext(
		ctx,
		`SELECT revision_number
		 FROM user_pmcs_revisions
		 WHERE checklist_id = $1 AND revision_number IS NOT NULL
		 ORDER BY revision_number`,
		checklistID,
	)
	require.NoError(t, err)
	defer rows.Close()
	for rows.Next() {
		var number int32
		require.NoError(t, rows.Scan(&number))
		numbers = append(numbers, number)
	}
	require.NoError(t, rows.Err())
	require.Equal(t, []int32{1, 2}, numbers)
}

func TestConcurrencyHigherReleaseWinsAndLowerReleaseCannotRollback(
	t *testing.T,
) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	fixture := newReleasedChecklistFixture(t, 3)
	first, err := fixture.repository.Release(
		ctx,
		fixture.ownerUID,
		fixture.checklist,
		fixture.revisions[0].Input.ID,
		checklistPrecondition(
			fixture.checklist,
			fixture.aggregate.SyncVersion,
		),
	)
	require.NoError(t, err)
	second, err := fixture.repository.Release(
		ctx,
		fixture.ownerUID,
		fixture.checklist,
		fixture.revisions[1].Input.ID,
		checklistPrecondition(
			fixture.checklist,
			first.Aggregate.SyncVersion,
		),
	)
	require.NoError(t, err)

	releases := []shared.PreparedRevision{
		fixture.revisions[2],
		fixture.revisions[0],
	}
	results := runConcurrentOwnedMutations(
		t,
		ctx,
		releases,
		func(revision shared.PreparedRevision) error {
			_, releaseErr := fixture.repository.Release(
				ctx,
				fixture.ownerUID,
				fixture.checklist,
				revision.Input.ID,
				checklistPrecondition(
					fixture.checklist,
					second.Aggregate.SyncVersion,
				),
			)
			return releaseErr
		},
	)

	var successes, rejected int
	for result := range results {
		if result == nil {
			successes++
			continue
		}
		var apiError *shared.APIError
		require.ErrorAs(t, result, &apiError)
		require.Contains(
			t,
			[]string{"invalid_transition", "stale_precondition"},
			apiError.Code,
		)
		rejected++
	}
	require.Equal(t, 1, successes)
	require.Equal(t, 1, rejected)

	var latest int32
	require.NoError(
		t,
		testDB.QueryRowContext(
			ctx,
			`SELECT latest_release_revision_number
			 FROM user_pmcs_community_sources
			 WHERE checklist_id = $1`,
			fixture.checklist,
		).Scan(&latest),
	)
	require.Equal(t, int32(3), latest)
}

func TestConcurrencyFirstMutationsCreateOneOrderedSyncState(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	ownerUID := newUserPmcsTestUser(t)
	repository := newOwnedRepository()
	drafts := []shared.PreparedRevision{
		preparedTree(t, uuid.New()),
		preparedTree(t, uuid.New()),
	}
	checklistIDs := []uuid.UUID{uuid.New(), uuid.New()}
	start := make(chan struct{})
	results := make(chan error, 2)
	var workers sync.WaitGroup
	for index := range drafts {
		index := index
		workers.Add(1)
		go func() {
			defer workers.Done()
			<-start
			_, createErr := repository.Create(
				ctx,
				ownerUID,
				checklistIDs[index],
				drafts[index],
				shared.Precondition{Mode: shared.PreconditionCreate},
			)
			results <- createErr
		}()
	}
	close(start)
	waitForWorkers(t, &workers, 10*time.Second)
	close(results)
	for result := range results {
		require.NoError(t, result)
	}

	var syncRows int
	require.NoError(
		t,
		testDB.QueryRowContext(
			ctx,
			`SELECT count(*) FROM user_pmcs_sync_state WHERE user_uid = $1`,
			ownerUID,
		).Scan(&syncRows),
	)
	require.Equal(t, 1, syncRows)

	rows, err := testDB.QueryContext(
		ctx,
		`SELECT account_change_version
		 FROM user_pmcs_checklists
		 WHERE owner_uid = $1
		 ORDER BY account_change_version`,
		ownerUID,
	)
	require.NoError(t, err)
	defer rows.Close()
	var versions []int64
	for rows.Next() {
		var version int64
		require.NoError(t, rows.Scan(&version))
		versions = append(versions, version)
	}
	require.NoError(t, rows.Err())
	require.Equal(t, []int64{1, 2}, versions)
	require.Equal(t, int64(2), accountVersion(t, ownerUID))
}

func TestConcurrencyDifferentUsersDoNotGloballySerialize(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	blockedUID := newUserPmcsTestUser(t)
	independentUID := newUserPmcsTestUser(t)
	blocker, observer := dedicatedConnections(t, ctx)
	lockTx, err := blocker.BeginTx(ctx, nil)
	require.NoError(t, err)
	_, err = lockTx.ExecContext(
		ctx,
		`INSERT INTO user_pmcs_sync_state (user_uid, current_version)
		 VALUES ($1, 0)
		 ON CONFLICT (user_uid) DO NOTHING`,
		blockedUID,
	)
	require.NoError(t, err)
	_, err = lockTx.ExecContext(
		ctx,
		`SELECT current_version
		 FROM user_pmcs_sync_state
		 WHERE user_uid = $1
		 FOR UPDATE`,
		blockedUID,
	)
	require.NoError(t, err)

	type outcome struct {
		uid string
		err error
	}
	results := make(chan outcome, 2)
	start := make(chan struct{})
	drafts := map[string]shared.PreparedRevision{
		blockedUID:     preparedTree(t, uuid.New()),
		independentUID: preparedTree(t, uuid.New()),
	}
	var workers sync.WaitGroup
	for _, uid := range []string{blockedUID, independentUID} {
		uid := uid
		workers.Add(1)
		go func() {
			defer workers.Done()
			<-start
			_, createErr := newOwnedRepository().Create(
				ctx,
				uid,
				uuid.New(),
				drafts[uid],
				shared.Precondition{Mode: shared.PreconditionCreate},
			)
			results <- outcome{uid: uid, err: createErr}
		}()
	}
	close(start)
	waitForBlockedUserPmcsQuery(t, ctx, observer)

	select {
	case result := <-results:
		require.Equal(t, independentUID, result.uid)
		require.NoError(t, result.err)
	case <-time.After(5 * time.Second):
		t.Fatal("independent user mutation was globally serialized")
	}
	require.NoError(t, lockTx.Rollback())
	waitForWorkers(t, &workers, 10*time.Second)
	close(results)
	for result := range results {
		require.NoError(t, result.err)
		require.Equal(t, blockedUID, result.uid)
	}
}

func TestConcurrencyDeadlockClassificationAndBoundedRetryExhaustion(
	t *testing.T,
) {
	require.True(
		t,
		persistence.IsRetryable(
			&pq.Error{Code: pq.ErrorCode("40P01")},
		),
	)
	require.True(
		t,
		persistence.IsRetryable(
			&pq.Error{Code: pq.ErrorCode("40001")},
		),
	)
	require.False(
		t,
		persistence.IsRetryable(
			&pq.Error{Code: pq.ErrorCode("23505")},
		),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	attempts := 0
	_, err := persistence.WithWriteTx(
		ctx,
		testDB,
		3,
		func(*sql.Tx) (struct{}, error) {
			attempts++
			return struct{}{}, &pq.Error{Code: pq.ErrorCode("40P01")}
		},
	)
	require.Error(t, err)
	require.Equal(t, 3, attempts)
}

func TestConcurrencyLaterMutationAppearsOnNextDeltaPageOnly(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	ownerUID := newUserPmcsTestUser(t)
	repository := newOwnedRepository()
	for range 2 {
		_, err := repository.Create(
			ctx,
			ownerUID,
			uuid.New(),
			preparedTree(t, uuid.New()),
			shared.Precondition{Mode: shared.PreconditionCreate},
		)
		require.NoError(t, err)
	}

	deltaRepository := newAccountDeltaRepository()
	firstPage, err := deltaRepository.GetDelta(
		ctx,
		ownerUID,
		0,
		1,
		shared.DefaultConfig().MaxDeltaResponseBytes,
	)
	require.NoError(t, err)
	require.Len(t, firstPage.Changes, 1)
	require.Equal(t, int64(2), firstPage.AccountVersion)
	require.Equal(t, int64(1), firstPage.ThroughCursor)
	require.True(t, firstPage.HasMore)

	laterChecklistID := uuid.New()
	_, err = repository.Create(
		ctx,
		ownerUID,
		laterChecklistID,
		preparedTree(t, uuid.New()),
		shared.Precondition{Mode: shared.PreconditionCreate},
	)
	require.NoError(t, err)

	nextPage, err := deltaRepository.GetDelta(
		ctx,
		ownerUID,
		firstPage.ThroughCursor,
		25,
		shared.DefaultConfig().MaxDeltaResponseBytes,
	)
	require.NoError(t, err)
	require.Equal(t, int64(3), nextPage.AccountVersion)
	require.Len(t, nextPage.Changes, 2)
	require.Equal(t, int64(2), nextPage.Changes[0].AccountChangeVersion)
	require.Equal(t, int64(3), nextPage.Changes[1].AccountChangeVersion)
	require.NotNil(t, nextPage.Changes[1].Checklist)
	require.Equal(t, laterChecklistID, nextPage.Changes[1].Checklist.ID)
}

func dedicatedConnections(
	t *testing.T,
	ctx context.Context,
) (*sql.Conn, *sql.Conn) {
	t.Helper()
	first, err := testDB.Conn(ctx)
	require.NoError(t, err)
	second, err := testDB.Conn(ctx)
	if err != nil {
		_ = first.Close()
		require.NoError(t, err)
	}
	t.Cleanup(func() {
		require.NoError(t, first.Close())
		require.NoError(t, second.Close())
	})
	return first, second
}

func waitForBlockedUserPmcsQuery(
	t *testing.T,
	ctx context.Context,
	observer *sql.Conn,
) {
	t.Helper()
	require.Eventually(t, func() bool {
		var blocked int
		err := observer.QueryRowContext(
			ctx,
			`SELECT count(*)
			 FROM pg_stat_activity
			 WHERE pid <> pg_backend_pid()
			   AND wait_event_type = 'Lock'
			   AND query LIKE '%user_pmcs_%'`,
		).Scan(&blocked)
		return err == nil && blocked > 0
	}, 5*time.Second, 20*time.Millisecond)
}

func waitForWorkers(
	t *testing.T,
	workers *sync.WaitGroup,
	timeout time.Duration,
) {
	t.Helper()
	completed := make(chan struct{})
	go func() {
		workers.Wait()
		close(completed)
	}()
	select {
	case <-completed:
	case <-time.After(timeout):
		t.Fatal("concurrent workers did not finish before the timeout")
	}
}

func requireOneSuccessAndOneStale(t *testing.T, results <-chan error) {
	t.Helper()
	var successes, stale int
	for result := range results {
		if result == nil {
			successes++
			continue
		}
		var apiError *shared.APIError
		require.ErrorAs(t, result, &apiError)
		require.Equal(t, "stale_precondition", apiError.Code)
		stale++
	}
	require.Equal(t, 1, successes)
	require.Equal(t, 1, stale)
}

func runConcurrentOwnedMutations(
	t *testing.T,
	ctx context.Context,
	revisions []shared.PreparedRevision,
	mutate func(shared.PreparedRevision) error,
) <-chan error {
	t.Helper()
	start := make(chan struct{})
	results := make(chan error, len(revisions))
	var workers sync.WaitGroup
	for _, revision := range revisions {
		revision := revision
		workers.Add(1)
		go func() {
			defer workers.Done()
			<-start
			results <- mutate(revision)
		}()
	}
	close(start)
	waitForWorkers(t, &workers, 10*time.Second)
	close(results)
	return results
}
