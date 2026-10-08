package user_saves_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/user_saves/categories"
	categoryitems "miltechserver/api/user_saves/categories/items"
	"miltechserver/bootstrap"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// categoriesDeadlockFixture seeds one category with one categorized item so
// a helper connection can lock the two tables in the opposite order used by
// categories.Delete / categories.DeleteAll (category row first, then the
// categorized item row), forcing Postgres to detect a genuine deadlock and
// abort one of the two transactions.
type categoriesDeadlockFixture struct {
	user     *bootstrap.User
	category model.UserItemCategory
	item     model.UserItemsCategorized
}

func seedCategoriesDeadlockFixture(t *testing.T, userID string) categoriesDeadlockFixture {
	t.Helper()
	clearUserSavesTables(t, testDB)

	user := &bootstrap.User{UserID: userID}
	ensureUser(t, testDB, user.UserID)

	now := time.Now().UTC()
	comment := ""
	image := ""
	category := model.UserItemCategory{
		ID:          uuid.New().String(),
		UserUID:     user.UserID,
		Name:        "Deadlock Category",
		Comment:     &comment,
		Image:       &image,
		LastUpdated: &now,
	}

	categoryRepo := categories.NewRepository(testDB)
	require.NoError(t, categoryRepo.Upsert(user, category))

	itemName := "Deadlock Item"
	quantity := int32(1)
	equipModel := "Model"
	uoc := "UOC"
	nickname := ""
	item := model.UserItemsCategorized{
		ID:          uuid.New().String(),
		UserID:      user.UserID,
		Niin:        "DL-1",
		CategoryID:  category.ID,
		ItemName:    &itemName,
		Quantity:    &quantity,
		EquipModel:  &equipModel,
		Uoc:         &uoc,
		SaveTime:    &now,
		Image:       &image,
		LastUpdated: &now,
		Nickname:    &nickname,
	}
	itemsRepo := categoryitems.NewRepository(testDB)
	require.NoError(t, itemsRepo.Upsert(user, item))

	return categoriesDeadlockFixture{user: user, category: category, item: item}
}

// dedicatedCategoriesConnection opens a connection to testDB independent of
// the repositories under test, so the helper transaction's row locks are
// held on a separate backend and can genuinely block/deadlock against the
// repository method's own transaction.
func dedicatedCategoriesConnection(t *testing.T, ctx context.Context) *sql.Conn {
	t.Helper()
	conn, err := testDB.Conn(ctx)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = conn.Close()
	})
	return conn
}

// categoriesBackendPID returns the Postgres backend PID for a connection,
// so we can watch pg_stat_activity for that specific backend blocking on a
// lock.
func categoriesBackendPID(t *testing.T, ctx context.Context, conn *sql.Conn) int {
	t.Helper()
	var pid int
	require.NoError(t, conn.QueryRowContext(ctx, `SELECT pg_backend_pid()`).Scan(&pid))
	return pid
}

// requireCategoriesBackendBlockedOnLock polls pg_stat_activity until the
// given backend PID is reported as waiting on a lock, using a dedicated
// observer connection so the poll itself never contends for the same locks.
func requireCategoriesBackendBlockedOnLock(t *testing.T, ctx context.Context, observer *sql.Conn, pid int) {
	t.Helper()
	require.Eventually(t, func() bool {
		var waitEventType sql.NullString
		err := observer.QueryRowContext(
			ctx,
			`SELECT wait_event_type FROM pg_stat_activity WHERE pid = $1`,
			pid,
		).Scan(&waitEventType)
		return err == nil && waitEventType.Valid && waitEventType.String == "Lock"
	}, 5*time.Second, 20*time.Millisecond, "backend %d never showed as lock-blocked", pid)
}

// TestCategoriesDeleteRollsBackOnDeadlock forces categories.Delete's second
// statement (the user_item_category DELETE) to fail with a genuine deadlock
// by holding the two row locks in the opposite order on a separate
// connection. Postgres's deadlock detector aborts one of the two
// transactions; we assert Delete surfaces an error and that neither the
// categorized item row nor the category row was removed, proving the first
// DELETE inside Delete's transaction rolled back too.
func TestCategoriesDeleteRollsBackOnDeadlock(t *testing.T) {
	fixture := seedCategoriesDeadlockFixture(t, "cat-deadlock-delete")
	categoryRepo := categories.NewRepository(testDB)
	itemsRepo := categoryitems.NewRepository(testDB)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	observer := dedicatedCategoriesConnection(t, ctx)
	blocker := dedicatedCategoriesConnection(t, ctx)
	blockerPID := categoriesBackendPID(t, ctx, blocker)

	blockerTx, err := blocker.BeginTx(ctx, nil)
	require.NoError(t, err)

	// Step 1: blocker locks the category row -- the second row Delete's
	// transaction will need.
	_, err = blockerTx.ExecContext(
		ctx,
		`SELECT id FROM user_item_category WHERE id = $1 AND user_uid = $2 FOR UPDATE`,
		fixture.category.ID, fixture.user.UserID,
	)
	require.NoError(t, err)

	deleteErrCh := make(chan error, 1)
	go func() {
		deleteErrCh <- categoryRepo.Delete(fixture.user, fixture.category)
	}()

	// Step 2: once Delete's own transaction has acquired the
	// user_items_categorized lock (its first statement) and moved on to
	// wait on the category row blocker already holds, have blocker attempt
	// to lock user_items_categorized too. That closes the wait-for cycle:
	// Delete waits on blocker's category lock, blocker waits on Delete's
	// items lock -- Postgres's deadlock detector must abort one side.
	requireDeleteBlockedOnCategoryLock(t, ctx, observer, fixture)

	blockerLockErrCh := make(chan error, 1)
	go func() {
		_, lockErr := blockerTx.ExecContext(
			ctx,
			`SELECT id FROM user_items_categorized WHERE category_id = $1 FOR UPDATE`,
			fixture.category.ID,
		)
		blockerLockErrCh <- lockErr
	}()

	requireCategoriesBackendBlockedOnLock(t, ctx, observer, blockerPID)

	var deleteErr, blockerLockErr error
	select {
	case deleteErr = <-deleteErrCh:
	case <-time.After(10 * time.Second):
		t.Fatal("categories.Delete did not complete after deadlock cycle formed")
	}
	select {
	case blockerLockErr = <-blockerLockErrCh:
	case <-time.After(10 * time.Second):
		t.Fatal("blocker transaction did not complete after deadlock cycle formed")
	}
	require.NoError(t, blockerTx.Rollback())

	// Exactly one side of the cycle must have failed with a deadlock; the
	// other proceeds normally. Assert Delete's outcome is consistent with
	// the persisted state either way.
	if deleteErr != nil {
		require.ErrorContains(t, deleteErr, "error deleting")
		require.Nil(t, blockerLockErr, "blocker should have won the deadlock and completed its lock")

		items, err := itemsRepo.GetByUser(fixture.user)
		require.NoError(t, err)
		require.Len(t, items, 1, "rollback must leave the categorized item untouched")

		cats, err := categoryRepo.GetByUser(fixture.user)
		require.NoError(t, err)
		require.Len(t, cats, 1, "rollback must leave the category untouched")
	} else {
		require.Error(t, blockerLockErr, "blocker should have been the deadlock victim if Delete succeeded")
	}
}

// requireDeleteBlockedOnCategoryLock polls until some backend other than
// the observer/blocker is reported as lock-waiting -- i.e. Delete's
// transaction has taken its own user_items_categorized lock and is now
// waiting on the user_item_category row the blocker holds.
func requireDeleteBlockedOnCategoryLock(t *testing.T, ctx context.Context, observer *sql.Conn, fixture categoriesDeadlockFixture) {
	t.Helper()
	require.Eventually(t, func() bool {
		var count int
		err := observer.QueryRowContext(
			ctx,
			`SELECT count(*) FROM pg_stat_activity
			 WHERE wait_event_type = 'Lock'
			   AND query LIKE '%user_item_category%'
			   AND query LIKE '%DELETE%'`,
		).Scan(&count)
		return err == nil && count > 0
	}, 5*time.Second, 20*time.Millisecond, "categories.Delete never blocked waiting on the category row lock")
}

// TestCategoriesDeleteAllRollsBackOnDeadlock mirrors
// TestCategoriesDeleteRollsBackOnDeadlock for DeleteAll, which deletes all
// of a user's categorized items and then all of their categories in a
// single transaction.
func TestCategoriesDeleteAllRollsBackOnDeadlock(t *testing.T) {
	fixture := seedCategoriesDeadlockFixture(t, "cat-deadlock-delete-all")
	categoryRepo := categories.NewRepository(testDB)
	itemsRepo := categoryitems.NewRepository(testDB)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	observer := dedicatedCategoriesConnection(t, ctx)
	blocker := dedicatedCategoriesConnection(t, ctx)
	blockerPID := categoriesBackendPID(t, ctx, blocker)

	blockerTx, err := blocker.BeginTx(ctx, nil)
	require.NoError(t, err)

	_, err = blockerTx.ExecContext(
		ctx,
		`SELECT id FROM user_item_category WHERE user_uid = $1 FOR UPDATE`,
		fixture.user.UserID,
	)
	require.NoError(t, err)

	deleteAllErrCh := make(chan error, 1)
	go func() {
		deleteAllErrCh <- categoryRepo.DeleteAll(fixture.user)
	}()

	require.Eventually(t, func() bool {
		var count int
		err := observer.QueryRowContext(
			ctx,
			`SELECT count(*) FROM pg_stat_activity
			 WHERE wait_event_type = 'Lock'
			   AND query LIKE '%user_item_category%'
			   AND query LIKE '%DELETE%'`,
		).Scan(&count)
		return err == nil && count > 0
	}, 5*time.Second, 20*time.Millisecond, "categories.DeleteAll never blocked waiting on the category row lock")

	blockerLockErrCh := make(chan error, 1)
	go func() {
		_, lockErr := blockerTx.ExecContext(
			ctx,
			`SELECT id FROM user_items_categorized WHERE user_id = $1 FOR UPDATE`,
			fixture.user.UserID,
		)
		blockerLockErrCh <- lockErr
	}()

	requireCategoriesBackendBlockedOnLock(t, ctx, observer, blockerPID)

	var deleteAllErr, blockerLockErr error
	select {
	case deleteAllErr = <-deleteAllErrCh:
	case <-time.After(10 * time.Second):
		t.Fatal("categories.DeleteAll did not complete after deadlock cycle formed")
	}
	select {
	case blockerLockErr = <-blockerLockErrCh:
	case <-time.After(10 * time.Second):
		t.Fatal("blocker transaction did not complete after deadlock cycle formed")
	}
	require.NoError(t, blockerTx.Rollback())

	if deleteAllErr != nil {
		require.ErrorContains(t, deleteAllErr, "error deleting all")
		require.Nil(t, blockerLockErr, "blocker should have won the deadlock and completed its lock")

		items, err := itemsRepo.GetByUser(fixture.user)
		require.NoError(t, err)
		require.Len(t, items, 1, "rollback must leave the categorized item untouched")

		cats, err := categoryRepo.GetByUser(fixture.user)
		require.NoError(t, err)
		require.Len(t, cats, 1, "rollback must leave the category untouched")
	} else {
		require.Error(t, blockerLockErr, "blocker should have been the deadlock victim if DeleteAll succeeded")
	}
}
