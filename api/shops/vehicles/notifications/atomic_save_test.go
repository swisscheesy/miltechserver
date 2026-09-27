package notifications

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
	"io"
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/request"
	"miltechserver/api/shops/shared"
	notificationitems "miltechserver/api/shops/vehicles/notifications/items"
	"miltechserver/bootstrap"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func atomicRequest() request.NotificationSaveRequest {
	return request.NotificationSaveRequest{OperationID: "11111111-1111-4111-8111-111111111111", ShopID: "22222222-2222-4222-8222-222222222222", VehicleID: "33333333-3333-4333-8333-333333333333", Details: request.NotificationSaveDetails{Title: "Repair", Type: "M1"}, Attachment: request.NotificationSaveAttachment{Intent: "keep"}, Items: []request.NotificationSaveItem{}}
}
func TestFingerprintSemanticSubmission(t *testing.T) {
	a := atomicRequest()
	a.Items = []request.NotificationSaveItem{{ID: "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", Niin: "123", Nomenclature: "part", Quantity: 1}, {ID: "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb", Niin: "456", Nomenclature: "other", Quantity: 2}}
	first, err := FingerprintNotificationSave(a)
	if err != nil {
		t.Fatal(err)
	}
	a.OperationID = "44444444-4444-4444-8444-444444444444"
	a.Items[0], a.Items[1] = a.Items[1], a.Items[0]
	second, err := FingerprintNotificationSave(a)
	if err != nil || first != second {
		t.Fatal("operation identity or item ordering affected fingerprint")
	}
	a.Items[0].Quantity++
	third, _ := FingerprintNotificationSave(a)
	if first == third {
		t.Fatal("payload difference lost")
	}
}
func TestAtomicValidation(t *testing.T) {
	for _, mutate := range []func(*request.NotificationSaveRequest){
		func(r *request.NotificationSaveRequest) { r.OperationID = "bad" },
		func(r *request.NotificationSaveRequest) { r.Attachment.Intent = "unknown" },
		func(r *request.NotificationSaveRequest) { r.Attachment.Intent = "attach" },
		func(r *request.NotificationSaveRequest) { id := r.ShopID; r.Attachment.ListID = &id },
		func(r *request.NotificationSaveRequest) { r.Details.Type = "BAD" },
		func(r *request.NotificationSaveRequest) {
			r.Items = []request.NotificationSaveItem{{ID: r.ShopID, Niin: "123", Nomenclature: "part", Quantity: 1}, {ID: r.ShopID, Niin: "456", Nomenclature: "other", Quantity: 1}}
		},
	} {
		r := atomicRequest()
		mutate(&r)
		if ValidateNotificationSave(r) == nil {
			t.Fatalf("accepted invalid request: %+v", r)
		}
	}
	if err := ValidateNotificationSave(atomicRequest()); err != nil {
		t.Fatal(err)
	}
}

// A driver boundary exercises real repository transaction control without a DB.
// It cannot establish PostgreSQL lock or isolation behavior.
func TestAtomicRepositoryRollbackAndReceipt(t *testing.T) {
	for _, phase := range []string{"claim", "details", "items", "audit", "receipt"} {
		t.Run(phase, func(t *testing.T) {
			state := &atomicDriverState{fail: phase}
			conn := sql.OpenDB(atomicConnector{state})
			defer conn.Close()
			r := atomicRequest()
			r.Items = []request.NotificationSaveItem{{ID: r.ShopID, Niin: "123", Nomenclature: "part", Quantity: 1}}
			receipt, err := NewRepository(conn).SaveAtomic(context.Background(), "user", r)
			if err == nil || receipt.NotificationID != "" || state.commits != 0 || state.rollbacks != 1 {
				t.Fatalf("partial save: %+v err=%v state=%+v", receipt, err, state)
			}
			if strings.Contains(err.Error(), "private") {
				t.Fatalf("driver error leaked: %v", err)
			}
		})
	}
	state := &atomicDriverState{}
	conn := sql.OpenDB(atomicConnector{state})
	defer conn.Close()
	repo := NewRepository(conn)
	r := atomicRequest()
	first, err := repo.SaveAtomic(context.Background(), "user", r)
	if err != nil {
		t.Fatal(err)
	}
	writes := state.businessWrites
	second, err := repo.SaveAtomic(context.Background(), "user", r)
	if err != nil || !second.Replayed || second.NotificationID != first.NotificationID || !second.CommittedAt.Equal(first.CommittedAt) || state.businessWrites != writes {
		t.Fatalf("bad replay: %+v %v", second, err)
	}
	r.Details.Title = "different"
	_, err = repo.SaveAtomic(context.Background(), "user", r)
	if err == nil || shared.ClassifyFailure(err).Code != "operation_payload_conflict" || state.businessWrites != writes {
		t.Fatalf("bad conflict: %v", err)
	}
}

func TestAtomicAmbiguousCommitResolvesWithSameOperation(t *testing.T) {
	state := &atomicDriverState{ambiguous: true}
	conn := sql.OpenDB(atomicConnector{state})
	defer conn.Close()
	repo := NewRepository(conn)
	r := atomicRequest()
	receipt, err := repo.SaveAtomic(context.Background(), "user", r)
	if err == nil || receipt.NotificationID != "" {
		t.Fatal("uncertain commit exposed success")
	}
	writes := state.businessWrites
	receipt, err = repo.SaveAtomic(context.Background(), "user", r)
	if err != nil || !receipt.Replayed || state.businessWrites != writes {
		t.Fatalf("retry executed body: %+v %v", receipt, err)
	}
}
func TestAtomicMembershipRequiredForWriteAndReplay(t *testing.T) {
	for _, replay := range []bool{false, true} {
		t.Run(fmt.Sprint(replay), func(t *testing.T) {
			state := &atomicDriverState{}
			conn := sql.OpenDB(atomicConnector{state})
			defer conn.Close()
			repo := NewRepository(conn)
			r := atomicRequest()
			if replay {
				if _, err := repo.SaveAtomic(context.Background(), "user", r); err != nil {
					t.Fatal(err)
				}
			}
			writes := state.businessWrites
			state.denied = true
			if _, err := repo.SaveAtomic(context.Background(), "user", r); err == nil || shared.ClassifyFailure(err).Status != 403 {
				t.Fatalf("membership denial lost: %v", err)
			}
			if state.businessWrites != writes {
				t.Fatal("denied body ran")
			}
		})
	}
}
func TestAtomicCancellationAndForeignItems(t *testing.T) {
	state := &atomicDriverState{foreignItem: true}
	conn := sql.OpenDB(atomicConnector{state})
	defer conn.Close()
	repo := NewRepository(conn)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := repo.SaveAtomic(ctx, "user", atomicRequest()); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	r := atomicRequest()
	r.Items = []request.NotificationSaveItem{{ID: r.ShopID, Niin: "123", Nomenclature: "part", Quantity: 1}}
	if _, err := repo.SaveAtomic(context.Background(), "user", r); err == nil || shared.ClassifyFailure(err).Status != 400 || state.commits != 0 {
		t.Fatalf("foreign item accepted: %v", err)
	}
}

type atomicDriverState struct {
	connections           int
	normalizedCommittedAt time.Time
	missingReceipt        bool

	deletedShop       bool
	changedAttachment int
	lockedLists       []string

	auditAttempts int

	notification            *model.ShopVehicleNotifications
	existingItems           [][]driver.Value
	auditKinds, auditFields []string
	queryLog                []string
	retryCount              int
	missingNotification     bool

	fail                               string
	denied, foreignItem, ambiguous     bool
	committed, pending                 bool
	fingerprint                        []byte
	target                             string
	committedAt                        time.Time
	commits, rollbacks, businessWrites int
}
type atomicConnector struct{ state *atomicDriverState }

func (c atomicConnector) Connect(context.Context) (driver.Conn, error) {
	c.state.connections++
	return &atomicConn{state: c.state}, nil
}
func (c atomicConnector) Driver() driver.Driver { return atomicDriver{} }

type atomicDriver struct{}

func (atomicDriver) Open(string) (driver.Conn, error) { return nil, errors.New("use connector") }

type atomicConn struct{ state *atomicDriverState }

func (*atomicConn) Prepare(string) (driver.Stmt, error) { return nil, errors.New("unexpected prepare") }
func (*atomicConn) Close() error                        { return nil }
func (c *atomicConn) Begin() (driver.Tx, error)         { return c, nil }
func (c *atomicConn) BeginTx(_ context.Context, opts driver.TxOptions) (driver.Tx, error) {
	if opts.Isolation != driver.IsolationLevel(sql.LevelReadCommitted) {
		return nil, errors.New("wrong isolation")
	}
	return c, nil
}
func (c *atomicConn) Commit() error {
	c.state.commits++
	c.state.committed = c.state.committed || c.state.pending
	c.state.pending = false
	if c.state.ambiguous {
		c.state.ambiguous = false
		return errors.New("private ambiguous commit")
	}
	return nil
}
func (c *atomicConn) Rollback() error { c.state.rollbacks++; c.state.pending = false; return nil }
func (c *atomicConn) ExecContext(_ context.Context, q string, args []driver.NamedValue) (driver.Result, error) {
	s := c.state
	phase := ""
	if s.retryCount > 0 {
		s.retryCount--
		return nil, &pq.Error{Code: "40001"}
	}
	switch {
	case strings.HasPrefix(q, "INSERT INTO shop_notification_operations"):
		phase = "claim"
		if s.fail == phase {
			return nil, errors.New("private driver failure")
		}
		if s.committed {
			return driver.RowsAffected(0), nil
		}
		s.pending = true
		s.fingerprint = append([]byte{}, args[2].Value.([]byte)...)
		s.target = args[3].Value.(string)
		s.committedAt = args[4].Value.(time.Time)
	case strings.HasPrefix(q, "INSERT INTO shop_vehicle_notifications"), strings.HasPrefix(q, "UPDATE shop_vehicle_notifications"), strings.HasPrefix(q, "DELETE FROM shop_vehicle_notifications"):
		phase = "details"
	case strings.Contains(q, "shop_notification_items"):
		phase = "items"
	case strings.Contains(q, "shop_vehicle_notification_changes"):
		s.auditAttempts++
		if s.fail == "legacy-audit" && s.commits == 0 {
			return nil, errors.New("audit inside business transaction")
		}
		phase = "audit"
		if s.commits > 0 && s.fail == "legacy-audit" {
			return nil, errors.New("private legacy audit failure")
		}
		s.auditKinds = append(s.auditKinds, args[4].Value.(string))
		s.auditFields = append(s.auditFields, args[5].Value.(string))
	default:
		return nil, fmt.Errorf("unexpected exec: %s", q)
	}
	if s.fail == phase {
		return nil, errors.New("private driver failure")
	}
	if phase != "claim" {
		s.businessWrites++
	}
	return driver.RowsAffected(1), nil
}
func (c *atomicConn) QueryContext(_ context.Context, q string, args []driver.NamedValue) (driver.Rows, error) {
	s := c.state
	s.queryLog = append(s.queryLog, q)
	var values []driver.Value
	switch {
	case strings.HasPrefix(q, "UPDATE shop_notification_operations"):
		if !strings.HasSuffix(q, "RETURNING committed_at") {
			return nil, errors.New("receipt update must return its stored timestamp")
		}
		if s.fail == "receipt" {
			return nil, errors.New("private driver failure")
		}
		if s.missingReceipt {
			return &atomicRows{cols: []string{"committed_at"}}, nil
		}
		s.businessWrites++
		s.committedAt = args[0].Value.(time.Time)
		if !s.normalizedCommittedAt.IsZero() {
			s.committedAt = s.normalizedCommittedAt
		}
		values = []driver.Value{s.committedAt}

	case strings.HasPrefix(q, "INSERT INTO shop_notification_items"):
		s.businessWrites++
		values = make([]driver.Value, len(args))
		for i, arg := range args {
			values[i] = arg.Value
		}
	case strings.HasPrefix(q, "INSERT INTO shop_vehicle_notifications"):
		s.businessWrites++
		values = make([]driver.Value, len(args))
		for i, arg := range args {
			values[i] = arg.Value
		}
	case strings.HasPrefix(q, "SELECT attached_shop_list"):
		if s.notification != nil && s.notification.AttachedShopList != nil {
			values = []driver.Value{*s.notification.AttachedShopList}
		} else {
			values = []driver.Value{nil}
		}
	case strings.HasPrefix(q, "SELECT id,shop_id,vehicle_id,title"):
		n := s.notification
		if n == nil {
			return &atomicRows{cols: make([]string, 10)}, nil
		}
		var list driver.Value
		if n.AttachedShopList != nil {
			list = *n.AttachedShopList
		}
		if s.changedAttachment > 0 {
			list = "changed-list"
			s.changedAttachment--
		}
		values = []driver.Value{n.ID, n.ShopID, n.VehicleID, n.Title, n.Description, n.Type, n.Completed, list, n.SaveTime, n.LastUpdated}
	case strings.HasPrefix(q, "SELECT id FROM shop_notification_items"):
		return &atomicRows{cols: []string{"id"}}, nil
	case strings.HasPrefix(q, "SELECT shop_id FROM shop_lists"):
		s.lockedLists = append(s.lockedLists, args[0].Value.(string))
		values = []driver.Value{atomicRequest().ShopID}
	case strings.HasPrefix(q, "SELECT shop_id,vehicle_id FROM shop_vehicle_notifications"):
		if s.missingNotification {
			return &atomicRows{cols: []string{"shop_id", "vehicle_id"}}, nil
		}
		values = []driver.Value{atomicRequest().ShopID, atomicRequest().VehicleID}
	case strings.HasPrefix(q, "SELECT fingerprint"):
		values = []driver.Value{s.fingerprint, s.target, s.committedAt}
	case strings.HasPrefix(q, "SELECT admin_only_lists"):
		if s.deletedShop {
			return &atomicRows{cols: []string{"admin_only_lists"}}, nil
		}
		values = []driver.Value{false}
	case strings.HasPrefix(q, "SELECT role"):
		if s.denied {
			return &atomicRows{cols: []string{"role"}}, nil
		}
		values = []driver.Value{"member"}
	case strings.HasPrefix(q, "SELECT shop_id,admin"):
		values = []driver.Value{atomicRequest().ShopID, "A-1"}
	case strings.HasPrefix(q, "SELECT id,niin"):
		return &atomicRows{cols: []string{"id", "niin", "nomenclature", "quantity"}, queue: s.existingItems}, nil
	case strings.HasPrefix(q, "SELECT notification_id FROM shop_notification_items"):
		if !s.foreignItem {
			return &atomicRows{cols: []string{"notification_id"}}, nil
		}
		values = []driver.Value{"foreign"}
	default:
		return nil, fmt.Errorf("unexpected query: %s", q)
	}
	cols := make([]string, len(values))
	return &atomicRows{cols: cols, values: values}, nil
}

type atomicRows struct {
	queue [][]driver.Value

	cols   []string
	values []driver.Value
	read   bool
}

func (r *atomicRows) Columns() []string { return r.cols }
func (*atomicRows) Close() error        { return nil }
func (r *atomicRows) Next(dest []driver.Value) error {
	if len(r.queue) > 0 {
		copy(dest, r.queue[0])
		r.queue = r.queue[1:]
		return nil
	}
	if r.read || r.values == nil {
		return io.EOF
	}
	copy(dest, r.values)
	r.read = true
	return nil
}

func TestLegacyDeleteRequiresLockedMembership(t *testing.T) {
	state := &atomicDriverState{denied: true}
	conn := sql.OpenDB(atomicConnector{state})
	defer conn.Close()
	err := NewRepository(conn).DeleteVehicleNotification(&bootstrap.User{UserID: "user"}, "target")
	if err == nil || shared.ClassifyFailure(err).Status != 403 || state.businessWrites != 0 || state.rollbacks != 1 {
		t.Fatalf("legacy delete bypassed locked membership: %v %+v", err, state)
	}
}

func TestAtomicRealRouteRejectsInvalidBeforeDatabaseAccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		name, header string
		invalid      bool
		status       int
	}{{"legacy", "", false, 400}, {"unknown", "3", false, 400}, {"invalid", "2", true, 400}} {
		t.Run(tc.name, func(t *testing.T) {
			state := &atomicDriverState{}
			db := sql.OpenDB(atomicConnector{state})
			defer db.Close()
			engine := gin.New()
			group := engine.Group("/api/v1/auth", shared.ContractMiddleware, func(c *gin.Context) { c.Set("user", &bootstrap.User{UserID: "user"}) })
			RegisterRoutes(group, NewService(NewRepository(db), nil))
			r := atomicRequest()
			if tc.invalid {
				r.Attachment.Intent = "invalid"
			}
			body, _ := json.Marshal(r)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/shops/vehicles/notifications/save", strings.NewReader(string(body)))
			req.Header.Set(shared.ContractHeader, tc.header)
			rec := httptest.NewRecorder()
			engine.ServeHTTP(rec, req)
			if rec.Code != tc.status {
				t.Fatalf("response %d %s", rec.Code, rec.Body.String())
			}
			// A fresh sql.DB cannot start a transaction or execute SQL without
			// opening a driver connection first.
			if state.connections != 0 {
				t.Fatalf("invalid route accessed database: %+v", state)
			}
		})
	}
}

type repositoryWithoutAtomicSaver struct{ Repository }

func TestAtomicServiceWithoutSaverReturnsTypedUnavailable(t *testing.T) {
	service := NewService(repositoryWithoutAtomicSaver{}, nil)
	receipt, err := service.SaveAtomic(context.Background(), "user", atomicRequest())
	if receipt.NotificationID != "" || err == nil {
		t.Fatalf("unexpected unsupported repository result: %+v %v", receipt, err)
	}
	failure := shared.ClassifyFailure(err)
	if failure.Status != http.StatusServiceUnavailable || failure.Code != "unsupported_contract" {
		t.Fatalf("missing typed unavailable failure: %+v", failure)
	}
}

func TestAtomicLastSaveWinsAuditsInterveningItems(t *testing.T) {
	r := atomicRequest()
	id := "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
	r.NotificationID = &id
	now := time.Now().UTC()
	state := &atomicDriverState{notification: &model.ShopVehicleNotifications{ID: id, ShopID: r.ShopID, VehicleID: r.VehicleID, Title: "Earlier", Type: "M1", SaveTime: now, LastUpdated: now}, existingItems: [][]driver.Value{{"bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb", "123", "intervening part", int64(2)}}}
	db := sql.OpenDB(atomicConnector{state})
	defer db.Close()
	if _, err := NewRepository(db).SaveAtomic(context.Background(), "user", r); err != nil {
		t.Fatal(err)
	}
	found := false
	for i, kind := range state.auditKinds {
		if kind == "items_removed" {
			found = true
			if !strings.Contains(state.auditFields[i], "intervening part") {
				t.Fatal("removal audit lost item")
			}
		}
	}
	if !found {
		t.Fatal("missing removal audit")
	}
}
func TestAtomicMissingEditIsNotificationNotFound(t *testing.T) {
	state := &atomicDriverState{missingNotification: true}
	db := sql.OpenDB(atomicConnector{state})
	defer db.Close()
	r := atomicRequest()
	id := r.ShopID
	r.NotificationID = &id
	_, err := NewRepository(db).SaveAtomic(context.Background(), "user", r)
	if err == nil || shared.ClassifyFailure(err).Code != "notification_not_found" {
		t.Fatalf("wrong missing edit error: %v", err)
	}
}
func TestAtomicRetriesOnlyRolledBackFailures(t *testing.T) {
	state := &atomicDriverState{retryCount: 2}
	db := sql.OpenDB(atomicConnector{state})
	defer db.Close()
	if _, err := NewRepository(db).SaveAtomic(context.Background(), "user", atomicRequest()); err != nil || state.rollbacks != 2 || state.commits != 1 {
		t.Fatalf("retry failed: %v %+v", err, state)
	}
}
func TestLegacyItemRequiresMembershipAndKeepsAuditBestEffort(t *testing.T) {
	r := atomicRequest()
	now := time.Now().UTC()
	id := "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
	for _, denied := range []bool{true, false} {
		t.Run(fmt.Sprint(denied), func(t *testing.T) {
			state := &atomicDriverState{denied: denied, fail: "legacy-audit", notification: &model.ShopVehicleNotifications{ID: id, ShopID: r.ShopID, VehicleID: r.VehicleID, Title: "title", Type: "M1", SaveTime: now, LastUpdated: now}}
			db := sql.OpenDB(atomicConnector{state})
			defer db.Close()
			item := model.ShopNotificationItems{ID: r.ShopID, ShopID: r.ShopID, NotificationID: id, Niin: "123", Nomenclature: "part", Quantity: 1, SaveTime: now}
			created, err := notificationitems.NewRepository(db).CreateNotificationItem(&bootstrap.User{UserID: "user"}, item)
			if denied {
				if err == nil || state.businessWrites != 0 || state.commits != 0 {
					t.Fatal("denied item wrote")
				}
				return
			}
			if err != nil || created == nil || created.ID != item.ID || state.commits != 1 {
				t.Fatalf("legacy success lost: %v", err)
			}
		})
	}
}

func TestAtomicReplayAfterTargetOrShopDeletion(t *testing.T) {
	for _, shopDeleted := range []bool{false, true} {
		t.Run(fmt.Sprint(shopDeleted), func(t *testing.T) {
			state := &atomicDriverState{}
			db := sql.OpenDB(atomicConnector{state})
			defer db.Close()
			repo := NewRepository(db)
			r := atomicRequest()
			first, err := repo.SaveAtomic(context.Background(), "user", r)
			if err != nil {
				t.Fatal(err)
			}
			state.missingNotification = true
			state.deletedShop = shopDeleted
			writes := state.businessWrites
			receipt, err := repo.SaveAtomic(context.Background(), "user", r)
			if err != nil || !receipt.Replayed || receipt.NotificationID != first.NotificationID || state.businessWrites != writes {
				t.Fatalf("deleted target replay mutated data: %v", err)
			}
		})
	}
}
func TestAtomicAttachmentLocksSortedAndRestartsOnChange(t *testing.T) {
	r := atomicRequest()
	id := "cccccccc-cccc-4ccc-8ccc-cccccccccccc"
	r.NotificationID = &id
	oldList := "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb"
	newList := "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
	r.Attachment = request.NotificationSaveAttachment{Intent: "attach", ListID: &newList}
	now := time.Now()
	state := &atomicDriverState{changedAttachment: 1, notification: &model.ShopVehicleNotifications{ID: id, ShopID: r.ShopID, VehicleID: r.VehicleID, Title: "title", Type: "M1", AttachedShopList: &oldList, SaveTime: now, LastUpdated: now}}
	db := sql.OpenDB(atomicConnector{state})
	defer db.Close()
	if _, err := NewRepository(db).SaveAtomic(context.Background(), "user", r); err != nil {
		t.Fatal(err)
	}
	if state.rollbacks != 1 || state.commits != 1 || len(state.lockedLists) != 4 {
		t.Fatalf("no whole-transaction restart: %+v", state)
	}
	for i := 0; i < 4; i += 2 {
		if state.lockedLists[i] != newList || state.lockedLists[i+1] != oldList {
			t.Fatalf("list lock order: %v", state.lockedLists)
		}
	}
}
func TestLegacyDetailAndDeleteAuditsAfterCommit(t *testing.T) {
	r := atomicRequest()
	now := time.Now()
	id := "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
	for _, deleting := range []bool{false, true} {
		t.Run(fmt.Sprint(deleting), func(t *testing.T) {
			state := &atomicDriverState{fail: "legacy-audit", notification: &model.ShopVehicleNotifications{ID: id, ShopID: r.ShopID, VehicleID: r.VehicleID, Title: "before", Type: "M1", SaveTime: now, LastUpdated: now}}
			db := sql.OpenDB(atomicConnector{state})
			defer db.Close()
			repo := NewRepository(db)
			user := &bootstrap.User{UserID: "user"}
			var err error
			if deleting {
				err = repo.DeleteVehicleNotification(user, id)
			} else {
				err = repo.UpdateVehicleNotification(user, VehicleNotificationUpdate{Notification: model.ShopVehicleNotifications{ID: id, Title: "after", Type: "M1", LastUpdated: now}})
			}
			if err != nil || state.commits != 1 || state.auditAttempts != 1 {
				t.Fatalf("best-effort audit changed legacy success: %v %+v", err, state)
			}
		})
	}
}

func TestAtomicReceiptUsesStoredTimestamp(t *testing.T) {
	stored := time.Date(2026, 9, 26, 12, 34, 56, 123456000, time.UTC)
	state := &atomicDriverState{normalizedCommittedAt: stored}
	db := sql.OpenDB(atomicConnector{state})
	defer db.Close()
	repo := NewRepository(db)
	r := atomicRequest()
	first, err := repo.SaveAtomic(context.Background(), "user", r)
	if err != nil {
		t.Fatal(err)
	}
	replay, err := repo.SaveAtomic(context.Background(), "user", r)
	if err != nil {
		t.Fatal(err)
	}
	if !first.CommittedAt.Equal(stored) || !replay.CommittedAt.Equal(stored) || !first.CommittedAt.Equal(replay.CommittedAt) {
		t.Fatalf("timestamps differ: first=%s replay=%s stored=%s", first.CommittedAt, replay.CommittedAt, stored)
	}
	if first.Replayed || !replay.Replayed {
		t.Fatal("incorrect replay markers")
	}
}
func TestAtomicMissingReceiptCompletionRollsBack(t *testing.T) {
	state := &atomicDriverState{missingReceipt: true}
	db := sql.OpenDB(atomicConnector{state})
	defer db.Close()
	receipt, err := NewRepository(db).SaveAtomic(context.Background(), "user", atomicRequest())
	if err == nil || !errors.Is(err, sql.ErrNoRows) || shared.ClassifyFailure(err).Status != 500 || receipt.NotificationID != "" || state.commits != 0 || state.rollbacks != 1 {
		t.Fatalf("missing receipt committed: receipt=%+v error=%v state=%+v", receipt, err, state)
	}
}
func TestAtomicValidationRejectsDuplicateIDsIndependently(t *testing.T) {
	r := atomicRequest()
	item := request.NotificationSaveItem{ID: r.ShopID, Niin: "123", Nomenclature: "part", Quantity: 1}
	r.Items = []request.NotificationSaveItem{item}
	if err := ValidateNotificationSave(r); err != nil {
		t.Fatalf("fixture not valid: %v", err)
	}
	r.Items = append(r.Items, item)
	if ValidateNotificationSave(r) == nil {
		t.Fatal("accepted duplicate item ID")
	}
}
