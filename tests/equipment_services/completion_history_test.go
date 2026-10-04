package equipment_services_test

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestCompletionDateStableOnRetryAndMetadataEdit(t *testing.T) {
	clearEquipmentServicesTables(t, testDB)
	ensureUser(t, testDB, "history-author")
	router := newTestRouter(t)
	shop := createShop(t, router, "history-author", "Completion history")
	vehicle := createVehicle(t, router, "history-author", shop)
	list := createList(t, router, "history-author", shop)
	service := createEquipmentService(t, router, "history-author", shop, vehicle, list, "original", nil, false)
	path := "/api/v1/auth/shops/" + shop + "/equipment-services/" + service
	metadata := func() map[string]any {
		return map[string]any{"service_id": service, "description": "edited", "service_type": "inspection", "list_id": list, "is_completed": true}
	}
	write := func(t *testing.T, method, suffix string, body map[string]any) map[string]any {
		resp := doJSONRequest(t, router, method, path+suffix, body, "history-author")
		require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
		return decodeMap(t, decodeStandardResponse(t, resp.Body).Data)
	}
	assertDate := func(t *testing.T, data map[string]any, expected *time.Time) {
		t.Helper()
		if expected == nil {
			require.Contains(t, data, "completion_date")
			require.Nil(t, data["completion_date"])
		} else {
			parsed, err := time.Parse(time.RFC3339Nano, data["completion_date"].(string))
			require.NoError(t, err)
			require.True(t, expected.Equal(parsed), "historical date overwritten: want %s, got %s", expected, parsed)
		}
		var persisted sql.NullTime
		require.NoError(t, testDB.QueryRow(`SELECT completion_date FROM equipment_services WHERE id=$1`, service).Scan(&persisted))
		require.Equal(t, expected != nil, persisted.Valid)
		if expected != nil {
			require.True(t, expected.Equal(persisted.Time))
		}
	}

	t.Run("first completion and repeat omitted or null", func(t *testing.T) {
		before := time.Now()
		first := write(t, http.MethodPost, "/complete", map[string]any{})
		firstDate, err := time.Parse(time.RFC3339Nano, first["completion_date"].(string))
		require.NoError(t, err)
		require.WithinRange(t, firstDate, before.Add(-time.Microsecond), time.Now())
		for _, body := range []map[string]any{{}, {"completion_date": nil}} {
			assertDate(t, write(t, http.MethodPost, "/complete", body), &firstDate)
		}
	})

	historic := time.Date(2018, 2, 3, 4, 5, 6, 123456000, time.UTC)
	_, err := testDB.Exec(`UPDATE equipment_services SET is_completed=true,completion_date=$2 WHERE id=$1`, service, historic)
	require.NoError(t, err)
	t.Run("historical metadata omission and null", func(t *testing.T) {
		for _, nullDate := range []bool{false, true} {
			body := metadata()
			if nullDate {
				body["completion_date"] = nil
			}
			assertDate(t, write(t, http.MethodPut, "", body), &historic)
		}
	})
	t.Run("explicit completion and metadata dates", func(t *testing.T) {
		explicit := time.Date(2020, 7, 8, 9, 10, 11, 0, time.FixedZone("offset", -7*60*60))
		assertDate(t, write(t, http.MethodPost, "/complete", map[string]any{"completion_date": explicit}), &explicit)
		explicitEdit := explicit.Add(time.Hour)
		body := metadata()
		body["completion_date"] = explicitEdit
		assertDate(t, write(t, http.MethodPut, "", body), &explicitEdit)
	})
	t.Run("completed legacy null remains null", func(t *testing.T) {
		_, err := testDB.Exec(`UPDATE equipment_services SET is_completed=true,completion_date=NULL WHERE id=$1`, service)
		require.NoError(t, err)
		for _, body := range []map[string]any{{}, {"completion_date": nil}} {
			assertDate(t, write(t, http.MethodPost, "/complete", body), nil)
		}
		assertDate(t, write(t, http.MethodPut, "", metadata()), nil)
		body := metadata()
		body["completion_date"] = nil
		assertDate(t, write(t, http.MethodPut, "", body), nil)
	})
	t.Run("reopen and omitted completion flag keep existing defaults", func(t *testing.T) {
		for _, omitFlag := range []bool{false, true} {
			_, err := testDB.Exec(`UPDATE equipment_services SET is_completed=true,completion_date=$2 WHERE id=$1`, service, historic)
			require.NoError(t, err)
			body := metadata()
			body["completion_date"] = historic
			if omitFlag {
				delete(body, "is_completed")
			} else {
				body["is_completed"] = false
			}
			data := write(t, http.MethodPut, "", body)
			require.Equal(t, false, data["is_completed"])
			assertDate(t, data, nil)
		}
		before := time.Now()
		data := write(t, http.MethodPut, "", metadata())
		date, err := time.Parse(time.RFC3339Nano, data["completion_date"].(string))
		require.NoError(t, err)
		require.WithinRange(t, date, before.Add(-time.Microsecond), time.Now())
	})
}

// Observe physical database waits so the writer's preflight must precede the
// commit; elapsed time alone cannot establish a stale request view.
func waitCompletionWriters(t *testing.T, ctx context.Context, tx *sql.Tx, count int) {
	t.Helper()
	for {
		_, err := tx.ExecContext(ctx, `SELECT pg_stat_clear_snapshot()`)
		require.NoError(t, err)
		var waiting int
		err = tx.QueryRowContext(ctx, `SELECT count(*) FROM pg_stat_activity WHERE datname=current_database() AND cardinality(pg_blocking_pids(pid))>0`).Scan(&waiting)
		require.NoError(t, err)
		if waiting >= count {
			return
		}
		select {
		case <-ctx.Done():
			t.Fatal("completion writers did not reach physical lock barrier")
		default:
			runtime.Gosched()
		}
	}
}

func TestCompletionHistoryConcurrentPersistedState(t *testing.T) {
	for _, order := range []string{"concurrent-first", "completion-before-metadata", "metadata-before-completion"} {
		t.Run(order, func(t *testing.T) {
			clearEquipmentServicesTables(t, testDB)
			ensureUser(t, testDB, "history-author")
			router := newTestRouter(t)
			shop := createShop(t, router, "history-author", "Concurrent completion")
			vehicle := createVehicle(t, router, "history-author", shop)
			list := createList(t, router, "history-author", shop)
			service := createEquipmentService(t, router, "history-author", shop, vehicle, list, "original", nil, false)
			ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
			defer cancel()
			tx, err := testDB.BeginTx(ctx, nil)
			require.NoError(t, err)
			defer tx.Rollback()
			_, err = tx.ExecContext(ctx, `SELECT id FROM shops WHERE id=$1 FOR UPDATE`, shop)
			require.NoError(t, err)
			path := "/api/v1/auth/shops/" + shop + "/equipment-services/" + service
			start := func(isMetadata bool) <-chan *httptest.ResponseRecorder {
				method, url, body := http.MethodPost, path+"/complete", `{}`
				if isMetadata {
					method, url = http.MethodPut, path
					body = `{"service_id":"` + service + `","list_id":"` + list + `","description":"edited","service_type":"inspection","is_completed":true}`
				}
				req, err := http.NewRequestWithContext(ctx, method, url, strings.NewReader(body))
				require.NoError(t, err)
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("X-User-ID", "history-author")
				done := make(chan *httptest.ResponseRecorder, 1)
				go func() {
					resp := httptest.NewRecorder()
					router.ServeHTTP(resp, req)
					done <- resp
				}()
				return done
			}
			first := start(order == "metadata-before-completion")
			waitCompletionWriters(t, ctx, tx, 1)
			second := start(order == "completion-before-metadata")
			waitCompletionWriters(t, ctx, tx, 2)
			released := time.Now()
			require.NoError(t, tx.Commit())
			var firstDate time.Time
			for i, done := range []<-chan *httptest.ResponseRecorder{first, second} {
				select {
				case resp := <-done:
					require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
					data := decodeMap(t, decodeStandardResponse(t, resp.Body).Data)
					date, err := time.Parse(time.RFC3339Nano, data["completion_date"].(string))
					require.NoError(t, err)
					if i == 0 {
						firstDate = date
						require.WithinRange(t, date, released.Add(-time.Microsecond), time.Now())
					} else {
						require.True(t, firstDate.Equal(date), "concurrent retry replaced the first completion date")
					}
				case <-ctx.Done():
					t.Fatal("completion writer did not finish")
				}
			}
			var persisted time.Time
			require.NoError(t, testDB.QueryRow(`SELECT completion_date FROM equipment_services WHERE id=$1`, service).Scan(&persisted))
			require.True(t, firstDate.Equal(persisted))
		})
	}
}

func TestCompletionHistoryCurrentAuthority(t *testing.T) {
	for _, action := range []string{"complete", "metadata"} {
		for _, revoke := range []string{"remove", "demote"} {
			t.Run(action+"/"+revoke, func(t *testing.T) {
				clearEquipmentServicesTables(t, testDB)
				ensureUser(t, testDB, "history-author")
				ensureUser(t, testDB, "history-admin")
				router := newTestRouter(t)
				shop := createShop(t, router, "history-author", "History authority")
				_, err := testDB.Exec(`INSERT INTO shop_members(id,shop_id,user_id,role,joined_at) VALUES($1,$2,'history-admin','admin',NOW())`, uuid.NewString(), shop)
				require.NoError(t, err)
				vehicle := createVehicle(t, router, "history-author", shop)
				list := createList(t, router, "history-author", shop)
				service := createEquipmentService(t, router, "history-author", shop, vehicle, list, "original", nil, true)
				var before, after string
				require.NoError(t, testDB.QueryRow(`SELECT to_jsonb(s)::text FROM equipment_services s WHERE id=$1`, service).Scan(&before))
				ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
				defer cancel()
				tx, err := testDB.BeginTx(ctx, nil)
				require.NoError(t, err)
				defer tx.Rollback()
				if revoke == "remove" {
					_, err = tx.ExecContext(ctx, `DELETE FROM shop_members WHERE shop_id=$1 AND user_id='history-admin'`, shop)
				} else {
					_, err = tx.ExecContext(ctx, `UPDATE shop_members SET role='member' WHERE shop_id=$1 AND user_id='history-admin'`, shop)
				}
				require.NoError(t, err)
				path := "/api/v1/auth/shops/" + shop + "/equipment-services/" + service
				method, body := http.MethodPost, `{}`
				if action == "complete" {
					path += "/complete"
				} else {
					method = http.MethodPut
					body = `{"service_id":"` + service + `","list_id":"` + list + `","description":"edited","service_type":"inspection","is_completed":true}`
				}
				req, err := http.NewRequestWithContext(ctx, method, path, strings.NewReader(body))
				require.NoError(t, err)
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("X-User-ID", "history-admin")
				resp := httptest.NewRecorder()
				done := make(chan struct{})
				go func() { defer close(done); router.ServeHTTP(resp, req) }()
				waitCompletionWriters(t, ctx, tx, 1)
				require.NoError(t, tx.Commit())
				select {
				case <-done:
				case <-ctx.Done():
					t.Fatal("revoked writer did not finish")
				}
				require.GreaterOrEqual(t, resp.Code, 400, resp.Body.String())
				require.NoError(t, testDB.QueryRow(`SELECT to_jsonb(s)::text FROM equipment_services s WHERE id=$1`, service).Scan(&after))
				require.Equal(t, before, after)
			})
		}
	}
}
