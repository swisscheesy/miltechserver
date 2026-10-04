package shops_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/shops/shared"
	"miltechserver/api/shops/vehicles"
	"miltechserver/bootstrap"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// The physical role lock lets HTTP preflight observe the old role while the
// transactional authority check must wait for the committed replacement.
func TestCurrentAuthorityWriterInterleavings(t *testing.T) {
	for _, writer := range []string{"create", "metadata", "absolute", "patch", "delete"} {
		for _, revoke := range []string{"remove", "demote"} {
			if revoke == "demote" && (writer == "create" || writer == "absolute" || writer == "patch") {
				continue
			} // Ordinary members retain usage/create permission.
			t.Run(writer+"/"+revoke, func(t *testing.T) {
				clearShopTables(t, testDB)
				ensureUser(t, testDB, "admin")
				r := newTestRouter(t)
				shop := createShop(t, r, "admin", "authority")
				role := "admin"
				if writer == "absolute" {
					role = "member"
				}
				authorizationMember(t, shop, "writer", role)
				vehicle := createVehicle(t, r, "admin", shop)
				var before string
				require.NoError(t, testDB.QueryRow(`SELECT coalesce(jsonb_agg(to_jsonb(v) ORDER BY id)::text,'[]') FROM shop_vehicle v`).Scan(&before))
				method, path := "PUT", "/api/v1/auth/shops/vehicles"
				body := map[string]any{"vehicle_id": vehicle, "admin": "admin", "model": "changed"}
				switch writer {
				case "create":
					method = "POST"
					body = map[string]any{"shop_id": shop, "admin": "new"}
				case "absolute":
					body = map[string]any{"vehicle_id": vehicle, "admin": "admin", "tracked_mileage": 33}
				case "patch":
					method = "PATCH"
					path += "/" + vehicle + "/usage"
					body = map[string]any{"operation": "add", "mileage_adjustment": 3}
				case "delete":
					method = "DELETE"
					path += "/" + vehicle
					body = nil
				}
				ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
				defer cancel()
				tx, err := testDB.BeginTx(ctx, nil)
				require.NoError(t, err)
				defer tx.Rollback()
				if revoke == "remove" {
					_, err = tx.ExecContext(ctx, `DELETE FROM shop_members WHERE shop_id=$1 AND user_id='writer'`, shop)
				} else {
					_, err = tx.ExecContext(ctx, `UPDATE shop_members SET role='member' WHERE shop_id=$1 AND user_id='writer'`, shop)
				}
				require.NoError(t, err)
				payload, err := json.Marshal(body)
				require.NoError(t, err)
				req, err := http.NewRequestWithContext(ctx, method, path, strings.NewReader(string(payload)))
				require.NoError(t, err)
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("X-User-ID", "writer")
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
				var after string
				require.NoError(t, testDB.QueryRow(`SELECT coalesce(jsonb_agg(to_jsonb(v) ORDER BY id)::text,'[]') FROM shop_vehicle v`).Scan(&after))
				require.Equal(t, before, after, fmt.Sprintf("%s must not persist after %s", writer, revoke))
			})
		}
	}
}

func authorityVehicleWrite(ctx context.Context, writer, shop, vehicle string) error {
	repo := vehicles.NewRepository(testDB)
	user := &bootstrap.User{UserID: "writer"}
	amount := int32(7)
	switch writer {
	case "create":
		_, err := repo.CreateShopVehicle(ctx, user, model.ShopVehicle{ID: uuid.NewString(), ShopID: shop, CreatorID: user.UserID, Admin: "new", SaveTime: time.Now(), LastUpdated: time.Now()})
		return err
	case "metadata":
		name := "changed"
		return repo.UpdateShopVehicleMetadata(ctx, user, vehicles.VehicleUpdateInput{Metadata: vehicles.VehicleMetadataUpdate{VehicleID: vehicle, Model: &name}})
	case "absolute":
		return repo.UpdateShopVehicleUsage(ctx, user, vehicles.ShopVehicleUsageUpdate{VehicleID: vehicle, TrackedMileage: &amount, LastUpdated: time.Now()})
	case "patch":
		_, err := repo.AdjustShopVehicleUsage(ctx, user, vehicles.UsageAdjustment{VehicleID: vehicle, Operation: vehicles.UsageOperationAdd, MileageAdjustment: &amount, LastUpdated: time.Now()})
		return err
	case "delete":
		return repo.DeleteShopVehicle(ctx, user, vehicle)
	default:
		panic("unknown writer")
	}
}

func waitAuthorityBlock(t *testing.T, tx *sql.Tx, ctx context.Context) {
	t.Helper()
	require.Eventually(t, func() bool {
		var blocked bool
		err := tx.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM pg_stat_activity WHERE pg_backend_pid() = ANY(pg_blocking_pids(pid)))`).Scan(&blocked)
		return err == nil && blocked
	}, 3*time.Second, 10*time.Millisecond)
}

func TestCurrentAuthorityVehicleCancellation(t *testing.T) {
	for _, writer := range []string{"create", "metadata", "absolute", "patch", "delete"} {
		for _, boundary := range []string{"pool", "shop", "vehicle"} {
			if writer == "create" && boundary == "vehicle" {
				continue
			}
			t.Run(writer+"/"+boundary, func(t *testing.T) {
				clearShopTables(t, testDB)
				ensureUser(t, testDB, "admin")
				r := newTestRouter(t)
				shop := createShop(t, r, "admin", "cancel")
				authorizationMember(t, shop, "writer", "admin")
				vehicle := createVehicle(t, r, "admin", shop)
				var before, after string
				require.NoError(t, testDB.QueryRow(`SELECT coalesce(jsonb_agg(to_jsonb(v) ORDER BY id)::text,'[]') FROM shop_vehicle v`).Scan(&before))
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
				if boundary == "vehicle" {
					_, err = tx.ExecContext(outer, `SELECT id FROM shop_vehicle WHERE id=$1 FOR UPDATE`, vehicle)
				}
				require.NoError(t, err)
				ctx, cancel := context.WithCancel(outer)
				defer cancel()
				done := make(chan error, 1)
				waits := testDB.Stats().WaitCount
				go func() { done <- authorityVehicleWrite(ctx, writer, shop, vehicle) }()
				if boundary == "pool" {
					require.Eventually(t, func() bool { return testDB.Stats().WaitCount > waits }, time.Second, time.Millisecond)
				} else {
					waitAuthorityBlock(t, tx, outer)
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
				require.NoError(t, testDB.QueryRow(`SELECT coalesce(jsonb_agg(to_jsonb(v) ORDER BY id)::text,'[]') FROM shop_vehicle v`).Scan(&after))
				require.Equal(t, before, after)
			})
		}
	}
}

func TestCurrentAuthorityVehicleAuthorizedFirst(t *testing.T) {
	for _, writer := range []string{"create", "metadata", "absolute", "patch", "delete"} {
		t.Run(writer, func(t *testing.T) {
			clearShopTables(t, testDB)
			ensureUser(t, testDB, "admin")
			r := newTestRouter(t)
			shop := createShop(t, r, "admin", "authorized first")
			authorizationMember(t, shop, "writer", "admin")
			vehicle := createVehicle(t, r, "admin", shop)
			ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
			defer cancel()
			blocker, err := testDB.BeginTx(ctx, nil)
			require.NoError(t, err)
			defer blocker.Rollback()
			// SHARE permits the authorization reads/row locks, but stops actual DML.
			_, err = blocker.ExecContext(ctx, `LOCK TABLE shop_vehicle IN SHARE MODE`)
			require.NoError(t, err)
			writeDone := make(chan error, 1)
			go func() { writeDone <- authorityVehicleWrite(ctx, writer, shop, vehicle) }()
			waitAuthorityBlock(t, blocker, ctx)
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
				_, _, err = shared.LockShopMutation(ctx, tx, shop, "admin")
				if err == nil {
					_, err = tx.ExecContext(ctx, `DELETE FROM shop_members WHERE shop_id=$1 AND user_id='writer'`, shop)
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
			var members int
			require.NoError(t, testDB.QueryRow(`SELECT count(*) FROM shop_members WHERE shop_id=$1 AND user_id='writer'`, shop).Scan(&members))
			require.Zero(t, members)
			var count int
			require.NoError(t, testDB.QueryRow(`SELECT count(*) FROM shop_vehicle WHERE shop_id=$1`, shop).Scan(&count))
			if writer == "create" {
				require.Equal(t, 2, count)
			} else if writer == "delete" {
				require.Zero(t, count)
			} else {
				require.Equal(t, 1, count)
				var current model.ShopVehicle
				currentPtr, err := vehicles.NewRepository(testDB).GetShopVehicleByID(ctx, &bootstrap.User{UserID: "admin"}, vehicle)
				require.NoError(t, err)
				current = *currentPtr
				if writer == "metadata" {
					require.Equal(t, "changed", current.Model)
				} else {
					require.NotNil(t, current.TrackedMileage)
					require.Equal(t, int32(7), *current.TrackedMileage)
				}
			}
		})
	}
}
