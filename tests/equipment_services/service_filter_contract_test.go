package equipment_services_test

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	. "github.com/go-jet/jet/v2/postgres"
	"github.com/lib/pq"
	"miltechserver/.gen/miltech_ng/public/model"
	. "miltechserver/.gen/miltech_ng/public/table"
	"miltechserver/api/equipment_services/calendar"
	"miltechserver/api/equipment_services/queries"
	serviceshared "miltechserver/api/equipment_services/shared"
	"miltechserver/api/equipment_services/status"
	"miltechserver/api/request"
	"miltechserver/bootstrap"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	shopshared "miltechserver/api/shops/shared"
)

func serviceFilterIDs(t *testing.T, router *gin.Engine, path string, values url.Values) ([]string, int64) {
	t.Helper()
	r := doJSONRequest(t, router, http.MethodGet, path+"?"+values.Encode(), nil, "filter-owner")
	require.Equal(t, 200, r.Code, r.Body.String())
	payload := decodeMap(t, decodeStandardResponse(t, r.Body).Data)
	ids := []string{}
	for _, value := range payload["services"].([]interface{}) {
		ids = append(ids, value.(map[string]interface{})["id"].(string))
	}
	return ids, int64(payload["total_count"].(float64))
}

func TestServiceFiltersConjunctive(t *testing.T) {
	clearEquipmentServicesTables(t, testDB)
	ensureUser(t, testDB, "filter-owner")
	router := newTestRouter(t)
	shop := createShop(t, router, "filter-owner", "filters")
	equipment := createVehicle(t, router, "filter-owner", shop)
	other := createVehicle(t, router, "filter-owner", shop)
	now := time.Now().UTC()
	past, soon, future := now.Add(-48*time.Hour), now.Add(48*time.Hour), now.Add(9*24*time.Hour)
	overdue := createEquipmentService(t, router, "filter-owner", shop, equipment, "", "overdue", &past, false)
	due := createEquipmentService(t, router, "filter-owner", shop, equipment, "", "soon", &soon, false)
	scheduled := createEquipmentService(t, router, "filter-owner", shop, equipment, "", "future", &future, false)
	completed := createEquipmentService(t, router, "filter-owner", shop, equipment, "", "done", &past, true)
	undated := createEquipmentService(t, router, "filter-owner", shop, equipment, "", "undated", nil, false)
	createEquipmentService(t, router, "filter-owner", shop, other, "", "other", &soon, false)
	_, err := testDB.Exec(`UPDATE equipment_services SET service_type='repair' WHERE id=$1`, due)
	require.NoError(t, err)
	for _, suffix := range []string{"/equipment-services", "/equipment/" + equipment + "/services"} {
		path := "/api/v1/auth/shops/" + shop + suffix
		t.Run(suffix, func(t *testing.T) {
			for _, tc := range []struct {
				name     string
				values   url.Values
				expected []string
			}{
				{"overdue", url.Values{"status": {"overdue"}}, []string{overdue}},
				{"due_soon", url.Values{"status": {"due_soon"}}, []string{due}},
				{"scheduled", url.Values{"status": {"scheduled"}}, []string{due, scheduled}},
				{"completed", url.Values{"status": {"completed"}}, []string{completed}},
				{"blank", url.Values{"status": {""}}, []string{overdue, due, scheduled, completed, undated}},
				{"contradiction", url.Values{"status": {"completed"}, "is_completed": {"false"}}, []string{}},
				{"type_conjunction", url.Values{"status": {"scheduled"}, "service_type": {"repair"}, "is_completed": {"false"}}, []string{due}},
				{"type_contradiction", url.Values{"status": {"due_soon"}, "service_type": {"inspection"}}, []string{}},
				{"dates", url.Values{"start_date": {soon.Add(-time.Hour).Format(time.RFC3339)}, "end_date": {soon.Add(time.Hour).Format(time.RFC3339)}}, []string{due}},
			} {
				t.Run(tc.name, func(t *testing.T) {
					tc.values.Set("equipment_id", equipment)
					ids, filteredTotal := serviceFilterIDs(t, router, path, tc.values)
					require.ElementsMatch(t, tc.expected, ids)
					require.EqualValues(t, len(tc.expected), filteredTotal)
				})
			}
			dueSoonIDs, _ := serviceFilterIDs(t, router, path, url.Values{"status": {"due_soon"}, "equipment_id": {equipment}})
			scheduledIDs, _ := serviceFilterIDs(t, router, path, url.Values{"status": {"scheduled"}, "equipment_id": {equipment}})
			require.ElementsMatch(t, []string{due}, dueSoonIDs)
			require.Subset(t, scheduledIDs, dueSoonIDs)
			for _, query := range []string{"status=unknown", "start_date=not-a-timestamp", "end_date=2026-10-04"} {
				req := httptest.NewRequest(http.MethodGet, path+"?"+query, nil)
				req.Header.Set("X-User-ID", "filter-owner")
				req.Header.Set(shopshared.ContractHeader, "2")
				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)
				var contract2MalformedDate struct {
					Code string `json:"code"`
				}
				require.NoError(t, json.Unmarshal(w.Body.Bytes(), &contract2MalformedDate))
				require.Equal(t, "invalid", contract2MalformedDate.Code, w.Body.String())
				require.Equal(t, 400, w.Code)
			}
		})
	}
}

func TestServicePageTies(t *testing.T) {
	clearEquipmentServicesTables(t, testDB)
	ensureUser(t, testDB, "filter-owner")
	router := newTestRouter(t)
	shop := createShop(t, router, "filter-owner", "ties")
	equipment := createVehicle(t, router, "filter-owner", shop)
	date := time.Now().UTC().Add(24 * time.Hour)
	ids := []string{}
	for i := 0; i < 5; i++ {
		ids = append(ids, createEquipmentService(t, router, "filter-owner", shop, equipment, "", "tied", &date, false))
	}
	sort.Strings(ids)
	undated := createEquipmentService(t, router, "filter-owner", shop, equipment, "", "null", nil, false)
	allIDs := append(append([]string(nil), ids...), undated)
	sort.Strings(allIDs)
	_, err := testDB.Exec(`UPDATE equipment_services SET created_at=$1 WHERE shop_id=$2`, date, shop)
	require.NoError(t, err)
	for _, tc := range []struct {
		suffix   string
		expected []string
	}{{"/equipment-services", allIDs}, {"/equipment/" + equipment + "/services", append([]string{undated}, ids...)}} {
		path := "/api/v1/auth/shops/" + shop + tc.suffix
		for i, id := range tc.expected {
			got, total := serviceFilterIDs(t, router, path, url.Values{"limit": {"1"}, "offset": {strconv.Itoa(i)}})
			require.Equal(t, []string{id}, got)
			require.EqualValues(t, 6, total)
		}
	}
	start, end := date.Add(-time.Hour).Format(time.RFC3339), date.Add(time.Hour).Format(time.RFC3339)
	for _, tc := range []struct{ suffix, key string }{{"/equipment-services/calendar?start_date=" + url.QueryEscape(start) + "&end_date=" + url.QueryEscape(end), "services"}, {"/equipment-services/due-soon", "due_soon_services"}} {
		w := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/"+shop+tc.suffix, nil, "filter-owner")
		require.Equal(t, 200, w.Code)
		payload := decodeMap(t, decodeStandardResponse(t, w.Body).Data)
		got := []string{}
		for _, record := range payload[tc.key].([]interface{}) {
			got = append(got, record.(map[string]interface{})["id"].(string))
		}
		require.Equal(t, ids, got)
	}
}

// The driver barrier runs only after the previous query has been consumed. Its
// writer uses a separate physical connection and commits before this query runs.
// No production hooks or timing assumptions are needed to reproduce a torn read.
type servicePageSnapshotProbe struct {
	mutex              sync.Mutex
	queries            []string
	arguments          [][]driver.NamedValue
	options            []driver.TxOptions
	before             func(context.Context, string) error
	outsideTransaction int
}
type servicePageSnapshotConnector struct {
	base  driver.Connector
	probe *servicePageSnapshotProbe
}

func (c *servicePageSnapshotConnector) Driver() driver.Driver { return c.base.Driver() }
func (c *servicePageSnapshotConnector) Connect(ctx context.Context) (driver.Conn, error) {
	conn, err := c.base.Connect(ctx)
	if err != nil {
		return nil, err
	}
	return &servicePageSnapshotConnection{Conn: conn, probe: c.probe}, nil
}

type servicePageSnapshotConnection struct {
	driver.Conn
	probe  *servicePageSnapshotProbe
	active bool
}

func (c *servicePageSnapshotConnection) BeginTx(ctx context.Context, options driver.TxOptions) (driver.Tx, error) {
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
	return &servicePageSnapshotTransaction{Tx: tx, connection: c}, nil
}

type servicePageSnapshotTransaction struct {
	driver.Tx
	connection *servicePageSnapshotConnection
}

func (tx *servicePageSnapshotTransaction) finish() {
	tx.connection.probe.mutex.Lock()
	tx.connection.active = false
	tx.connection.probe.mutex.Unlock()
}
func (tx *servicePageSnapshotTransaction) Commit() error { defer tx.finish(); return tx.Tx.Commit() }
func (tx *servicePageSnapshotTransaction) Rollback() error {
	defer tx.finish()
	return tx.Tx.Rollback()
}

func (c *servicePageSnapshotConnection) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	c.probe.mutex.Lock()
	c.probe.queries = append(c.probe.queries, query)
	c.probe.arguments = append(c.probe.arguments, append([]driver.NamedValue(nil), args...))
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
func servicePageSnapshotDatabase(t *testing.T, probe *servicePageSnapshotProbe) *sql.DB {
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
	db := sql.OpenDB(&servicePageSnapshotConnector{base: connector, probe: probe})
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	return db
}
func requireServicePageSnapshot(t *testing.T, probe *servicePageSnapshotProbe) {
	t.Helper()
	require.Zero(t, probe.outsideTransaction, "every component must use the same transaction")
	require.Len(t, probe.options, 1, "page must own exactly one transaction")
	require.Equal(t, driver.IsolationLevel(sql.LevelRepeatableRead), probe.options[0].Isolation)
	require.True(t, probe.options[0].ReadOnly)
}

func TestServicePageSnapshot(t *testing.T) {
	for _, route := range []string{"shop", "equipment"} {
		t.Run(route, func(t *testing.T) {
			clearEquipmentServicesTables(t, testDB)
			ensureUser(t, testDB, "filter-owner")
			router := newTestRouter(t)
			shop := createShop(t, router, "filter-owner", "snapshot")
			equipment := createVehicle(t, router, "filter-owner", shop)
			future := time.Now().Add(48 * time.Hour)
			serviceID := createEquipmentService(t, router, "filter-owner", shop, equipment, "", "before", &future, false)
			changed := false
			probe := &servicePageSnapshotProbe{before: func(ctx context.Context, query string) error {
				if !changed && strings.Contains(query, "equipment_services.id") && !strings.Contains(query, "count(") {
					changed = true
					_, err := testDB.ExecContext(ctx, `DELETE FROM equipment_services WHERE id=$1`, serviceID)
					return err
				}
				return nil
			}}
			status := "due_soon"
			req := request.GetEquipmentServicesRequest{Limit: 1000, Status: &status}
			repo := queries.NewRepository(servicePageSnapshotDatabase(t, probe))
			user := &bootstrap.User{UserID: "filter-owner"}
			var services []model.EquipmentServices
			var total int64
			var err error
			if route == "shop" {
				services, total, err = repo.GetByShop(context.Background(), user, shop, req)
			} else {
				services, total, err = repo.GetByEquipment(context.Background(), user, equipment, req)
			}
			require.NoError(t, err)
			require.True(t, changed)
			require.EqualValues(t, 1, total)
			require.Len(t, services, 1)
			require.Equal(t, serviceID, services[0].ID)
			requireServicePageSnapshot(t, probe)
			require.Len(t, probe.arguments, 2)
			require.Len(t, probe.arguments[1], len(probe.arguments[0])+2)
			require.Equal(t, probe.arguments[0], probe.arguments[1][:len(probe.arguments[0])], "count and rows must use identical scope, filters and evaluation time")
			// A later request starts a fresh snapshot and sees the committed deletion.
			services, total, err = queries.NewRepository(testDB).GetByShop(context.Background(), user, shop, req)
			require.NoError(t, err)
			require.Zero(t, total)
			require.Empty(t, services)
		})
	}
}

func TestServiceReadContextCancellation(t *testing.T) {
	clearEquipmentServicesTables(t, testDB)
	ensureUser(t, testDB, "filter-owner")
	router := newTestRouter(t)
	shop := createShop(t, router, "filter-owner", "cancel")
	equipment := createVehicle(t, router, "filter-owner", shop)
	now := time.Now().UTC()
	future := now.Add(48 * time.Hour)
	createEquipmentService(t, router, "filter-owner", shop, equipment, "", "soon", &future, false)
	paths := []string{"/equipment-services", "/equipment/" + equipment + "/services", "/equipment-services/calendar?start_date=" + url.QueryEscape(now.Format(time.RFC3339)) + "&end_date=" + url.QueryEscape(future.Format(time.RFC3339)), "/equipment-services/overdue", "/equipment-services/due-soon"}
	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			outer, stop := context.WithTimeout(context.Background(), 6*time.Second)
			defer stop()
			blocker, err := testDB.BeginTx(outer, nil)
			require.NoError(t, err)
			defer blocker.Rollback()
			_, err = blocker.ExecContext(outer, `LOCK TABLE equipment_services IN ACCESS EXCLUSIVE MODE`)
			require.NoError(t, err)
			ctx, cancel := context.WithCancel(outer)
			defer cancel()
			req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/shops/"+shop+path, nil).WithContext(ctx)
			req.Header.Set("X-User-ID", "filter-owner")
			w := httptest.NewRecorder()
			done := make(chan struct{})
			go func() { defer close(done); router.ServeHTTP(w, req) }()
			waitEquipmentAuthorityBlock(t, blocker, outer)
			cancel()
			returned := false
			select {
			case <-done:
				returned = true
			case <-time.After(200 * time.Millisecond):
			}
			require.NoError(t, blocker.Rollback())
			select {
			case <-done:
			case <-outer.Done():
				t.Fatal("request did not drain")
			}
			require.True(t, returned, "request must release blocked read on cancellation")
			require.GreaterOrEqual(t, w.Code, 400)
		})
	}
}

func TestServiceCalendarMalformedTimestamp(t *testing.T) {
	clearEquipmentServicesTables(t, testDB)
	ensureUser(t, testDB, "filter-owner")
	router := newTestRouter(t)
	shop := createShop(t, router, "filter-owner", "calendar invalid")
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/shops/"+shop+"/equipment-services/calendar?start_date=bad&end_date=bad", nil)
	req.Header.Set("X-User-ID", "filter-owner")
	req.Header.Set(shopshared.ContractHeader, "2")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	var result struct {
		Code string `json:"code"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &result))
	require.Equal(t, "invalid", result.Code)
	require.Equal(t, 400, w.Code)
	require.NotContains(t, w.Body.String(), "parsing time")
}

func TestServiceFiltersTimestampBoundary(t *testing.T) {
	clearEquipmentServicesTables(t, testDB)
	ensureUser(t, testDB, "filter-owner")
	router := newTestRouter(t)
	shop := createShop(t, router, "filter-owner", "boundaries")
	equipment := createVehicle(t, router, "filter-owner", shop)
	now := time.Date(2026, 10, 4, 12, 0, 0, 0, time.FixedZone("offset", -7*3600))
	before, at, after, seventh, beyond := now.Add(-time.Second), now, now.Add(time.Second), now.Add(7*24*time.Hour), now.Add(7*24*time.Hour+time.Second)
	overdue := createEquipmentService(t, router, "filter-owner", shop, equipment, "", "before", &before, false)
	exact := createEquipmentService(t, router, "filter-owner", shop, equipment, "", "exact", &at, false)
	soon := createEquipmentService(t, router, "filter-owner", shop, equipment, "", "after", &after, false)
	boundary := createEquipmentService(t, router, "filter-owner", shop, equipment, "", "seventh", &seventh, false)
	future := createEquipmentService(t, router, "filter-owner", shop, equipment, "", "beyond", &beyond, false)
	createEquipmentService(t, router, "filter-owner", shop, equipment, "", "null", nil, false)
	createEquipmentService(t, router, "filter-owner", shop, equipment, "", "completed", &before, true)
	for _, tc := range []struct {
		status   string
		expected []string
	}{{"overdue", []string{overdue}}, {"due_soon", []string{soon, boundary}}, {"scheduled", []string{soon, boundary, future}}} {
		t.Run(tc.status, func(t *testing.T) {
			predicate, err := serviceshared.ServiceFilterPredicate(serviceshared.ServiceFilters{Status: tc.status, EquipmentID: &equipment}, now)
			require.NoError(t, err)
			var records []model.EquipmentServices
			err = SELECT(EquipmentServices.AllColumns).FROM(EquipmentServices).WHERE(predicate).QueryContext(context.Background(), testDB, &records)
			require.NoError(t, err)
			ids := []string{}
			for _, record := range records {
				ids = append(ids, record.ID)
			}
			require.ElementsMatch(t, tc.expected, ids)
		})
	}
	predicate, err := serviceshared.ServiceFilterPredicate(serviceshared.ServiceFilters{EquipmentID: &equipment, From: &at, To: &seventh}, now)
	require.NoError(t, err)
	var records []model.EquipmentServices
	err = SELECT(EquipmentServices.AllColumns).FROM(EquipmentServices).WHERE(predicate).QueryContext(context.Background(), testDB, &records)
	require.NoError(t, err)
	ids := []string{}
	for _, record := range records {
		ids = append(ids, record.ID)
	}
	require.ElementsMatch(t, []string{exact, soon, boundary}, ids)
}

func TestServiceRepositoryContextCancellation(t *testing.T) {
	clearEquipmentServicesTables(t, testDB)
	ensureUser(t, testDB, "filter-owner")
	router := newTestRouter(t)
	shop := createShop(t, router, "filter-owner", "pool")
	equipment := createVehicle(t, router, "filter-owner", shop)
	user := &bootstrap.User{UserID: "filter-owner"}
	now := time.Now()
	calls := []struct {
		name string
		call func(context.Context) error
	}{
		{"shop", func(ctx context.Context) error {
			_, _, err := queries.NewRepository(testDB).GetByShop(ctx, user, shop, request.GetEquipmentServicesRequest{Limit: 1000})
			return err
		}},
		{"equipment", func(ctx context.Context) error {
			_, _, err := queries.NewRepository(testDB).GetByEquipment(ctx, user, equipment, request.GetEquipmentServicesRequest{Limit: 1000})
			return err
		}},
		{"calendar", func(ctx context.Context) error {
			_, err := calendar.NewRepository(testDB).GetInDateRange(ctx, user, shop, now, now.Add(24*time.Hour), nil)
			return err
		}},
		{"overdue", func(ctx context.Context) error {
			_, err := status.NewRepository(testDB).GetOverdue(ctx, user, shop, nil, 50)
			return err
		}},
		{"due_soon", func(ctx context.Context) error {
			_, err := status.NewRepository(testDB).GetDueSoon(ctx, user, shop, 7, nil, 50)
			return err
		}},
	}

	for _, tc := range calls {
		for _, boundary := range []string{"pool", "query"} {
			t.Run(tc.name+"/"+boundary, func(t *testing.T) {
				if boundary == "pool" {
					testDB.SetMaxIdleConns(0)
					defer testDB.SetMaxIdleConns(2)
					testDB.SetMaxOpenConns(2)
					defer testDB.SetMaxOpenConns(0)
				}
				outer, stop := context.WithTimeout(context.Background(), 5*time.Second)
				defer stop()
				blocker, err := testDB.BeginTx(outer, nil)
				require.NoError(t, err)
				defer blocker.Rollback()
				if boundary == "query" {
					_, err = blocker.ExecContext(outer, `LOCK TABLE equipment_services IN ACCESS EXCLUSIVE MODE`)
					require.NoError(t, err)
				}
				waits := testDB.Stats().WaitCount
				ctx, cancel := context.WithCancel(outer)
				defer cancel()
				done := make(chan error, 1)
				go func() { done <- tc.call(ctx) }()
				if boundary == "pool" {
					require.Eventually(t, func() bool { return testDB.Stats().WaitCount > waits }, time.Second, time.Millisecond)
				} else {
					waitEquipmentAuthorityBlock(t, blocker, outer)
				}
				cancel()
				select {
				case err := <-done:
					require.Error(t, err)
					if boundary == "pool" {
						require.ErrorIs(t, err, context.Canceled)
					}
				case <-time.After(time.Second):
					t.Fatal("repository ignored cancellation")
				}
				require.NoError(t, blocker.Rollback())
			})
		}
	}
}

func TestServiceDedicatedDaysAhead(t *testing.T) {
	clearEquipmentServicesTables(t, testDB)
	ensureUser(t, testDB, "filter-owner")
	router := newTestRouter(t)
	shop := createShop(t, router, "filter-owner", "days")
	equipment := createVehicle(t, router, "filter-owner", shop)
	soonDate, futureDate := time.Now().Add(48*time.Hour), time.Now().Add(20*24*time.Hour)
	soon := createEquipmentService(t, router, "filter-owner", shop, equipment, "", "soon", &soonDate, false)
	future := createEquipmentService(t, router, "filter-owner", shop, equipment, "", "future", &futureDate, false)
	path := "/api/v1/auth/shops/" + shop + "/equipment-services/due-soon"
	for _, tc := range []struct {
		days     string
		expected []string
	}{{"1", []string{}}, {"7", []string{soon}}, {"30", []string{soon, future}}} {
		w := doJSONRequest(t, router, http.MethodGet, path+"?days_ahead="+tc.days, nil, "filter-owner")
		require.Equal(t, 200, w.Code)
		payload := decodeMap(t, decodeStandardResponse(t, w.Body).Data)
		ids := []string{}
		for _, value := range payload["due_soon_services"].([]interface{}) {
			ids = append(ids, value.(map[string]interface{})["id"].(string))
		}
		require.ElementsMatch(t, tc.expected, ids)
	}
	for _, days := range []string{"0", "31", "-1"} {
		req := httptest.NewRequest(http.MethodGet, path+"?days_ahead="+days, nil)
		req.Header.Set("X-User-ID", "filter-owner")
		req.Header.Set(shopshared.ContractHeader, "2")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		require.Equal(t, 400, w.Code, w.Body.String())
	}
}

func TestServiceExplicitEmptyEquipmentFilter(t *testing.T) {
	clearEquipmentServicesTables(t, testDB)
	ensureUser(t, testDB, "filter-owner")
	router := newTestRouter(t)
	shop := createShop(t, router, "filter-owner", "empty")
	equipment := createVehicle(t, router, "filter-owner", shop)
	now := time.Now().UTC()
	past, future := now.Add(-48*time.Hour), now.Add(48*time.Hour)
	createEquipmentService(t, router, "filter-owner", shop, equipment, "", "past", &past, false)
	createEquipmentService(t, router, "filter-owner", shop, equipment, "", "future", &future, false)
	bounds := "&start_date=" + url.QueryEscape(past.Add(-time.Hour).Format(time.RFC3339)) + "&end_date=" + url.QueryEscape(future.Add(time.Hour).Format(time.RFC3339))
	for _, tc := range []struct{ path, key string }{{"/equipment-services?equipment_id=", "services"}, {"/equipment/" + equipment + "/services?equipment_id=", "services"}, {"/equipment-services/calendar?equipment_id=" + bounds, "services"}, {"/equipment-services/overdue?equipment_id=", "overdue_services"}, {"/equipment-services/due-soon?equipment_id=", "due_soon_services"}} {
		t.Run(tc.key+tc.path, func(t *testing.T) {
			w := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/"+shop+tc.path, nil, "filter-owner")
			require.Equal(t, 200, w.Code)
			payload := decodeMap(t, decodeStandardResponse(t, w.Body).Data)
			require.Empty(t, payload[tc.key])
			require.EqualValues(t, 0, payload["total_count"])
			omitted := strings.Replace(tc.path, "equipment_id=", "", 1)
			w = doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/"+shop+omitted, nil, "filter-owner")
			require.Equal(t, 200, w.Code)
			payload = decodeMap(t, decodeStandardResponse(t, w.Body).Data)
			require.NotEmpty(t, payload[tc.key], "omitting the optional filter retains scoped services")
		})
	}
}

func TestServiceStatusElapsedDays(t *testing.T) {
	clearEquipmentServicesTables(t, testDB)
	ensureUser(t, testDB, "filter-owner")
	router := newTestRouter(t)
	shop := createShop(t, router, "filter-owner", "elapsed")
	equipment := createVehicle(t, router, "filter-owner", shop)
	date := time.Date(1500, 1, 1, 12, 0, 0, 0, time.UTC)
	service := createEquipmentService(t, router, "filter-owner", shop, equipment, "", "historical", &date, false)
	var expected int
	require.NoError(t, testDB.QueryRow(`SELECT EXTRACT(DAY FROM NOW()-service_date)::integer FROM equipment_services WHERE id=$1`, service).Scan(&expected))
	w := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/"+shop+"/equipment-services/overdue", nil, "filter-owner")
	require.Equal(t, 200, w.Code)
	payload := decodeMap(t, decodeStandardResponse(t, w.Body).Data)
	records := payload["overdue_services"].([]interface{})
	require.Len(t, records, 1)
	require.EqualValues(t, expected, records[0].(map[string]interface{})["days_overdue"])
}
