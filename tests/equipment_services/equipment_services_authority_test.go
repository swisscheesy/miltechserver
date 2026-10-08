package equipment_services_test

import (
	"context"
	"database/sql"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"miltechserver/api/equipment_services/completion"
	"miltechserver/api/equipment_services/core"
	shopshared "miltechserver/api/shops/shared"
	"miltechserver/bootstrap"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestEquipmentServiceActualShop(t *testing.T) {
	for _, action := range []string{"complete", "delete", "update"} {
		t.Run(action, func(t *testing.T) {
			clearEquipmentServicesTables(t, testDB)
			ensureUser(t, testDB, "admin")
			ensureUser(t, testDB, "actor")
			r := newTestRouter(t)
			a := createShop(t, r, "actor", "A")
			b := createShop(t, r, "admin", "B")
			_, err := testDB.Exec(`INSERT INTO shop_members(id,shop_id,user_id,role,joined_at) VALUES($1,$2,'actor','member',NOW())`, uuid.NewString(), b)
			require.NoError(t, err)
			v := createVehicle(t, r, "admin", b)
			l := createList(t, r, "admin", b)
			s := createEquipmentService(t, r, "admin", b, v, l, "original", nil, false)
			method, path, body := http.MethodPost, "/api/v1/auth/shops/"+a+"/equipment-services/"+s+"/complete", any(map[string]any{})
			if action == "delete" {
				method = http.MethodDelete
				path = "/api/v1/auth/shops/" + a + "/equipment-services/" + s
				body = nil
			}
			if action == "update" {
				method = http.MethodPut
				path = "/api/v1/auth/shops/" + a + "/equipment-services/" + s
				body = map[string]any{"service_id": s, "description": "bad", "service_type": "inspection", "list_id": l}
			}
			var before, after string
			require.NoError(t, testDB.QueryRow(`SELECT to_jsonb(s)::text FROM equipment_services s WHERE id=$1`, s).Scan(&before))
			resp := doJSONRequest(t, r, method, path, body, "actor")
			require.GreaterOrEqual(t, resp.Code, 400, resp.Body.String())
			require.NoError(t, testDB.QueryRow(`SELECT to_jsonb(s)::text FROM equipment_services s WHERE id=$1`, s).Scan(&after))
			require.Equal(t, before, after)
		})
	}
}

func TestCurrentAuthorityEquipmentInterleavings(t *testing.T) {
	for _, action := range []string{"complete", "delete"} {
		for _, revoke := range []string{"remove", "demote"} {
			t.Run(action+"/"+revoke, func(t *testing.T) {
				clearEquipmentServicesTables(t, testDB)
				ensureUser(t, testDB, "admin")
				ensureUser(t, testDB, "actor")
				r := newTestRouter(t)
				shop := createShop(t, r, "admin", "authority")
				_, err := testDB.Exec(`INSERT INTO shop_members(id,shop_id,user_id,role,joined_at) VALUES($1,$2,'actor','admin',NOW())`, uuid.NewString(), shop)
				require.NoError(t, err)
				v := createVehicle(t, r, "admin", shop)
				l := createList(t, r, "admin", shop)
				s := createEquipmentService(t, r, "admin", shop, v, l, "original", nil, false)
				var before, after string
				require.NoError(t, testDB.QueryRow(`SELECT to_jsonb(s)::text FROM equipment_services s WHERE id=$1`, s).Scan(&before))
				ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
				defer cancel()
				tx, err := testDB.BeginTx(ctx, nil)
				require.NoError(t, err)
				defer tx.Rollback()
				if revoke == "remove" {
					_, err = tx.ExecContext(ctx, `DELETE FROM shop_members WHERE shop_id=$1 AND user_id='actor'`, shop)
				} else {
					_, err = tx.ExecContext(ctx, `UPDATE shop_members SET role='member' WHERE shop_id=$1 AND user_id='actor'`, shop)
				}
				require.NoError(t, err)
				method, path := http.MethodPost, "/api/v1/auth/shops/"+shop+"/equipment-services/"+s+"/complete"
				if action == "delete" {
					method = http.MethodDelete
					path = "/api/v1/auth/shops/" + shop + "/equipment-services/" + s
				}
				req, err := http.NewRequestWithContext(ctx, method, path, strings.NewReader(`{}`))
				require.NoError(t, err)
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("X-User-ID", "actor")
				resp := httptest.NewRecorder()
				done := make(chan struct{})
				go func() { defer close(done); r.ServeHTTP(resp, req) }()
				require.Eventually(t, func() bool {
					select {
					case <-done:
						return true
					default:
					}
					var blocked bool
					err := tx.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM pg_stat_activity WHERE pg_backend_pid() = ANY(pg_blocking_pids(pid)))`).Scan(&blocked)
					return err == nil && blocked
				}, 3*time.Second, 10*time.Millisecond)
				require.NoError(t, tx.Commit())
				select {
				case <-done:
				case <-ctx.Done():
					t.Fatal("writer did not finish")
				}
				require.GreaterOrEqual(t, resp.Code, 400, resp.Body.String())
				require.NoError(t, testDB.QueryRow(`SELECT to_jsonb(s)::text FROM equipment_services s WHERE id=$1`, s).Scan(&after))
				require.Equal(t, before, after)
			})
		}
	}
}

func authorityEquipmentWrite(ctx context.Context, action, shop, service string) error {
	user := &bootstrap.User{UserID: "actor"}
	if action == "complete" {
		_, err := completion.NewRepository(testDB).Complete(ctx, user, shop, service, nil)
		return err
	}
	return core.NewRepository(testDB).Delete(ctx, user, shop, service)
}

func waitEquipmentAuthorityBlock(t *testing.T, tx *sql.Tx, ctx context.Context) {
	t.Helper()
	require.Eventually(t, func() bool {
		var blocked bool
		err := tx.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM pg_stat_activity WHERE pg_backend_pid() = ANY(pg_blocking_pids(pid)))`).Scan(&blocked)
		return err == nil && blocked
	}, 3*time.Second, 10*time.Millisecond)
}

func TestCurrentAuthorityEquipmentCancellation(t *testing.T) {
	for _, action := range []string{"complete", "delete"} {
		for _, boundary := range []string{"pool", "shop", "service"} {
			t.Run(action+"/"+boundary, func(t *testing.T) {
				clearEquipmentServicesTables(t, testDB)
				ensureUser(t, testDB, "admin")
				ensureUser(t, testDB, "actor")
				r := newTestRouter(t)
				shop := createShop(t, r, "admin", "cancel")
				_, err := testDB.Exec(`INSERT INTO shop_members(id,shop_id,user_id,role,joined_at) VALUES($1,$2,'actor','admin',NOW())`, uuid.NewString(), shop)
				require.NoError(t, err)
				v := createVehicle(t, r, "admin", shop)
				l := createList(t, r, "admin", shop)
				s := createEquipmentService(t, r, "admin", shop, v, l, "original", nil, false)
				var before, after string
				require.NoError(t, testDB.QueryRow(`SELECT to_jsonb(s)::text FROM equipment_services s WHERE id=$1`, s).Scan(&before))
				outer, stop := context.WithTimeout(context.Background(), 8*time.Second)
				defer stop()
				if boundary == "pool" {
					// TestMain reserves one connection for its advisory lock. Drain idle
					// connections before reducing the pool, then occupy the only other slot.
					testDB.SetMaxIdleConns(0)
					defer testDB.SetMaxIdleConns(2)
					testDB.SetMaxOpenConns(2)
					defer testDB.SetMaxOpenConns(0)
				}
				tx, err := testDB.BeginTx(outer, nil)
				require.NoError(t, err)
				defer tx.Rollback()
				if boundary == "shop" {
					_, err = tx.ExecContext(outer, `SELECT id FROM shops WHERE id=$1 FOR UPDATE`, shop)
				}
				if boundary == "service" {
					_, err = tx.ExecContext(outer, `SELECT id FROM equipment_services WHERE id=$1 FOR UPDATE`, s)
				}
				require.NoError(t, err)
				ctx, cancel := context.WithCancel(outer)
				defer cancel()
				done := make(chan error, 1)
				waits := testDB.Stats().WaitCount
				go func() { done <- authorityEquipmentWrite(ctx, action, shop, s) }()
				if boundary == "pool" {
					require.Eventually(t, func() bool { return testDB.Stats().WaitCount > waits }, time.Second, time.Millisecond)
				} else {
					waitEquipmentAuthorityBlock(t, tx, outer)
				}
				cancel()
				select {
				case err := <-done:
					require.Error(t, err)
				case <-time.After(time.Second):
					t.Fatal("cancelled writer remained blocked")
				}
				require.NoError(t, tx.Rollback())
				if boundary == "pool" {
					testDB.SetMaxOpenConns(0)
				}
				require.NoError(t, testDB.QueryRow(`SELECT to_jsonb(s)::text FROM equipment_services s WHERE id=$1`, s).Scan(&after))
				require.Equal(t, before, after)
			})
		}
	}
}

func TestCurrentAuthorityEquipmentAuthorizedFirst(t *testing.T) {
	for _, action := range []string{"complete", "delete"} {
		t.Run(action, func(t *testing.T) {
			clearEquipmentServicesTables(t, testDB)
			ensureUser(t, testDB, "admin")
			ensureUser(t, testDB, "actor")
			r := newTestRouter(t)
			shop := createShop(t, r, "admin", "authorized first")
			_, err := testDB.Exec(`INSERT INTO shop_members(id,shop_id,user_id,role,joined_at) VALUES($1,$2,'actor','admin',NOW())`, uuid.NewString(), shop)
			require.NoError(t, err)
			v := createVehicle(t, r, "admin", shop)
			l := createList(t, r, "admin", shop)
			s := createEquipmentService(t, r, "admin", shop, v, l, "original", nil, false)
			ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
			defer cancel()
			blocker, err := testDB.BeginTx(ctx, nil)
			require.NoError(t, err)
			defer blocker.Rollback()
			_, err = blocker.ExecContext(ctx, `LOCK TABLE equipment_services IN SHARE MODE`)
			require.NoError(t, err)
			writeDone := make(chan error, 1)
			go func() { writeDone <- authorityEquipmentWrite(ctx, action, shop, s) }()
			waitEquipmentAuthorityBlock(t, blocker, ctx)
			remover, err := testDB.Conn(ctx)
			require.NoError(t, err)
			defer remover.Close()
			var pid int
			require.NoError(t, remover.QueryRowContext(ctx, `SELECT pg_backend_pid()`).Scan(&pid))
			removeDone := make(chan error, 1)
			go func() {
				tx, err := remover.BeginTx(ctx, nil)
				if err != nil {
					removeDone <- err
					return
				}
				defer tx.Rollback()
				_, _, err = shopshared.LockShopMutation(ctx, tx, shop, "admin")
				if err == nil {
					_, err = tx.ExecContext(ctx, `DELETE FROM shop_members WHERE shop_id=$1 AND user_id='actor'`, shop)
				}
				if err == nil {
					err = tx.Commit()
				}
				removeDone <- err
			}()
			require.Eventually(t, func() bool {
				var blocked bool
				err := blocker.QueryRowContext(ctx, `SELECT cardinality(pg_blocking_pids($1))>0`, pid).Scan(&blocked)
				return err == nil && blocked
			}, time.Second, 10*time.Millisecond)
			require.NoError(t, blocker.Commit())
			require.NoError(t, <-writeDone)
			require.NoError(t, <-removeDone)
			var count int
			require.NoError(t, testDB.QueryRow(`SELECT count(*) FROM equipment_services WHERE id=$1 AND is_completed=true`, s).Scan(&count))
			if action == "complete" {
				require.Equal(t, 1, count)
			} else {
				require.Zero(t, count)
				require.NoError(t, testDB.QueryRow(`SELECT count(*) FROM equipment_services WHERE id=$1`, s).Scan(&count))
				require.Zero(t, count)
			}
		})
	}
}

func TestEquipmentServiceActualShopCreateAndRead(t *testing.T) {
	for _, action := range []string{"create", "read"} {
		t.Run(action, func(t *testing.T) {
			clearEquipmentServicesTables(t, testDB)
			ensureUser(t, testDB, "actor")
			r := newTestRouter(t)
			a := createShop(t, r, "actor", "A")
			b := createShop(t, r, "actor", "B")
			v := createVehicle(t, r, "actor", b)
			l := createList(t, r, "actor", b)
			s := createEquipmentService(t, r, "actor", b, v, l, "original", nil, false)
			method, suffix, body := http.MethodGet, "/equipment-services/"+s, any(nil)
			if action == "create" {
				method = http.MethodPost
				suffix = "/equipment-services"
				body = map[string]any{"equipment_id": v, "list_id": l, "description": "new", "service_type": "inspection"}
			}
			var before, after string
			require.NoError(t, testDB.QueryRow(`SELECT jsonb_agg(to_jsonb(s) ORDER BY id)::text FROM equipment_services s`).Scan(&before))
			denied := doJSONRequest(t, r, method, "/api/v1/auth/shops/"+a+suffix, body, "actor")
			require.GreaterOrEqual(t, denied.Code, 400, denied.Body.String())
			require.NoError(t, testDB.QueryRow(`SELECT jsonb_agg(to_jsonb(s) ORDER BY id)::text FROM equipment_services s`).Scan(&after))
			require.Equal(t, before, after)
			allowed := doJSONRequest(t, r, method, "/api/v1/auth/shops/"+b+suffix, body, "actor")
			if action == "create" {
				require.Equal(t, 201, allowed.Code, allowed.Body.String())
			} else {
				require.Equal(t, 200, allowed.Code, allowed.Body.String())
			}
			result := decodeMap(t, decodeStandardResponse(t, allowed.Body).Data)
			require.Equal(t, b, result["shop_id"])
		})
	}
}
