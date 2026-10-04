package shops_test

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"miltechserver/api/shops/aggregates"
	"miltechserver/api/shops/shared"
	"miltechserver/api/shops/vehicles/notifications"
	"miltechserver/bootstrap"
	"miltechserver/tests/testutil"

	"github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

// The driver barrier runs only after the previous query has been consumed. Its
// writer uses a separate physical connection and commits before this query runs.
// No production hooks or timing assumptions are needed to reproduce a torn read.
type aggregateSnapshotProbe struct {
	mutex              sync.Mutex
	queries            []string
	options            []driver.TxOptions
	before             func(context.Context, string) error
	outsideTransaction int
	maxParameters      int
}
type aggregateSnapshotConnector struct {
	base  driver.Connector
	probe *aggregateSnapshotProbe
}

func (c *aggregateSnapshotConnector) Driver() driver.Driver { return c.base.Driver() }
func (c *aggregateSnapshotConnector) Connect(ctx context.Context) (driver.Conn, error) {
	conn, err := c.base.Connect(ctx)
	if err != nil {
		return nil, err
	}
	return &aggregateSnapshotConnection{Conn: conn, probe: c.probe}, nil
}

type aggregateSnapshotConnection struct {
	driver.Conn
	probe  *aggregateSnapshotProbe
	active bool
}

func (c *aggregateSnapshotConnection) BeginTx(ctx context.Context, options driver.TxOptions) (driver.Tx, error) {
	c.probe.mutex.Lock()
	c.probe.options = append(c.probe.options, options)
	c.probe.mutex.Unlock()
	tx, err := c.Conn.(driver.ConnBeginTx).BeginTx(ctx, options)
	if err != nil {
		return nil, err
	}
	c.probe.mutex.Lock()
	c.active = true
	c.probe.mutex.Unlock()
	return &aggregateSnapshotTransaction{Tx: tx, connection: c}, nil
}

type aggregateSnapshotTransaction struct {
	driver.Tx
	connection *aggregateSnapshotConnection
}

func (tx *aggregateSnapshotTransaction) finish() {
	tx.connection.probe.mutex.Lock()
	tx.connection.active = false
	tx.connection.probe.mutex.Unlock()
}
func (tx *aggregateSnapshotTransaction) Commit() error   { defer tx.finish(); return tx.Tx.Commit() }
func (tx *aggregateSnapshotTransaction) Rollback() error { defer tx.finish(); return tx.Tx.Rollback() }

func (c *aggregateSnapshotConnection) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	c.probe.mutex.Lock()
	c.probe.queries = append(c.probe.queries, query)
	if (strings.Contains(query, "shop_notification_items") || strings.Contains(query, "WITH ranked_equipment")) && len(args) > c.probe.maxParameters {
		c.probe.maxParameters = len(args)
	}
	if !c.active {
		c.probe.outsideTransaction++
	}
	c.probe.mutex.Unlock()
	if c.probe.before != nil {
		if err := c.probe.before(ctx, query); err != nil {
			return nil, err
		}
	}
	return c.Conn.(driver.QueryerContext).QueryContext(ctx, query, args)
}
func aggregateSnapshotDatabase(t *testing.T, probe *aggregateSnapshotProbe) *sql.DB {
	t.Helper()
	parsed, err := url.Parse(os.Getenv("TEST_DATABASE_URL"))
	require.NoError(t, err)
	query := parsed.Query()
	if query.Get("sslmode") == "" {
		query.Set("sslmode", "disable")
		parsed.RawQuery = query.Encode()
	}
	connector, err := pq.NewConnector(parsed.String())
	require.NoError(t, err)
	db := sql.OpenDB(&aggregateSnapshotConnector{base: connector, probe: probe})
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	return db
}
func requireAggregateSnapshot(t *testing.T, probe *aggregateSnapshotProbe) {
	t.Helper()
	require.Zero(t, probe.outsideTransaction, "every component must use the same transaction")
	require.Len(t, probe.options, 1, "aggregate must own exactly one transaction")
	require.Equal(t, driver.IsolationLevel(sql.LevelRepeatableRead), probe.options[0].Isolation)
	require.True(t, probe.options[0].ReadOnly)
}

func TestAggregateSingleSnapshot(t *testing.T) {
	for _, entry := range []string{"shop", "vehicle", "legacy"} {
		t.Run(entry, func(t *testing.T) {
			clearShopTables(t, testDB)
			ensureUser(t, testDB, "user-1")
			router := newTestRouter(t)
			shopID := createShop(t, router, "user-1", "old")
			vehicleID := createVehicle(t, router, "user-1", shopID)
			notificationID := createNotificationRow(t, shopID, vehicleID, "old", time.Now())
			_, err := testDB.Exec(`INSERT INTO shop_notification_items(id,shop_id,notification_id,niin,nomenclature,quantity,save_time) VALUES('snapshot-item',$1,$2,'123456789','old',1,NOW())`, shopID, notificationID)
			require.NoError(t, err)
			committed := false
			probe := &aggregateSnapshotProbe{before: func(ctx context.Context, query string) error {
				if committed || !strings.Contains(query, "shop_notification_items") {
					return nil
				}
				tx, err := testDB.BeginTx(ctx, nil)
				if err != nil {
					return err
				}
				defer tx.Rollback()
				if _, err = tx.ExecContext(ctx, `UPDATE shop_vehicle_notifications SET title='new' WHERE id=$1`, notificationID); err != nil {
					return err
				}
				if _, err = tx.ExecContext(ctx, `UPDATE shop_notification_items SET quantity=2 WHERE id='snapshot-item'`); err != nil {
					return err
				}
				if _, err = tx.ExecContext(ctx, `DELETE FROM shop_members WHERE shop_id=$1 AND user_id='user-1'`, shopID); err != nil {
					return err
				}
				if err = tx.Commit(); err != nil {
					return err
				}
				committed = true
				return nil
			}}
			repo := aggregates.NewRepository(aggregateSnapshotDatabase(t, probe))
			user := &bootstrap.User{UserID: "user-1"}
			if entry == "shop" {
				result, err := repo.GetShopSnapshot(context.Background(), user, shopID, aggregates.ShopSnapshotOptions{Includes: map[string]bool{"notifications": true}})
				require.NoError(t, err)
				require.True(t, committed)
				require.Len(t, result.Notifications, 1)
				require.Equal(t, "old", result.Notifications[0].Notification.Title)
				require.EqualValues(t, 1, result.Notifications[0].Items[0].Quantity, "old header must not be paired with newly committed items")
				require.EqualValues(t, 1, result.Shop.Counts.Notifications)
				requireAggregateSnapshot(t, probe)
				_, err = repo.GetShopSnapshot(context.Background(), user, shopID, aggregates.ShopSnapshotOptions{})
				require.ErrorIs(t, err, shared.ErrShopAccessDenied)
			} else if entry == "legacy" {
				legacy := notifications.NewService(notifications.NewRepository(aggregateSnapshotDatabase(t, probe)), nil)
				result, err := legacy.GetVehicleNotificationsWithItems(context.Background(), user, vehicleID)
				require.NoError(t, err)
				require.True(t, committed)
				require.Len(t, result, 1)
				require.Equal(t, "old", result[0].Notification.Title)
				require.EqualValues(t, 1, result[0].Items[0].Quantity)
				requireAggregateSnapshot(t, probe)
				_, err = legacy.GetVehicleNotificationsWithItems(context.Background(), user, vehicleID)
				require.Error(t, err)
			} else {
				service := aggregates.NewService(repo, shared.NewShopAuthorization(testDB))
				result, err := service.GetVehicleMaintenanceSnapshot(context.Background(), user, vehicleID, aggregates.SnapshotLimits{})
				require.NoError(t, err)
				require.True(t, committed)
				require.Len(t, result.Notifications, 1)
				require.Equal(t, "old", result.Notifications[0].Notification.Title)
				require.EqualValues(t, 1, result.Notifications[0].Items[0].Quantity)
				require.EqualValues(t, 1, result.Counts.NotificationItems)
				requireAggregateSnapshot(t, probe)
				_, err = service.GetVehicleMaintenanceSnapshot(context.Background(), user, vehicleID, aggregates.SnapshotLimits{})
				require.ErrorIs(t, err, aggregates.ErrAccessDenied)
			}
		})
	}
	t.Run("bootstrap", func(t *testing.T) {
		clearShopTables(t, testDB)
		ensureUser(t, testDB, "user-1")
		router := newTestRouter(t)
		shopID := createShop(t, router, "user-1", "old")
		vehicleID := createVehicle(t, router, "user-1", shopID)
		committed := false
		probe := &aggregateSnapshotProbe{before: func(ctx context.Context, query string) error {
			if committed || !strings.Contains(query, "WITH ranked_equipment") {
				return nil
			}
			committed = true
			_, err := testDB.ExecContext(ctx, `DELETE FROM shop_vehicle WHERE id=$1`, vehicleID)
			return err
		}}
		result, err := aggregates.NewRepository(aggregateSnapshotDatabase(t, probe)).GetBootstrap(context.Background(), &bootstrap.User{UserID: "user-1"}, aggregates.BootstrapOptions{})
		require.NoError(t, err)
		require.True(t, committed)
		require.Len(t, result, 1)
		require.EqualValues(t, 1, result[0].Counts.Vehicles)
		require.Len(t, result[0].Equipment, 1)
		requireAggregateSnapshot(t, probe)
	})
	t.Run("pmcs counts", func(t *testing.T) {
		clearShopTables(t, testDB)
		ensureUser(t, testDB, "user-1")
		router := newTestRouter(t)
		shopID := createShop(t, router, "user-1", "old")
		vehicleID := createVehicle(t, router, "user-1", shopID)
		inspectionID := createPmcsInspection(t, testDB, vehicleID, "pmcs_sbs/hmmwv/hmmwv_up_armor_pmcs.json", time.Now(), "user-1")
		committed := false
		probe := &aggregateSnapshotProbe{before: func(ctx context.Context, query string) error {
			if committed || !strings.Contains(query, "FROM public.user_pmcs_faults") {
				return nil
			}
			committed = true
			createPmcsFault(t, testDB, inspectionID, "before", 0)
			createPmcsComment(t, testDB, inspectionID, "user-1", "new")
			return nil
		}}
		result, err := aggregates.NewRepository(aggregateSnapshotDatabase(t, probe)).GetEquipmentPmcsHistory(context.Background(), &bootstrap.User{UserID: "user-1"})
		require.NoError(t, err)
		require.True(t, committed)
		require.Len(t, result, 1)
		require.Len(t, result[0].HistoricalPmcs, 1)
		require.Zero(t, result[0].HistoricalPmcs[0].FaultCount)
		require.Zero(t, result[0].HistoricalPmcs[0].CommentCount)
		requireAggregateSnapshot(t, probe)
	})
}

func TestAggregateAuthorsBatched(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "user-1")
	ensureUser(t, testDB, "user-2")
	router := newTestRouter(t)
	shopID := createShop(t, router, "user-1", "authors")
	vehicleID := createVehicle(t, router, "user-1", shopID)
	_, err := testDB.Exec(`UPDATE users SET username=CASE uid WHEN 'user-1' THEN 'First' ELSE 'Second' END WHERE uid IN ('user-1','user-2')`)
	require.NoError(t, err)
	createEquipmentService(t, "user-1", shopID, vehicleID, "", "first", nil, false)
	createEquipmentService(t, "user-2", shopID, vehicleID, "", "second", nil, false)
	renamed := false
	probe := &aggregateSnapshotProbe{before: func(ctx context.Context, query string) error {
		if renamed || !strings.Contains(query, "FROM equipment_services es\n") {
			return nil
		}
		renamed = true
		_, err := testDB.ExecContext(ctx, `UPDATE users SET username='New Name' WHERE uid IN ('user-1','user-2')`)
		return err
	}}
	repo := aggregates.NewRepository(aggregateSnapshotDatabase(t, probe))
	user := &bootstrap.User{UserID: "user-1"}
	result, err := repo.GetShopSnapshot(context.Background(), user, shopID, aggregates.ShopSnapshotOptions{Includes: map[string]bool{"services": true}})
	require.NoError(t, err)
	require.Len(t, result.Services, 2)
	require.True(t, renamed)
	requireAggregateSnapshot(t, probe)
	require.ElementsMatch(t, []string{"First", "Second"}, []string{result.Services[0].CreatedByUsername, result.Services[1].CreatedByUsername})
	independentAuthors := 0
	for _, query := range probe.queries {
		if strings.Contains(strings.ToLower(query), "from users") {
			independentAuthors++
		}
	}
	require.LessOrEqual(t, independentAuthors, 1)
	t.Logf("component queries=%d independent author queries=%d", len(probe.queries), independentAuthors)
	failure := errors.New("author query failed")
	probe.before = func(_ context.Context, query string) error {
		if strings.Contains(query, "LEFT JOIN users") {
			return failure
		}
		return nil
	}
	_, err = repo.GetShopSnapshot(context.Background(), user, shopID, aggregates.ShopSnapshotOptions{Includes: map[string]bool{"services": true}})
	require.ErrorIs(t, err, failure)
	ctx, cancel := context.WithCancel(context.Background())
	probe.before = func(queryCtx context.Context, query string) error {
		if strings.Contains(query, "LEFT JOIN users") {
			require.Same(t, ctx, queryCtx)
			cancel()
		}
		return nil
	}
	_, err = repo.GetShopSnapshot(ctx, user, shopID, aggregates.ShopSnapshotOptions{Includes: map[string]bool{"services": true}})
	require.Error(t, err)
	require.ErrorIs(t, ctx.Err(), context.Canceled)
	cancel()
	_, err = repo.GetShopSnapshot(ctx, user, shopID, aggregates.ShopSnapshotOptions{})
	require.ErrorIs(t, err, context.Canceled)
}

func TestAggregateListSnapshotMembership(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "user-1")
	router := newTestRouter(t)
	shopID := createShop(t, router, "user-1", "lists")
	listID := createList(t, router, "user-1", shopID)
	createListItem(t, router, "user-1", listID, "123456789", "part")
	removed := false
	probe := &aggregateSnapshotProbe{before: func(ctx context.Context, query string) error {
		if removed || !strings.Contains(query, "WITH ranked_lists") {
			return nil
		}
		removed = true
		_, err := testDB.ExecContext(ctx, `DELETE FROM shop_members WHERE shop_id=$1 AND user_id='user-1'`, shopID)
		return err
	}}
	repo := aggregates.NewRepository(aggregateSnapshotDatabase(t, probe))
	user := &bootstrap.User{UserID: "user-1"}
	lists, err := repo.GetListsWithItems(context.Background(), user, shopID, aggregates.ListTreeLimits{})
	require.NoError(t, err)
	require.True(t, removed)
	require.Len(t, lists, 1)
	require.Len(t, lists[0].Items, 1)
	requireAggregateSnapshot(t, probe)
	_, err = repo.GetListsWithItems(context.Background(), user, shopID, aggregates.ListTreeLimits{})
	require.ErrorIs(t, err, shared.ErrShopAccessDenied)
}

func TestAggregateContext(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "user-1")
	router := newTestRouter(t)
	shopID := createShop(t, router, "user-1", "context")
	vehicleID := createVehicle(t, router, "user-1", shopID)
	db, err := testutil.OpenDisposableTestDB(os.Getenv("TEST_DATABASE_URL"), os.Getenv("TEST_DATABASE_MARKER"))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	db.SetMaxOpenConns(1)
	repo := aggregates.NewRepository(db)
	legacy := notifications.NewService(notifications.NewRepository(db), nil)
	cases := []struct {
		name string
		call func(context.Context, *bootstrap.User) error
	}{
		{"legacy", func(ctx context.Context, user *bootstrap.User) error {
			_, err := legacy.GetVehicleNotificationsWithItems(ctx, user, vehicleID)
			return err
		}},
		{"shop", func(ctx context.Context, user *bootstrap.User) error {
			_, err := repo.GetShopSnapshot(ctx, user, shopID, aggregates.ShopSnapshotOptions{})
			return err
		}},
		{"vehicle", func(ctx context.Context, user *bootstrap.User) error {
			_, err := repo.GetVehicleMaintenanceSnapshot(ctx, user, vehicleID, aggregates.SnapshotLimits{})
			return err
		}},
		{"lists", func(ctx context.Context, user *bootstrap.User) error {
			_, err := repo.GetListsWithItems(ctx, user, shopID, aggregates.ListTreeLimits{})
			return err
		}},
		{"bootstrap", func(ctx context.Context, user *bootstrap.User) error {
			_, err := repo.GetBootstrap(ctx, user, aggregates.BootstrapOptions{})
			return err
		}},
		{"pmcs", func(ctx context.Context, user *bootstrap.User) error {
			_, err := repo.GetEquipmentPmcsHistory(ctx, user)
			return err
		}},
	}
	user := &bootstrap.User{UserID: "user-1"}
	for _, tc := range cases {
		t.Run(tc.name+"/nil-user", func(t *testing.T) {
			if tc.name == "legacy" {
				require.EqualError(t, tc.call(context.Background(), nil), "unauthorized user")
				return
			}
			require.ErrorIs(t, tc.call(context.Background(), nil), aggregates.ErrUnauthorized)
		})
		t.Run(tc.name+"/canceled", func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			require.ErrorIs(t, tc.call(ctx, user), context.Canceled)
		})
		t.Run(tc.name+"/pool", func(t *testing.T) {
			conn, err := db.Conn(context.Background())
			require.NoError(t, err)
			defer conn.Close()
			waits := db.Stats().WaitCount
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			done := make(chan error, 1)
			go func() { done <- tc.call(ctx, user) }()
			require.Eventually(t, func() bool { return db.Stats().WaitCount > waits }, time.Second, time.Millisecond)
			cancel()
			select {
			case err := <-done:
				require.ErrorIs(t, err, context.Canceled)
			case <-time.After(time.Second):
				t.Fatal("aggregate did not cancel pool wait")
			}
		})
		t.Run(tc.name+"/blocked-query", func(t *testing.T) {
			blocker, err := testDB.BeginTx(context.Background(), nil)
			require.NoError(t, err)
			defer blocker.Rollback()
			_, err = blocker.Exec(`LOCK TABLE shop_members IN ACCESS EXCLUSIVE MODE`)
			require.NoError(t, err)
			var blockerPID int
			require.NoError(t, blocker.QueryRow(`SELECT pg_backend_pid()`).Scan(&blockerPID))
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			done := make(chan error, 1)
			go func() { done <- tc.call(ctx, user) }()
			require.Eventually(t, func() bool {
				var blocked bool
				err := testDB.QueryRow(`SELECT EXISTS(SELECT 1 FROM pg_stat_activity WHERE $1=ANY(pg_blocking_pids(pid)))`, blockerPID).Scan(&blocked)
				return err == nil && blocked
			}, time.Second, time.Millisecond)
			cancel()
			select {
			case err := <-done:
				require.Error(t, err)
			case <-time.After(time.Second):
				t.Fatal("aggregate did not cancel physical query")
			}
			require.Eventually(t, func() bool { return db.Stats().InUse == 0 }, time.Second, time.Millisecond)
		})
	}
}
