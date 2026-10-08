package messages

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"miltechserver/api/shops/shared"
	"miltechserver/bootstrap"
	"strings"
	"testing"
	"time"
)

const syncShop = "11111111-1111-4111-8111-111111111111"
const syncID = "22222222-2222-4222-8222-222222222222"
const syncOtherID = "33333333-3333-4333-8333-333333333333"

var syncUser = &bootstrap.User{UserID: "member"}

type syncQuery struct {
	contains string
	args     []driver.Value
	rows     [][]driver.Value
	err      error
}
type syncConn struct {
	t          *testing.T
	queries    []syncQuery
	committed  bool
	rolledBack bool
}
type syncConnector struct{ c *syncConn }

func (c syncConnector) Connect(context.Context) (driver.Conn, error) { return c.c, nil }
func (syncConnector) Driver() driver.Driver                          { return syncDriver{} }

type syncDriver struct{}

func (syncDriver) Open(string) (driver.Conn, error)   { return nil, errors.New("use connector") }
func (*syncConn) Prepare(string) (driver.Stmt, error) { return nil, errors.New("unexpected prepare") }
func (*syncConn) Close() error                        { return nil }
func (*syncConn) Begin() (driver.Tx, error)           { return nil, errors.New("use BeginTx") }
func (c *syncConn) BeginTx(_ context.Context, o driver.TxOptions) (driver.Tx, error) {
	if o.Isolation != driver.IsolationLevel(sql.LevelRepeatableRead) || !o.ReadOnly {
		c.t.Fatal("read must use read-only repeatable read")
	}
	return c, nil
}
func (c *syncConn) Commit() error   { c.committed = true; return nil }
func (c *syncConn) Rollback() error { c.rolledBack = true; return nil }
func (c *syncConn) QueryContext(_ context.Context, q string, args []driver.NamedValue) (driver.Rows, error) {
	if len(c.queries) == 0 {
		return nil, fmt.Errorf("unexpected query: %s", q)
	}
	expected := c.queries[0]
	c.queries = c.queries[1:]
	if !strings.Contains(q, expected.contains) {
		return nil, fmt.Errorf("query missing %q: %s", expected.contains, q)
	}
	if expected.args != nil {
		if len(args) != len(expected.args) {
			c.t.Fatalf("args %v expected %v", args, expected.args)
		}
		for i, a := range args {
			if fmt.Sprint(a.Value) != fmt.Sprint(expected.args[i]) {
				c.t.Fatalf("arg %d: %v expected %v", i, a.Value, expected.args[i])
			}
		}
	}
	if expected.err != nil {
		return nil, expected.err
	}
	count := 1
	if len(expected.rows) > 0 {
		count = len(expected.rows[0])
	}
	return &syncRows{rows: expected.rows, count: count}, nil
}

type syncRows struct {
	rows  [][]driver.Value
	count int
}

func (r *syncRows) Columns() []string {
	names := make([]string, r.count)
	for i := range names {
		names[i] = fmt.Sprint(i)
	}
	return names
}
func (*syncRows) Close() error { return nil }
func (r *syncRows) Next(dest []driver.Value) error {
	if len(r.rows) == 0 {
		return io.EOF
	}
	copy(dest, r.rows[0])
	r.rows = r.rows[1:]
	return nil
}
func syncRepo(t *testing.T, queries ...syncQuery) (*RepositoryImpl, *syncConn) {
	t.Helper()
	c := &syncConn{t: t, queries: queries}
	db := sql.OpenDB(syncConnector{c})
	t.Cleanup(func() {
		db.Close()
		if len(c.queries) != 0 {
			t.Errorf("%d unconsumed queries", len(c.queries))
		}
	})
	return NewRepository(db, nil, nil), c
}
func syncMembership(member bool) syncQuery {
	return syncQuery{contains: "SELECT EXISTS", args: []driver.Value{syncShop, "member"}, rows: [][]driver.Value{{member}}}
}
func syncCounter(n int64) syncQuery {
	return syncQuery{contains: "SELECT last_number FROM public.shop_message_counters", args: []driver.Value{syncShop}, rows: [][]driver.Value{{n, int64(1), n, false}}}
}
func syncRow(id string, n int64) []driver.Value {
	return []driver.Value{id, syncShop, "member", "body", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), nil, false, nil, "Member", n}
}

func TestMessageSyncReaderCatchUpGapsAndBound(t *testing.T) {
	repo, c := syncRepo(t, syncMembership(true), syncCounter(9), syncQuery{contains: "m.insertion_number > $2 AND m.insertion_number <= $3", args: []driver.Value{syncShop, int64(2), int64(8), int64(2)}, rows: [][]driver.Value{syncRow(syncID, 4), syncRow(syncOtherID, 7)}})
	bound := "8"
	page, err := repo.CatchUpMessages(context.Background(), syncUser, syncShop, "2", &bound, 1)
	if err != nil {
		t.Fatal(err)
	}
	if page.NextAfter != "4" || page.Through != "8" || !page.HasMore || len(page.Rows) != 1 || !c.committed {
		t.Fatalf("page=%+v commit=%v", page, c.committed)
	}
}
func TestMessageSyncReaderAllDeletedAdvancesToBound(t *testing.T) {
	repo, _ := syncRepo(t, syncMembership(true), syncCounter(9), syncQuery{contains: "ORDER BY m.insertion_number ASC", rows: [][]driver.Value{}})
	page, err := repo.CatchUpMessages(context.Background(), syncUser, syncShop, "2", nil, 100)
	if err != nil {
		t.Fatal(err)
	}
	if page.NextAfter != "9" || page.Through != "9" || page.HasMore || page.Rows == nil {
		t.Fatalf("%+v", page)
	}
}
func TestMessageSyncReaderRejectsFutureBound(t *testing.T) {
	repo, c := syncRepo(t, syncMembership(true), syncCounter(9))
	bound := "10"
	page, err := repo.CatchUpMessages(context.Background(), syncUser, syncShop, "2", &bound, 100)
	if err == nil || page != nil || c.committed || !c.rolledBack {
		t.Fatalf("page=%v err=%v", page, err)
	}
}
func TestMessageSyncReaderDenialDoesNotReportMissing(t *testing.T) {
	repo, c := syncRepo(t, syncMembership(false))
	page, err := repo.ReconcileMessages(context.Background(), syncUser, syncShop, []string{syncID})
	if err == nil || page != nil || c.committed || !c.rolledBack {
		t.Fatalf("page=%v err=%v", page, err)
	}
}
func TestMessageSyncReaderReconcileScopedAndDeduplicated(t *testing.T) {
	repo, _ := syncRepo(t, syncMembership(true), syncCounter(1), syncQuery{contains: "m.shop_id = $1 AND m.id IN ($2,$3)", args: []driver.Value{syncShop, syncID, syncOtherID}, rows: [][]driver.Value{syncRow(syncID, 1)}})
	page, err := repo.ReconcileMessages(context.Background(), syncUser, syncShop, []string{syncID, syncOtherID, syncID})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Rows) != 1 || len(page.MissingIDs) != 1 || page.MissingIDs[0] != syncOtherID || page.Rows[0].ParentID != nil {
		t.Fatalf("%+v", page)
	}
}
func TestMessageSyncReaderDeletedHistoryAnchor(t *testing.T) {
	row := syncRow(syncID, 1)
	timestamp := row[4].(time.Time)
	cursor := "eyJ2IjoyLCJzaG9wX2lkIjoiMTExMTExMTEtMTExMS00MTExLTgxMTEtMTExMTExMTExMTExIiwiY3JlYXRlZF9hdCI6IjIwMjYtMDEtMDFUMDA6MDA6MDBaIiwiaWQiOiIyMjIyMjIyMi0yMjIyLTQyMjItODIyMi0yMjIyMjIyMjIyMjIifQ"
	repo, _ := syncRepo(t, syncMembership(true), syncCounter(1), syncQuery{contains: "(m.created_at,m.id) < ($2,$3)", args: []driver.Value{syncShop, timestamp, syncID, int64(51)}, rows: [][]driver.Value{syncRow(syncOtherID, 1)}})
	page, err := repo.MessageHistory(context.Background(), syncUser, syncShop, cursor, 50)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Rows) != 1 || page.HasMore {
		t.Fatalf("%+v", page)
	}
}
func TestMessageSyncReaderInitialSnapshot(t *testing.T) {
	repo, c := syncRepo(t, syncMembership(true), syncCounter(20), syncQuery{contains: "ORDER BY m.created_at DESC,m.id DESC", args: []driver.Value{syncShop, int64(2)}, rows: [][]driver.Value{syncRow(syncID, 20), syncRow(syncOtherID, 19)}})
	page, err := repo.InitialMessages(context.Background(), syncUser, syncShop, 1)
	if err != nil {
		t.Fatal(err)
	}
	if page.Watermark != "20" || !page.HasOlder || page.OlderCursor == nil || len(page.Rows) != 1 || !c.committed {
		t.Fatalf("%+v", page)
	}
}
func TestMessageSyncReaderFailureDoesNotAdvance(t *testing.T) {
	repo, c := syncRepo(t, syncMembership(true), syncCounter(9), syncQuery{contains: "m.insertion_number", err: errors.New("injected query failure")})
	page, err := repo.CatchUpMessages(context.Background(), syncUser, syncShop, "2", nil, 100)
	if err == nil || page != nil || c.committed {
		t.Fatalf("%v %v", page, err)
	}
}

// Scripted driver checks cannot prove PostgreSQL commit ordering or SQL execution.
// These pages check the reader's bound/cursor orchestration over supplied results.
func TestMessageSyncReaderBurstOver100KeepsHeldBound(t *testing.T) {
	first := make([][]driver.Value, 101)
	for i := range first {
		first[i] = syncRow(fmt.Sprintf("%08d-2222-4222-8222-222222222222", i+1), int64(i+1))
	}
	repo, _ := syncRepo(t,
		syncMembership(true), syncCounter(105), syncQuery{contains: "ORDER BY m.insertion_number ASC LIMIT $4", args: []driver.Value{syncShop, int64(0), int64(105), int64(101)}, rows: first},
		syncMembership(true), syncCounter(110), syncQuery{contains: "m.insertion_number <= $3", args: []driver.Value{syncShop, int64(100), int64(105), int64(101)}, rows: [][]driver.Value{syncRow(syncOtherID, 105)}},
	)
	page, err := repo.CatchUpMessages(context.Background(), syncUser, syncShop, "0", nil, 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Rows) != 100 || !page.HasMore || page.NextAfter != "100" || page.Through != "105" {
		t.Fatalf("%+v", page)
	}
	page, err = repo.CatchUpMessages(context.Background(), syncUser, syncShop, page.NextAfter, &page.Through, 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Rows) != 1 || page.HasMore || page.NextAfter != "105" || page.Through != "105" {
		t.Fatalf("%+v", page)
	}
}
func TestMessageSyncReaderRemovalBetweenReconcileChunks(t *testing.T) {
	repo, c := syncRepo(t, syncMembership(true), syncCounter(1), syncQuery{contains: "m.shop_id = $1 AND m.id IN ($2)", rows: [][]driver.Value{syncRow(syncID, 1)}}, syncMembership(false))
	first, err := repo.ReconcileMessages(context.Background(), syncUser, syncShop, []string{syncID})
	if err != nil || len(first.Rows) != 1 {
		t.Fatalf("%+v %v", first, err)
	}
	c.committed = false
	second, err := repo.ReconcileMessages(context.Background(), syncUser, syncShop, []string{syncOtherID})
	if err == nil || second != nil || c.committed {
		t.Fatalf("%+v %v", second, err)
	}
}
func syncHasMessages(exists bool) syncQuery {
	return syncQuery{contains: "SELECT EXISTS (SELECT 1 FROM shop_messages WHERE shop_id = $1)", args: []driver.Value{syncShop}, rows: [][]driver.Value{{exists}}}
}
func requireSyncUnavailable(t *testing.T, err error) {
	t.Helper()
	var failure *shared.Failure
	if !errors.As(err, &failure) || failure.Status != 503 || failure.Code != "unsupported_contract" {
		t.Fatalf("expected unsupported_contract 503, got %v", err)
	}
}
func TestMessageSyncReaderMissingCounterFailsInitial(t *testing.T) {
	repo, c := syncRepo(t, syncMembership(true), syncQuery{contains: "SELECT last_number", rows: [][]driver.Value{{nil, int64(1), int64(1), false}}})
	page, err := repo.InitialMessages(context.Background(), syncUser, syncShop, 50)
	if err == nil || page != nil || c.committed {
		t.Fatalf("%+v %v", page, err)
	}
	requireSyncUnavailable(t, err)
}

// A Shop created after migration 018 has no counter until its first message.
func TestMessageSyncReaderMissingCounterWithoutMessagesStartsAtZero(t *testing.T) {
	repo, c := syncRepo(t, syncMembership(true), syncQuery{contains: "SELECT last_number", rows: [][]driver.Value{{nil, int64(0), int64(0), false}}}, syncQuery{contains: "ORDER BY m.created_at DESC,m.id DESC", args: []driver.Value{syncShop, int64(51)}, rows: [][]driver.Value{}})
	page, err := repo.InitialMessages(context.Background(), syncUser, syncShop, 50)
	if err != nil {
		t.Fatal(err)
	}
	if page.Watermark != "0" || page.HasOlder || page.OlderCursor != nil || page.Rows == nil || len(page.Rows) != 0 || !c.committed {
		t.Fatalf("%+v", page)
	}
}
func TestMessageSyncReaderNullInsertionNumberFails(t *testing.T) {
	row := syncRow(syncID, 1)
	row[9] = nil
	repo, c := syncRepo(t, syncMembership(true), syncCounter(1), syncQuery{contains: "m.insertion_number", rows: [][]driver.Value{row}})
	page, err := repo.InitialMessages(context.Background(), syncUser, syncShop, 50)
	if err == nil || page != nil || c.committed {
		t.Fatalf("%+v %v", page, err)
	}
	requireSyncUnavailable(t, err)
}
