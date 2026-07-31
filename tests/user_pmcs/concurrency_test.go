package user_pmcs_test

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/stretchr/testify/require"

	"miltechserver/api/user_pmcs/community"
	"miltechserver/api/user_pmcs/owned"
	"miltechserver/api/user_pmcs/persistence"
	"miltechserver/api/user_pmcs/shared"
	userpmcssync "miltechserver/api/user_pmcs/sync"
)

func TestConcurrencyDatabaseSafetyGate(t *testing.T) {
	requireUserPmcsTestDatabase(t, testDB)
}

func TestConcurrencyBarrierIdentifiesOnlyExpectedFixtureWorkers(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	userUID := newUserPmcsTestUser(t)
	blocker, waiter := dedicatedConnections(t, ctx)
	observer, err := testDB.Conn(ctx)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, observer.Close())
	})

	blockerPID := backendPID(t, ctx, blocker)
	waiterPID := backendPID(t, ctx, waiter)
	observerPID := backendPID(t, ctx, observer)
	_, err = testDB.ExecContext(
		ctx,
		`INSERT INTO user_pmcs_sync_state (user_uid, current_version)
		 VALUES ($1, 0)`,
		userUID,
	)
	require.NoError(t, err)
	lockTx, err := blocker.BeginTx(ctx, nil)
	require.NoError(t, err)
	_, err = lockTx.ExecContext(
		ctx,
		`SELECT current_version
		 FROM user_pmcs_sync_state
		 WHERE user_uid = $1
		 FOR UPDATE`,
		userUID,
	)
	require.NoError(t, err)

	waitResult := make(chan error, 1)
	go func() {
		_, waitErr := waiter.ExecContext(
			ctx,
			`SELECT current_version
			 FROM user_pmcs_sync_state
			 WHERE user_uid = $1
			 FOR UPDATE`,
			userUID,
		)
		waitResult <- waitErr
	}()

	require.Eventually(t, func() bool {
		blocked, queryErr := queryBlockedWorkerPIDs(
			ctx,
			observer,
			blockerPID,
			[]int{waiterPID, observerPID},
		)
		return queryErr == nil && slices.Equal([]int{waiterPID}, blocked)
	}, 5*time.Second, 20*time.Millisecond)
	require.NoError(t, lockTx.Rollback())
	require.NoError(t, <-waitResult)
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
		`SELECT current_version
		 FROM user_pmcs_sync_state
		 WHERE user_uid = $1
		 FOR UPDATE`,
		ownerUID,
	)
	require.NoError(t, err)
	blockerPID := backendPID(t, ctx, blocker)
	workerOne := newDedicatedRepositoryWorker(t, ctx, "same-etag-one")
	workerTwo := newDedicatedRepositoryWorker(t, ctx, "same-etag-two")
	workersByIndex := []dedicatedRepositoryWorker{workerOne, workerTwo}

	drafts := []shared.PreparedRevision{
		preparedTree(t, uuid.New()),
		preparedTree(t, uuid.New()),
	}
	start := make(chan struct{})
	results := make(chan error, len(drafts))
	var workers sync.WaitGroup
	for index, draft := range drafts {
		index, draft := index, draft
		workers.Add(1)
		go func() {
			defer workers.Done()
			<-start
			_, putErr := workersByIndex[index].owned().PutDraft(
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
	waitForExactBlockedWorkers(
		t,
		ctx,
		observer,
		blockerPID,
		[]int{workerOne.pid, workerTwo.pid},
		[]int{workerOne.pid, workerTwo.pid},
		"user_pmcs_sync_state",
	)
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

	blocker, observer := dedicatedConnections(t, ctx)
	lockTx, err := blocker.BeginTx(ctx, nil)
	require.NoError(t, err)
	_, err = lockTx.ExecContext(
		ctx,
		`SELECT current_version
		 FROM user_pmcs_sync_state
		 WHERE user_uid = $1
		 FOR UPDATE`,
		ownerUID,
	)
	require.NoError(t, err)
	blockerPID := backendPID(t, ctx, blocker)
	workerOne := newDedicatedRepositoryWorker(t, ctx, "publication-one")
	workerTwo := newDedicatedRepositoryWorker(t, ctx, "publication-two")
	workerRepositories := []owned.Repository{
		workerOne.owned(),
		workerTwo.owned(),
	}
	publications := []shared.PreparedRevision{
		preparePublication(t, preparedTree(t, uuid.New()).Input, 2),
		preparePublication(t, preparedTree(t, uuid.New()).Input, 2),
	}
	results := make(chan error, len(publications))
	start := make(chan struct{})
	var workers sync.WaitGroup
	for index, publication := range publications {
		index, publication := index, publication
		workers.Add(1)
		go func() {
			defer workers.Done()
			<-start
			_, publishErr := workerRepositories[index].Publish(
				ctx,
				ownerUID,
				checklistID,
				publication,
				checklistPrecondition(
					checklistID,
					first.Aggregate.SyncVersion,
				),
			)
			results <- publishErr
		}()
	}
	close(start)
	waitForExactBlockedWorkers(
		t,
		ctx,
		observer,
		blockerPID,
		[]int{workerOne.pid, workerTwo.pid},
		[]int{workerOne.pid, workerTwo.pid},
		"user_pmcs_sync_state",
	)
	require.NoError(t, lockTx.Rollback())
	waitForWorkers(t, &workers, 10*time.Second)
	close(results)
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

	blocker, observer := dedicatedConnections(t, ctx)
	lockTx, err := blocker.BeginTx(ctx, nil)
	require.NoError(t, err)
	_, err = lockTx.ExecContext(
		ctx,
		`SELECT current_version
		 FROM user_pmcs_sync_state
		 WHERE user_uid = $1
		 FOR UPDATE`,
		fixture.ownerUID,
	)
	require.NoError(t, err)
	blockerPID := backendPID(t, ctx, blocker)
	workerOne := newDedicatedRepositoryWorker(t, ctx, "release-higher")
	workerTwo := newDedicatedRepositoryWorker(t, ctx, "release-lower")
	workerRepositories := []community.Repository{
		workerOne.community(),
		workerTwo.community(),
	}
	releases := []shared.PreparedRevision{
		fixture.revisions[2],
		fixture.revisions[0],
	}
	results := make(chan error, len(releases))
	start := make(chan struct{})
	var workers sync.WaitGroup
	for index, revision := range releases {
		index, revision := index, revision
		workers.Add(1)
		go func() {
			defer workers.Done()
			<-start
			_, releaseErr := workerRepositories[index].Release(
				ctx,
				fixture.ownerUID,
				fixture.checklist,
				revision.Input.ID,
				checklistPrecondition(
					fixture.checklist,
					second.Aggregate.SyncVersion,
				),
			)
			results <- releaseErr
		}()
	}
	close(start)
	waitForExactBlockedWorkers(
		t,
		ctx,
		observer,
		blockerPID,
		[]int{workerOne.pid, workerTwo.pid},
		[]int{workerOne.pid, workerTwo.pid},
		"user_pmcs_sync_state",
	)
	require.NoError(t, lockTx.Rollback())
	waitForWorkers(t, &workers, 10*time.Second)
	close(results)

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
	drafts := []shared.PreparedRevision{
		preparedTree(t, uuid.New()),
		preparedTree(t, uuid.New()),
	}
	checklistIDs := []uuid.UUID{uuid.New(), uuid.New()}
	blocker, observer := dedicatedConnections(t, ctx)
	lockTx, err := blocker.BeginTx(ctx, nil)
	require.NoError(t, err)
	_, err = lockTx.ExecContext(
		ctx,
		`LOCK TABLE users IN ACCESS EXCLUSIVE MODE`,
	)
	require.NoError(t, err)
	blockerPID := backendPID(t, ctx, blocker)
	workerOne := newDedicatedRepositoryWorker(t, ctx, "first-mutation-one")
	workerTwo := newDedicatedRepositoryWorker(t, ctx, "first-mutation-two")
	workerRepositories := []owned.Repository{
		workerOne.owned(),
		workerTwo.owned(),
	}
	start := make(chan struct{})
	results := make(chan error, 2)
	var workers sync.WaitGroup
	for index := range drafts {
		index := index
		workers.Add(1)
		go func() {
			defer workers.Done()
			<-start
			_, createErr := workerRepositories[index].Create(
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
	waitForExactBlockedWorkers(
		t,
		ctx,
		observer,
		blockerPID,
		[]int{workerOne.pid, workerTwo.pid},
		[]int{workerOne.pid, workerTwo.pid},
		"users",
	)
	require.NoError(t, lockTx.Rollback())
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
	blockerPID := backendPID(t, ctx, blocker)
	blockedWorker := newDedicatedRepositoryWorker(t, ctx, "blocked-user")
	independentWorker := newDedicatedRepositoryWorker(
		t,
		ctx,
		"independent-user",
	)

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
	workerByUID := map[string]dedicatedRepositoryWorker{
		blockedUID:     blockedWorker,
		independentUID: independentWorker,
	}
	var workers sync.WaitGroup
	for _, uid := range []string{blockedUID, independentUID} {
		uid := uid
		workers.Add(1)
		go func() {
			defer workers.Done()
			<-start
			_, createErr := workerByUID[uid].owned().Create(
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
	waitForExactBlockedWorkers(
		t,
		ctx,
		observer,
		blockerPID,
		[]int{blockedWorker.pid, independentWorker.pid},
		[]int{blockedWorker.pid},
		"user_pmcs_sync_state",
	)

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
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
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

	blocker, observer := dedicatedConnections(t, ctx)
	lockTx, err := blocker.BeginTx(ctx, nil)
	require.NoError(t, err)
	_, err = lockTx.ExecContext(
		ctx,
		`LOCK TABLE user_pmcs_subscriptions IN ACCESS EXCLUSIVE MODE`,
	)
	require.NoError(t, err)
	blockerPID := backendPID(t, ctx, blocker)
	snapshotWorker := newDedicatedRepositoryWorker(t, ctx, "delta-snapshot")
	mutationWorker := newDedicatedRepositoryWorker(t, ctx, "delta-mutation")

	type deltaOutcome struct {
		page *userpmcssync.AccountDelta
		err  error
	}
	deltaResults := make(chan deltaOutcome, 1)
	go func() {
		page, deltaErr := snapshotWorker.delta().GetDelta(
			ctx,
			ownerUID,
			0,
			25,
			shared.DefaultConfig().MaxDeltaResponseBytes,
		)
		deltaResults <- deltaOutcome{page: page, err: deltaErr}
	}()
	waitForExactBlockedWorkers(
		t,
		ctx,
		observer,
		blockerPID,
		[]int{snapshotWorker.pid, mutationWorker.pid},
		[]int{snapshotWorker.pid},
		"user_pmcs_account_delta_roots",
	)

	laterChecklistID := uuid.New()
	mutationResults := make(chan error, 1)
	go func() {
		_, mutationErr := mutationWorker.owned().Create(
			ctx,
			ownerUID,
			laterChecklistID,
			preparedTree(t, uuid.New()),
			shared.Precondition{Mode: shared.PreconditionCreate},
		)
		mutationResults <- mutationErr
	}()
	select {
	case mutationErr := <-mutationResults:
		require.NoError(t, mutationErr)
	case <-ctx.Done():
		t.Fatalf("later mutation did not commit while snapshot waited: %v", ctx.Err())
	}
	require.NoError(t, lockTx.Rollback())
	firstOutcome := <-deltaResults
	require.NoError(t, firstOutcome.err)
	firstPage := firstOutcome.page
	require.Len(t, firstPage.Changes, 2)
	require.Equal(t, int64(2), firstPage.AccountVersion)
	require.Equal(t, int64(2), firstPage.ThroughCursor)
	require.False(t, firstPage.HasMore)
	for _, change := range firstPage.Changes {
		require.NotEqual(t, laterChecklistID, change.Checklist.ID)
	}

	nextPage, err := snapshotWorker.delta().GetDelta(
		ctx,
		ownerUID,
		firstPage.ThroughCursor,
		25,
		shared.DefaultConfig().MaxDeltaResponseBytes,
	)
	require.NoError(t, err)
	require.Equal(t, int64(3), nextPage.AccountVersion)
	require.Len(t, nextPage.Changes, 1)
	require.Equal(t, int64(3), nextPage.Changes[0].AccountChangeVersion)
	require.NotNil(t, nextPage.Changes[0].Checklist)
	require.Equal(t, laterChecklistID, nextPage.Changes[0].Checklist.ID)
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

func backendPID(
	t *testing.T,
	ctx context.Context,
	connection *sql.Conn,
) int {
	t.Helper()
	var pid int
	require.NoError(
		t,
		connection.QueryRowContext(ctx, `SELECT pg_backend_pid()`).Scan(&pid),
	)
	return pid
}

func queryBlockedWorkerPIDs(
	ctx context.Context,
	observer *sql.Conn,
	blockerPID int,
	candidatePIDs []int,
	queryToken ...string,
) ([]int, error) {
	token := ""
	if len(queryToken) > 0 {
		token = queryToken[0]
	}
	rows, err := observer.QueryContext(
		ctx,
		`WITH RECURSIVE blocked_chain(pid) AS (
		     SELECT $2::integer
		     UNION
		     SELECT activity.pid
		     FROM pg_stat_activity AS activity
		     JOIN blocked_chain AS blocker
		       ON blocker.pid = ANY(pg_blocking_pids(activity.pid))
		 )
		 SELECT activity.pid
		 FROM pg_stat_activity AS activity
		 JOIN blocked_chain ON blocked_chain.pid = activity.pid
		 WHERE activity.pid = ANY($1)
		   AND activity.wait_event_type = 'Lock'
		   AND ($3 = '' OR activity.query LIKE '%' || $3 || '%')
		 ORDER BY activity.pid`,
		pq.Array(candidatePIDs),
		blockerPID,
		token,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var blocked []int
	for rows.Next() {
		var pid int
		if err := rows.Scan(&pid); err != nil {
			return nil, err
		}
		blocked = append(blocked, pid)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return blocked, rows.Close()
}

func waitForExactBlockedWorkers(
	t *testing.T,
	ctx context.Context,
	observer *sql.Conn,
	blockerPID int,
	candidatePIDs []int,
	expectedPIDs []int,
	queryToken string,
) time.Duration {
	t.Helper()
	expected := append([]int(nil), expectedPIDs...)
	slices.Sort(expected)
	started := time.Now()
	require.Eventually(t, func() bool {
		blocked, err := queryBlockedWorkerPIDs(
			ctx,
			observer,
			blockerPID,
			candidatePIDs,
			queryToken,
		)
		return err == nil && slices.Equal(expected, blocked)
	}, 5*time.Second, 20*time.Millisecond)
	return time.Since(started)
}

type dedicatedRepositoryWorker struct {
	database        *sql.DB
	pid             int
	applicationName string
}

func newDedicatedRepositoryWorker(
	t *testing.T,
	ctx context.Context,
	label string,
) dedicatedRepositoryWorker {
	t.Helper()
	dsn := disableSSLWhenUnspecified(os.Getenv("TEST_DATABASE_URL"))
	connector, err := pq.NewConnector(dsn)
	require.NoError(t, err)
	database := sql.OpenDB(connector)
	database.SetMaxOpenConns(1)
	database.SetMaxIdleConns(1)

	connection, err := database.Conn(ctx)
	require.NoError(t, err)
	applicationName := fmt.Sprintf(
		"upmcs-%s-%s",
		label,
		uuid.NewString()[:8],
	)
	var (
		configuredName string
		pid            int
	)
	require.NoError(
		t,
		connection.QueryRowContext(
			ctx,
			`SELECT set_config('application_name', $1, false),
			        pg_backend_pid()`,
			applicationName,
		).Scan(&configuredName, &pid),
	)
	require.Equal(t, applicationName, configuredName)
	require.NoError(t, connection.Close())
	requireUserPmcsTestDatabase(t, database)

	t.Cleanup(func() {
		require.NoError(t, database.Close())
	})
	return dedicatedRepositoryWorker{
		database:        database,
		pid:             pid,
		applicationName: applicationName,
	}
}

func (worker dedicatedRepositoryWorker) owned() owned.Repository {
	config := shared.DefaultConfig()
	return owned.NewRepository(
		persistence.NewStore(worker.database, config.TransactionMaxAttempts),
		config,
	)
}

func (worker dedicatedRepositoryWorker) community() community.Repository {
	config := shared.DefaultConfig()
	return community.NewRepository(
		persistence.NewStore(worker.database, config.TransactionMaxAttempts),
		config,
	)
}

func (worker dedicatedRepositoryWorker) delta() userpmcssync.Repository {
	config := shared.DefaultConfig()
	return userpmcssync.NewRepository(
		persistence.NewStore(worker.database, config.TransactionMaxAttempts),
	)
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
