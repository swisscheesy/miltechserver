package shops_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/request"
	"miltechserver/api/shops/core"
	"miltechserver/api/shops/settings"
	"miltechserver/bootstrap"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestShopCreateMembershipAtomic(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "membership-failure")
	_, err := testDB.Exec(`ALTER TABLE shop_members ADD CONSTRAINT task9_membership_failure CHECK (user_id <> 'membership-failure')`)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, err := testDB.Exec(`ALTER TABLE shop_members DROP CONSTRAINT task9_membership_failure`)
		require.NoError(t, err)
	})
	resp := doJSONRequest(t, newTestRouter(t), http.MethodPost, "/api/v1/auth/shops", map[string]any{"name": "must rollback"}, "membership-failure")
	var shopsAfterFailedCreate, membersAfterFailedCreate int
	require.NoError(t, testDB.QueryRow(`SELECT count(*) FROM shops`).Scan(&shopsAfterFailedCreate))
	require.NoError(t, testDB.QueryRow(`SELECT count(*) FROM shop_members`).Scan(&membersAfterFailedCreate))
	require.Zero(t, shopsAfterFailedCreate)
	require.Zero(t, membersAfterFailedCreate)
	require.GreaterOrEqual(t, resp.Code, 400, resp.Body.String())
}

func TestShopRestrictionIntent(t *testing.T) {
	t.Run("create true", func(t *testing.T) {
		clearShopTables(t, testDB)
		ensureUser(t, testDB, "admin")
		resp := doJSONRequest(t, newTestRouter(t), "POST", "/api/v1/auth/shops", map[string]any{"name": "restricted", "admin_only_lists": true}, "admin")
		require.Equal(t, 201, resp.Code)
		var restriction bool
		require.NoError(t, testDB.QueryRow(`SELECT admin_only_lists FROM shops`).Scan(&restriction))
		require.True(t, restriction)
	})
	t.Run("rename preserves details and restriction", func(t *testing.T) {
		clearShopTables(t, testDB)
		ensureUser(t, testDB, "admin")
		r := newTestRouter(t)
		shop := createShop(t, r, "admin", "restricted")
		_, err := testDB.Exec(`UPDATE shops SET admin_only_lists=true WHERE id=$1`, shop)
		require.NoError(t, err)
		resp := doJSONRequest(t, r, "PUT", "/api/v1/auth/shops/"+shop, map[string]any{"name": "renamed", "admin_only_lists": false}, "admin")
		require.Equal(t, 200, resp.Code)
		var restrictionAfterRename bool
		var details string
		require.NoError(t, testDB.QueryRow(`SELECT admin_only_lists,details FROM shops WHERE id=$1`, shop).Scan(&restrictionAfterRename, &details))
		require.True(t, restrictionAfterRename)
		require.Equal(t, "Details", details)
	})
	t.Run("nullable details intent", func(t *testing.T) {
		clearShopTables(t, testDB)
		ensureUser(t, testDB, "admin")
		r := newTestRouter(t)
		shop := createShop(t, r, "admin", "metadata")
		for _, test := range []struct {
			body    map[string]any
			details string
		}{
			{map[string]any{"name": "omitted"}, "Details"},
			{map[string]any{"name": "null", "details": nil}, "Details"},
			{map[string]any{"name": "empty", "details": ""}, ""},
			{map[string]any{"name": "replacement", "details": "new"}, "new"},
		} {
			resp := doJSONRequest(t, r, "PUT", "/api/v1/auth/shops/"+shop, test.body, "admin")
			require.Equal(t, 200, resp.Code)
			var details string
			require.NoError(t, testDB.QueryRow(`SELECT details FROM shops WHERE id=$1`, shop).Scan(&details))
			require.Equal(t, test.details, details)
		}
		resp := doJSONRequest(t, r, "PUT", "/api/v1/auth/shops/"+shop, map[string]any{"details": "missing name"}, "admin")
		require.Equal(t, 400, resp.Code)
	})

	for _, endpoint := range []string{"settings/admin-only-lists", "settings"} {
		t.Run(endpoint, func(t *testing.T) {
			clearShopTables(t, testDB)
			ensureUser(t, testDB, "admin")
			r := newTestRouter(t)
			shop := createShop(t, r, "admin", "policy")
			path := "/api/v1/auth/shops/" + shop + "/" + endpoint
			resp := doJSONRequest(t, r, "PUT", path, map[string]any{"admin_only_lists": true}, "admin")
			require.Equal(t, 200, resp.Code)
			for _, body := range []map[string]any{{}, {"admin_only_lists": nil}} {
				resp = doJSONRequest(t, r, "PUT", path, body, "admin")
				require.Equal(t, 400, resp.Code)
				var restriction bool
				require.NoError(t, testDB.QueryRow(`SELECT admin_only_lists FROM shops WHERE id=$1`, shop).Scan(&restriction))
				require.True(t, restriction)
			}
			resp = doJSONRequest(t, r, "PUT", path, map[string]any{"admin_only_lists": false}, "admin")
			require.Equal(t, 200, resp.Code)
			var restrictionAfterExplicitFalse bool
			require.NoError(t, testDB.QueryRow(`SELECT admin_only_lists FROM shops WHERE id=$1`, shop).Scan(&restrictionAfterExplicitFalse))
			require.False(t, restrictionAfterExplicitFalse)
		})
	}
}

func TestPromotedAdminRenameCurrentAuthority(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "creator")
	r := newTestRouter(t)
	shop := createShop(t, r, "creator", "original")
	authorizationMember(t, shop, "promoted", "admin")
	resp := doJSONRequest(t, r, "PUT", "/api/v1/auth/shops/"+shop, map[string]any{"name": "promoted rename"}, "promoted")
	require.Equal(t, 200, resp.Code, resp.Body.String())
	var name string
	require.NoError(t, testDB.QueryRow(`SELECT name FROM shops WHERE id=$1`, shop).Scan(&name))
	require.Equal(t, "promoted rename", name)
}

func TestPromotedAdminRenameSettingsRemovalFirst(t *testing.T) {
	for _, writer := range []string{"rename", "settings", "settings/admin-only-lists"} {
		t.Run(writer, func(t *testing.T) {
			clearShopTables(t, testDB)
			ensureUser(t, testDB, "creator")
			r := newTestRouter(t)
			shop := createShop(t, r, "creator", "original")
			// Use the creator so the old creator predicate cannot mask stale authority.
			var before string
			require.NoError(t, testDB.QueryRow(`SELECT to_jsonb(s)::text FROM shops s WHERE id=$1`, shop).Scan(&before))
			ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
			defer cancel()
			tx, err := testDB.BeginTx(ctx, nil)
			require.NoError(t, err)
			defer tx.Rollback()
			_, err = tx.ExecContext(ctx, `DELETE FROM shop_members WHERE shop_id=$1 AND user_id='creator'`, shop)
			require.NoError(t, err)
			path := "/api/v1/auth/shops/" + shop
			body := map[string]any{"name": "unauthorized rename"}
			if writer != "rename" {
				path += "/" + writer
				body = map[string]any{"admin_only_lists": true}
			}
			payload, err := json.Marshal(body)
			require.NoError(t, err)
			req, err := http.NewRequestWithContext(ctx, "PUT", path, strings.NewReader(string(payload)))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-User-ID", "creator")
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
				t.Fatal("request remained blocked")
			}
			require.GreaterOrEqual(t, resp.Code, 400, resp.Body.String())
			var after string
			require.NoError(t, testDB.QueryRow(`SELECT to_jsonb(s)::text FROM shops s WHERE id=$1`, shop).Scan(&after))
			require.Equal(t, before, after)
		})
	}
}

func TestShopCoreSettingsCancellation(t *testing.T) {
	user := &bootstrap.User{UserID: "admin"}
	coreRepo := core.NewRepository(testDB, nil, nil)
	settingsRepo := settings.NewRepository(testDB)
	for _, boundary := range []string{"pool", "database"} {
		for _, operation := range []string{"create", "rename", "delete", "shops", "detail", "stats", "overview", "setting-read", "settings-read", "setting-write", "settings-write"} {
			t.Run(boundary+"/"+operation, func(t *testing.T) {
				clearShopTables(t, testDB)
				ensureUser(t, testDB, "admin")
				shop := createShop(t, newTestRouter(t), "admin", "cancellation")
				var before string
				require.NoError(t, testDB.QueryRow(`SELECT to_jsonb(s)::text FROM shops s WHERE id=$1`, shop).Scan(&before))
				outer, stop := context.WithTimeout(context.Background(), 8*time.Second)
				defer stop()
				if boundary == "pool" {
					testDB.SetMaxIdleConns(0)
					defer testDB.SetMaxIdleConns(2)
					testDB.SetMaxOpenConns(2)
					defer testDB.SetMaxOpenConns(0)
				}
				tx, err := testDB.BeginTx(outer, nil)
				require.NoError(t, err)
				defer tx.Rollback()
				if boundary == "database" {
					// ACCESS EXCLUSIVE blocks both reads and writes at actual statement execution.
					_, err = tx.ExecContext(outer, `LOCK TABLE shops, shop_members IN ACCESS EXCLUSIVE MODE`)
					require.NoError(t, err)
				}
				ctx, cancel := context.WithCancel(outer)
				defer cancel()
				done := make(chan error, 1)
				waits := testDB.Stats().WaitCount
				go func() {
					var err error
					switch operation {
					case "create":
						_, err = coreRepo.CreateShop(ctx, user, model.Shops{ID: uuid.NewString(), Name: "cancelled", CreatedBy: user.UserID})
					case "rename":
						_, err = coreRepo.UpdateShop(ctx, user, model.Shops{ID: shop, Name: "cancelled"})
					case "delete":
						err = coreRepo.DeleteShop(ctx, user, shop)
					case "shops":
						_, err = coreRepo.GetShopsByUser(ctx, user)
					case "detail":
						_, err = coreRepo.GetShopByID(ctx, user, shop)
					case "stats":
						_, err = coreRepo.GetShopsWithStatsForUser(ctx, user)
					case "overview":
						_, err = coreRepo.GetShopEquipmentOverview(ctx, user)
					case "setting-read":
						_, err = settingsRepo.GetShopAdminOnlyListsSetting(ctx, shop)
					case "settings-read":
						_, err = settingsRepo.GetShopSettings(ctx, shop)
					case "setting-write":
						err = settingsRepo.UpdateShopAdminOnlyListsSetting(ctx, user, shop, true)
					case "settings-write":
						value := true
						err = settingsRepo.UpdateShopSettings(ctx, user, shop, request.UpdateShopSettingsRequest{AdminOnlyLists: &value})
					}
					done <- err
				}()
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
					t.Fatal("cancelled repository operation remained blocked")
				}
				require.NoError(t, tx.Rollback())
				if boundary == "pool" {
					testDB.SetMaxOpenConns(0)
				}
				var after string
				require.NoError(t, testDB.QueryRow(`SELECT to_jsonb(s)::text FROM shops s WHERE id=$1`, shop).Scan(&after))
				require.Equal(t, before, after)
				var members int
				require.NoError(t, testDB.QueryRow(`SELECT count(*) FROM shop_members WHERE shop_id=$1 AND role='admin'`, shop).Scan(&members))
				require.Equal(t, 1, members)
				var count int
				require.NoError(t, testDB.QueryRow(`SELECT count(*) FROM shops`).Scan(&count))
				require.Equal(t, 1, count)
			})
		}
	}
}
