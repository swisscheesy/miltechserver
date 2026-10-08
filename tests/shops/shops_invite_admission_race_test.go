package shops_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/shops/members"
	"miltechserver/api/shops/members/invites"
	"miltechserver/api/shops/shared"
	"miltechserver/bootstrap"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func inviteRequest(t *testing.T, ctx context.Context, router http.Handler, method, path, user string, body any) (*httptest.ResponseRecorder, <-chan struct{}) {
	t.Helper()
	payload, err := json.Marshal(body)
	require.NoError(t, err)
	req, err := http.NewRequestWithContext(ctx, method, path, strings.NewReader(string(payload)))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", user)
	resp := httptest.NewRecorder()
	done := make(chan struct{})
	go func() { defer close(done); router.ServeHTTP(resp, req) }()
	return resp, done
}

func finishInviteRequest(t *testing.T, done <-chan struct{}) {
	t.Helper()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("invite request did not finish")
	}
}

func inviteFixture(t *testing.T) (*gin.Engine, string, string, string) {
	t.Helper()
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "owner")
	ensureUser(t, testDB, "claimant")
	r := newTestRouter(t)
	shop := createShop(t, r, "owner", "admission")
	id, code := createInviteCode(t, r, "owner", shop)
	return r, shop, id, code
}

func lockInviteShop(t *testing.T, ctx context.Context, shop string) *sql.Tx {
	t.Helper()
	tx, err := testDB.BeginTx(ctx, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = tx.Rollback() })
	_, err = tx.ExecContext(ctx, `SELECT id FROM shops WHERE id=$1 FOR UPDATE`, shop)
	require.NoError(t, err)
	return tx
}

func TestInviteRevocationWinsClaim(t *testing.T) {
	for _, action := range []string{"revoke", "delete"} {
		t.Run(action, func(t *testing.T) {
			r, shop, id, code := inviteFixture(t)
			ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
			defer cancel()
			tx := lockInviteShop(t, ctx, shop)
			query := `UPDATE shop_invite_codes SET is_active=false WHERE id=$1`
			if action == "delete" {
				query = `DELETE FROM shop_invite_codes WHERE id=$1`
			}
			_, err := tx.ExecContext(ctx, query, id)
			require.NoError(t, err)
			resp, done := inviteRequest(t, ctx, r, "POST", "/api/v1/auth/shops/join", "claimant", map[string]any{"invite_code": code})
			waitAuthorityBlock(t, tx, ctx)
			require.NoError(t, tx.Commit())
			finishInviteRequest(t, done)
			var count int
			require.NoError(t, testDB.QueryRow(`SELECT count(*) FROM shop_members WHERE shop_id=$1 AND user_id='claimant'`, shop).Scan(&count))
			require.Zero(t, count, "claim must recheck the committed invite state")
			require.GreaterOrEqual(t, resp.Code, 400, resp.Body.String())
		})
	}
}

func TestDuplicateClaimPreservesPromotion(t *testing.T) {
	r, shop, _, code := inviteFixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	tx := lockInviteShop(t, ctx, shop)
	_, err := tx.ExecContext(ctx, `INSERT INTO shop_members(id,shop_id,user_id,role) VALUES ($1,$2,'claimant','member')`, shop+"_claimant", shop)
	require.NoError(t, err)
	resp, done := inviteRequest(t, ctx, r, "POST", "/api/v1/auth/shops/join", "claimant", map[string]any{"invite_code": code})
	waitAuthorityBlock(t, tx, ctx)
	_, err = tx.ExecContext(ctx, `UPDATE shop_members SET role='admin' WHERE shop_id=$1 AND user_id='claimant'`, shop)
	require.NoError(t, err)
	require.NoError(t, tx.Commit())
	finishInviteRequest(t, done)
	var role string
	require.NoError(t, testDB.QueryRow(`SELECT role FROM shop_members WHERE shop_id=$1 AND user_id='claimant'`, shop).Scan(&role))
	require.Equal(t, "admin", role)
	require.GreaterOrEqual(t, resp.Code, 400, resp.Body.String())
}

func TestUnsupportedInviteControls(t *testing.T) {
	for _, fields := range []map[string]any{{"max_uses": 0}, {"max_uses": 1}, {"max_uses": -1}, {"expires_at": ""}, {"expires_at": "2099-01-01T00:00:00Z"}} {
		t.Run(strings.TrimSpace(string(mustInviteJSON(t, fields))), func(t *testing.T) {
			r, shop, _, _ := inviteFixture(t)
			fields["shop_id"] = shop
			resp := doJSONRequest(t, r, "POST", "/api/v1/auth/shops/invite-codes", fields, "owner")
			require.Equal(t, 400, resp.Code, resp.Body.String())
			var count int
			require.NoError(t, testDB.QueryRow(`SELECT count(*) FROM shop_invite_codes WHERE shop_id=$1`, shop).Scan(&count))
			require.Equal(t, 1, count)
		})
	}
}

func mustInviteJSON(t *testing.T, body any) []byte {
	t.Helper()
	data, err := json.Marshal(body)
	require.NoError(t, err)
	return data
}

func TestRemovalAllowsRejoin(t *testing.T) {
	r, shop, _, code := inviteFixture(t)
	for i := 0; i < 2; i++ {
		resp := doJSONRequest(t, r, "POST", "/api/v1/auth/shops/join", map[string]any{"invite_code": code}, "claimant")
		require.Equal(t, 200, resp.Code, resp.Body.String())
		if i == 0 {
			resp = doJSONRequest(t, r, "DELETE", "/api/v1/auth/shops/members/remove", map[string]any{"shop_id": shop, "target_user_id": "claimant"}, "owner")
			require.Equal(t, 200, resp.Code, resp.Body.String())
		}
	}
}

func TestInviteCreateRemovalFirst(t *testing.T) {
	r, shop, _, _ := inviteFixture(t)
	authorizationMember(t, shop, "claimant", "member")
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	tx := lockInviteShop(t, ctx, shop)
	_, err := tx.ExecContext(ctx, `DELETE FROM shop_members WHERE shop_id=$1 AND user_id='claimant'`, shop)
	require.NoError(t, err)
	resp, done := inviteRequest(t, ctx, r, "POST", "/api/v1/auth/shops/invite-codes", "claimant", map[string]any{"shop_id": shop})
	waitAuthorityBlock(t, tx, ctx)
	require.NoError(t, tx.Commit())
	finishInviteRequest(t, done)
	var count int
	require.NoError(t, testDB.QueryRow(`SELECT count(*) FROM shop_invite_codes WHERE shop_id=$1 AND created_by='claimant'`, shop).Scan(&count))
	require.Zero(t, count)
	require.GreaterOrEqual(t, resp.Code, 400, resp.Body.String())
}

func TestInviteClaimFirst(t *testing.T) {
	for _, action := range []string{"revoke", "delete"} {
		t.Run(action, func(t *testing.T) {
			r, shop, id, code := inviteFixture(t)
			ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
			defer cancel()
			tx, err := testDB.BeginTx(ctx, nil)
			require.NoError(t, err)
			defer tx.Rollback()
			_, err = tx.ExecContext(ctx, `LOCK TABLE shop_members IN SHARE MODE`)
			require.NoError(t, err)
			claim, claimDone := inviteRequest(t, ctx, r, "POST", "/api/v1/auth/shops/join", "claimant", map[string]any{"invite_code": code})
			waitAuthorityBlock(t, tx, ctx)
			var claimantPID int
			require.NoError(t, tx.QueryRowContext(ctx, `SELECT pid FROM pg_stat_activity WHERE pg_backend_pid() = ANY(pg_blocking_pids(pid))`).Scan(&claimantPID))
			path := "/api/v1/auth/shops/invite-codes/" + id
			if action == "delete" {
				path += "/delete"
			}
			revoke, revokeDone := inviteRequest(t, ctx, r, "DELETE", path, "owner", nil)
			// Observe the second physical waiter or the old uncoordinated writer's completion.
			require.Eventually(t, func() bool {
				select {
				case <-revokeDone:
					return true
				default:
				}
				var blocked bool
				err := testDB.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM pg_stat_activity WHERE $1 = ANY(pg_blocking_pids(pid)))`, claimantPID).Scan(&blocked)
				return err == nil && blocked
			}, 3*time.Second, 10*time.Millisecond)
			var completedEarly bool
			select {
			case <-revokeDone:
				completedEarly = true
			default:
			}
			require.NoError(t, tx.Commit())
			finishInviteRequest(t, claimDone)
			finishInviteRequest(t, revokeDone)
			require.False(t, completedEarly, "revocation must wait for the admitted claim's Shop lock")
			require.Equal(t, 200, claim.Code, claim.Body.String())
			require.Equal(t, 200, revoke.Code, revoke.Body.String())
			var count int
			require.NoError(t, testDB.QueryRow(`SELECT count(*) FROM shop_members WHERE shop_id=$1 AND user_id='claimant'`, shop).Scan(&count))
			require.Equal(t, 1, count)
		})
	}
}

func TestInviteCancellationAtDatabase(t *testing.T) {
	for _, action := range []string{"join", "create", "list", "revoke", "delete"} {
		t.Run(action, func(t *testing.T) {
			r, shop, id, code := inviteFixture(t)
			outer, stop := context.WithTimeout(context.Background(), 8*time.Second)
			defer stop()
			tx, err := testDB.BeginTx(outer, nil)
			require.NoError(t, err)
			defer tx.Rollback()
			_, err = tx.ExecContext(outer, `LOCK TABLE shop_invite_codes IN ACCESS EXCLUSIVE MODE`)
			require.NoError(t, err)
			method, path, user := "POST", "/api/v1/auth/shops/join", "claimant"
			var body any = map[string]any{"invite_code": code}
			switch action {
			case "create":
				path, user, body = "/api/v1/auth/shops/invite-codes", "owner", map[string]any{"shop_id": shop}
			case "list":
				method, path, user, body = "GET", "/api/v1/auth/shops/"+shop+"/invite-codes", "owner", nil
			case "revoke", "delete":
				method, path, user, body = "DELETE", "/api/v1/auth/shops/invite-codes/"+id, "owner", nil
				if action == "delete" {
					path += "/delete"
				}
			}
			ctx, cancel := context.WithCancel(outer)
			defer cancel()
			resp, done := inviteRequest(t, ctx, r, method, path, user, body)
			waitAuthorityBlock(t, tx, outer)
			cancel()
			var stopped bool
			select {
			case <-done:
				stopped = true
			case <-time.After(time.Second):
			}
			require.NoError(t, tx.Rollback())
			finishInviteRequest(t, done)
			require.True(t, stopped, "cancelled request must stop its actual SQL statement")
			require.GreaterOrEqual(t, resp.Code, 400)
			var count int
			require.NoError(t, testDB.QueryRow(`SELECT count(*) FROM shop_invite_codes WHERE shop_id=$1 AND is_active`, shop).Scan(&count))
			require.Equal(t, 1, count)
			require.NoError(t, testDB.QueryRow(`SELECT count(*) FROM shop_members WHERE shop_id=$1`, shop).Scan(&count))
			require.Equal(t, 1, count)
		})
	}
}

func TestInviteUnrestrictedDefaultsAndPolicy(t *testing.T) {
	for _, contract := range []string{"", "2"} {
		t.Run("contract="+contract, func(t *testing.T) {
			r, shop, id, code := inviteFixture(t)
			authorizationMember(t, shop, "claimant", "member")
			headers := map[string]string{shared.ContractHeader: contract}
			for _, body := range []map[string]any{{"shop_id": shop}, {"shop_id": shop, "max_uses": nil, "expires_at": nil}} {
				resp := doJSONRequestWithHeaders(t, r, "POST", "/api/v1/auth/shops/invite-codes", body, "claimant", headers)
				require.Equal(t, 201, resp.Code, resp.Body.String())
				data := decodeMap(t, decodeStandardResponse(t, resp.Body).Data)
				require.Regexp(t, `^[0-9A-F]{8}$`, data["code"])
				require.Equal(t, true, data["is_active"])
			}
			resp := doJSONRequestWithHeaders(t, r, "POST", "/api/v1/auth/shops/invite-codes", map[string]any{"shop_id": shop, "max_uses": 0, "expires_at": ""}, "claimant", headers)
			require.Equal(t, 400, resp.Code)
			require.Contains(t, resp.Body.String(), "invite expiry and max-use controls are not supported")
			resp = doJSONRequestWithHeaders(t, r, "POST", "/api/v1/auth/shops/join", map[string]any{"invite_code": code}, "claimant", headers)
			duplicateStatus := 500
			deniedStatus := 500
			if contract == "2" {
				duplicateStatus, deniedStatus = 409, 403
			}
			require.Equal(t, duplicateStatus, resp.Code, resp.Body.String())
			require.Contains(t, resp.Body.String(), "user is already a member of this shop")
			for _, suffix := range []string{"", "/delete"} {
				resp := doJSONRequestWithHeaders(t, r, "DELETE", "/api/v1/auth/shops/invite-codes/"+id+suffix, nil, "claimant", headers)
				require.Equal(t, deniedStatus, resp.Code, resp.Body.String())
			}
			var count int
			require.NoError(t, testDB.QueryRow(`SELECT count(*) FROM shop_invite_codes WHERE shop_id=$1 AND is_active`, shop).Scan(&count))
			require.Equal(t, 3, count)
		})
	}
}

func TestInviteRevocationCurrentAuthority(t *testing.T) {
	for _, action := range []string{"revoke", "delete"} {
		for _, change := range []string{"remove", "demote"} {
			t.Run(action+"/"+change, func(t *testing.T) {
				r, shop, id, _ := inviteFixture(t)
				authorizationMember(t, shop, "claimant", "admin")
				ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
				defer cancel()
				tx := lockInviteShop(t, ctx, shop)
				query := `DELETE FROM shop_members WHERE shop_id=$1 AND user_id='claimant'`
				if change == "demote" {
					query = `UPDATE shop_members SET role='member' WHERE shop_id=$1 AND user_id='claimant'`
				}
				_, err := tx.ExecContext(ctx, query, shop)
				require.NoError(t, err)
				path := "/api/v1/auth/shops/invite-codes/" + id
				if action == "delete" {
					path += "/delete"
				}
				resp, done := inviteRequest(t, ctx, r, "DELETE", path, "claimant", nil)
				waitAuthorityBlock(t, tx, ctx)
				require.NoError(t, tx.Commit())
				finishInviteRequest(t, done)
				require.Equal(t, 500, resp.Code, resp.Body.String())
				var active bool
				require.NoError(t, testDB.QueryRow(`SELECT is_active FROM shop_invite_codes WHERE id=$1`, id).Scan(&active))
				require.True(t, active)
			})
		}
	}
}

func TestInviteRepositoryDerivesShopAuthority(t *testing.T) {
	r, shop, id, _ := inviteFixture(t)
	other := createShop(t, r, "claimant", "other")
	require.NotEqual(t, shop, other)
	repo := invites.NewRepository(testDB)
	user := &bootstrap.User{UserID: "claimant"}
	require.Error(t, repo.DeactivateInviteCode(context.Background(), user, id))
	require.Error(t, repo.DeleteInviteCode(context.Background(), user, id))
	_, err := repo.CreateInviteCode(context.Background(), user, model.ShopInviteCodes{ID: uuid.NewString(), ShopID: shop, Code: "FOREIGN1", CreatedBy: "owner"})
	require.Error(t, err)
	var count int
	require.NoError(t, testDB.QueryRow(`SELECT count(*) FROM shop_invite_codes WHERE shop_id=$1 AND is_active`, shop).Scan(&count))
	require.Equal(t, 1, count)
}

func TestInviteMemberRepositoryCancellation(t *testing.T) {
	for _, boundary := range []string{"pool", "database"} {
		for _, operation := range []string{"join", "invite-create", "invite-code", "invite-id", "invite-list", "invite-revoke", "invite-delete", "member-admin", "member-exists", "member-list", "member-remove", "member-promote", "member-leave"} {
			t.Run(boundary+"/"+operation, func(t *testing.T) {
				_, shop, id, code := inviteFixture(t)
				repo := members.NewRepository(testDB, nil, nil)
				inviteRepo := invites.NewRepository(testDB)
				user := &bootstrap.User{UserID: "owner"}
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
					_, err := tx.ExecContext(outer, `LOCK TABLE shops, shop_members, shop_invite_codes IN ACCESS EXCLUSIVE MODE`)
					require.NoError(t, err)
				}
				ctx, cancel := context.WithCancel(outer)
				defer cancel()
				done := make(chan error, 1)
				waits := testDB.Stats().WaitCount
				go func() {
					var err error
					switch operation {
					case "join":
						err = repo.JoinViaInvite(ctx, &bootstrap.User{UserID: "claimant"}, code)
					case "invite-create":
						_, err = inviteRepo.CreateInviteCode(ctx, user, model.ShopInviteCodes{ID: uuid.NewString(), ShopID: shop, Code: "CANCEL01", CreatedBy: "owner"})
					case "invite-code":
						_, err = inviteRepo.GetInviteCodeByCode(ctx, code)
					case "invite-id":
						_, err = inviteRepo.GetInviteCodeByID(ctx, id)
					case "invite-list":
						_, err = inviteRepo.GetInviteCodesByShop(ctx, user, shop)
					case "invite-revoke":
						err = inviteRepo.DeactivateInviteCode(ctx, user, id)
					case "invite-delete":
						err = inviteRepo.DeleteInviteCode(ctx, user, id)
					case "member-admin":
						_, err = repo.IsUserShopAdmin(ctx, user, shop)
					case "member-exists":
						_, err = repo.IsUserMemberOfShop(ctx, user, shop)
					case "member-list":
						_, err = repo.GetShopMembers(ctx, user, shop)
					case "member-remove":
						err = repo.RemoveMemberFromShop(ctx, user, shop, "claimant")
					case "member-promote":
						err = repo.UpdateMemberRole(ctx, user, shop, "owner", "member")
					case "member-leave":
						err = repo.LeaveShop(ctx, user, shop)
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
				var count int
				require.NoError(t, testDB.QueryRow(`SELECT count(*) FROM shops WHERE id=$1`, shop).Scan(&count))
				require.Equal(t, 1, count)
				require.NoError(t, testDB.QueryRow(`SELECT count(*) FROM shop_members WHERE shop_id=$1 AND role='admin'`, shop).Scan(&count))
				require.Equal(t, 1, count)
				require.NoError(t, testDB.QueryRow(`SELECT count(*) FROM shop_members WHERE shop_id=$1`, shop).Scan(&count))
				require.Equal(t, 1, count)
				require.NoError(t, testDB.QueryRow(`SELECT count(*) FROM shop_invite_codes WHERE shop_id=$1 AND is_active`, shop).Scan(&count))
				require.Equal(t, 1, count)
			})
		}
	}
}

func TestInviteJoinCancellationAfterResolution(t *testing.T) {
	for _, boundary := range []string{"shop", "insert"} {
		t.Run(boundary, func(t *testing.T) {
			_, shop, _, code := inviteFixture(t)
			outer, stop := context.WithTimeout(context.Background(), 8*time.Second)
			defer stop()
			tx, err := testDB.BeginTx(outer, nil)
			require.NoError(t, err)
			defer tx.Rollback()
			if boundary == "shop" {
				_, err = tx.ExecContext(outer, `SELECT id FROM shops WHERE id=$1 FOR UPDATE`, shop)
			} else {
				_, err = tx.ExecContext(outer, `LOCK TABLE shop_members IN SHARE MODE`)
			}
			require.NoError(t, err)
			ctx, cancel := context.WithCancel(outer)
			defer cancel()
			done := make(chan error, 1)
			go func() {
				done <- members.NewRepository(testDB, nil, nil).JoinViaInvite(ctx, &bootstrap.User{UserID: "claimant"}, code)
			}()
			waitAuthorityBlock(t, tx, outer)
			cancel()
			select {
			case err := <-done:
				require.Error(t, err)
			case <-time.After(time.Second):
				t.Fatal("cancelled admission remained blocked")
			}
			require.NoError(t, tx.Rollback())
			var count int
			require.NoError(t, testDB.QueryRow(`SELECT count(*) FROM shop_members WHERE shop_id=$1 AND user_id='claimant'`, shop).Scan(&count))
			require.Zero(t, count)
		})
	}
}
