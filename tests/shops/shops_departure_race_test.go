package shops_test

import (
	"context"
	"encoding/json"
	"fmt"
	"miltechserver/api/shops/core"
	"miltechserver/api/shops/members"
	"miltechserver/bootstrap"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func departureFixture(t *testing.T, role string) (*gin.Engine, string) {
	t.Helper()
	r, shop, _, _ := inviteFixture(t)
	_, err := testDB.Exec(`INSERT INTO shop_members(id,shop_id,user_id,role) VALUES ($1,$2,'claimant',$3)`, shop+"_claimant", shop, role)
	require.NoError(t, err)
	return r, shop
}

func assertDepartureState(t *testing.T, shop string, shops, members, admins int) {
	t.Helper()
	var gotShops, gotMembers, gotAdmins, memberless int
	require.NoError(t, testDB.QueryRow(`SELECT count(*) FROM shops WHERE id=$1`, shop).Scan(&gotShops))
	require.NoError(t, testDB.QueryRow(`SELECT count(*),count(*) FILTER (WHERE role='admin') FROM shop_members WHERE shop_id=$1`, shop).Scan(&gotMembers, &gotAdmins))
	require.NoError(t, testDB.QueryRow(`SELECT count(*) FROM shops s WHERE NOT EXISTS (SELECT 1 FROM shop_members m WHERE m.shop_id=s.id)`).Scan(&memberless))
	require.Zero(t, memberless, "departures must not leave a surviving memberless Shop")
	require.Equal(t, shops, gotShops)
	require.Equal(t, members, gotMembers)
	require.Equal(t, admins, gotAdmins)
}

func TestConcurrentFinalDepartures(t *testing.T) {
	for _, scenario := range []string{"leave-leave", "leave-remove", "remove-leave"} {
		t.Run(scenario, func(t *testing.T) {
			r, shop := departureFixture(t, "admin")
			ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
			defer cancel()
			tx := lockInviteShop(t, ctx, shop)
			remove := func() (*httptest.ResponseRecorder, <-chan struct{}) {
				return inviteRequest(t, ctx, r, "DELETE", "/api/v1/auth/shops/members/remove", "claimant", map[string]any{"shop_id": shop, "target_user_id": "owner"})
			}
			leave := func(user string) (*httptest.ResponseRecorder, <-chan struct{}) {
				return inviteRequest(t, ctx, r, "DELETE", "/api/v1/auth/shops/"+shop+"/leave", user, nil)
			}
			var first, second *httptest.ResponseRecorder
			var firstDone, secondDone <-chan struct{}
			if scenario == "remove-leave" {
				first, firstDone = remove()
			} else {
				first, firstDone = leave("owner")
			}
			waitAuthorityBlock(t, tx, ctx)
			if scenario == "leave-leave" {
				second, secondDone = leave("claimant")
			} else if scenario == "leave-remove" {
				second, secondDone = remove()
			} else {
				second, secondDone = leave("owner")
			}
			require.Eventually(t, func() bool {
				var count int
				_, clearErr := tx.ExecContext(ctx, `SELECT pg_stat_clear_snapshot()`)
				if clearErr != nil {
					return false
				}
				err := tx.QueryRowContext(ctx, `SELECT count(*) FROM pg_stat_activity WHERE datname=current_database() AND wait_event_type='Lock'`).Scan(&count)
				return err == nil && count >= 2
			}, time.Second, time.Millisecond)
			require.NoError(t, tx.Commit())
			finishInviteRequest(t, firstDone)
			finishInviteRequest(t, secondDone)
			require.Equal(t, 200, first.Code, first.Body.String())
			if scenario == "leave-leave" {
				require.Equal(t, 200, second.Code, second.Body.String())
				assertDepartureState(t, shop, 0, 0, 0)
			} else {
				require.GreaterOrEqual(t, second.Code, 400, second.Body.String())
				assertDepartureState(t, shop, 1, 1, 1)
				resp := doJSONRequest(t, r, "DELETE", "/api/v1/auth/shops/"+shop+"/leave", nil, "claimant")
				require.Equal(t, 200, resp.Code, resp.Body.String())
				assertDepartureState(t, shop, 0, 0, 0)
			}
		})
	}
}

func TestLastAdminRequiresSuccessor(t *testing.T) {
	r, shop := departureFixture(t, "member")
	resp := doJSONRequest(t, r, "DELETE", "/api/v1/auth/shops/"+shop+"/leave", nil, "owner")
	require.GreaterOrEqual(t, resp.Code, 400, resp.Body.String())
	assertDepartureState(t, shop, 1, 2, 1)
	promote := doJSONRequest(t, r, "PUT", "/api/v1/auth/shops/members/promote", map[string]any{"shop_id": shop, "target_user_id": "claimant"}, "owner")
	require.Equal(t, 200, promote.Code, promote.Body.String())
	resp = doJSONRequest(t, r, "DELETE", "/api/v1/auth/shops/"+shop+"/leave", nil, "owner")
	require.Equal(t, 200, resp.Code, resp.Body.String())
	assertDepartureState(t, shop, 1, 1, 1)
	resp = doJSONRequest(t, r, "DELETE", "/api/v1/auth/shops/"+shop+"/leave", nil, "claimant")
	require.Equal(t, 200, resp.Code, resp.Body.String())
	assertDepartureState(t, shop, 0, 0, 0)
}

func TestNoncreatorFinalMemberMayLeave(t *testing.T) {
	for _, role := range []string{"admin", "member"} {
		t.Run(role, func(t *testing.T) {
			r, shop := departureFixture(t, role)
			_, err := testDB.Exec(`DELETE FROM shop_members WHERE shop_id=$1 AND user_id='owner'`, shop)
			require.NoError(t, err)
			resp := doJSONRequest(t, r, "DELETE", "/api/v1/auth/shops/"+shop+"/leave", nil, "claimant")
			require.Equal(t, 200, resp.Code, resp.Body.String())
			assertDepartureState(t, shop, 0, 0, 0)
		})
	}
}

func TestExplicitDeleteCreatorPolicy(t *testing.T) {
	for _, scenario := range []string{"creator-admin", "creator-member", "noncreator-admin", "removed-creator"} {
		t.Run(scenario, func(t *testing.T) {
			r, shop := departureFixture(t, "admin")
			actor := "owner"
			switch scenario {
			case "creator-member":
				_, err := testDB.Exec(`UPDATE shop_members SET role='member' WHERE shop_id=$1 AND user_id='owner'`, shop)
				require.NoError(t, err)
			case "noncreator-admin":
				actor = "claimant"
			case "removed-creator":
				_, err := testDB.Exec(`DELETE FROM shop_members WHERE shop_id=$1 AND user_id='owner'`, shop)
				require.NoError(t, err)
			}
			resp := doJSONRequest(t, r, "DELETE", "/api/v1/auth/shops/"+shop, nil, actor)
			if scenario == "creator-admin" || scenario == "creator-member" {
				require.Equal(t, 200, resp.Code, resp.Body.String())
				assertDepartureState(t, shop, 0, 0, 0)
			} else {
				require.GreaterOrEqual(t, resp.Code, 400, resp.Body.String())
				count := 2
				if scenario == "removed-creator" {
					count = 1
				}
				assertDepartureState(t, shop, 1, count, count)
			}
		})
	}
}

func TestLastAdminFailureEnvelope(t *testing.T) {
	for _, contract := range []string{"", "2"} {
		t.Run(fmt.Sprintf("contract-%s", contract), func(t *testing.T) {
			r, shop := departureFixture(t, "member")
			req := httptest.NewRequest("DELETE", "/api/v1/auth/shops/"+shop+"/leave", nil)
			req.Header.Set("X-User-ID", "owner")
			req.Header.Set("X-MilTech-Shops-Contract", contract)
			resp := httptest.NewRecorder()
			r.ServeHTTP(resp, req)
			var body map[string]any
			require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
			require.Contains(t, body, "data")
			require.Nil(t, body["data"])
			if contract == "2" {
				require.Equal(t, 409, resp.Code)
				require.Equal(t, "conflict", body["code"])
			} else {
				require.Equal(t, 500, resp.Code)
				require.NotContains(t, body, "code")
			}
			assertDepartureState(t, shop, 1, 2, 1)
		})
	}
}

func TestLastAdminRechecksSuccessor(t *testing.T) {
	r, shop := departureFixture(t, "admin")
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	tx := lockInviteShop(t, ctx, shop)
	resp, done := inviteRequest(t, ctx, r, "DELETE", "/api/v1/auth/shops/"+shop+"/leave", "owner", nil)
	waitAuthorityBlock(t, tx, ctx)
	_, err := tx.ExecContext(ctx, `UPDATE shop_members SET role='member' WHERE shop_id=$1 AND user_id='claimant'`, shop)
	require.NoError(t, err)
	require.NoError(t, tx.Commit())
	finishInviteRequest(t, done)
	require.GreaterOrEqual(t, resp.Code, 400, resp.Body.String())
	assertDepartureState(t, shop, 1, 2, 1)
}

func TestExplicitDeleteCreatorRechecksMembership(t *testing.T) {
	r, shop := departureFixture(t, "admin")
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	tx := lockInviteShop(t, ctx, shop)
	resp, done := inviteRequest(t, ctx, r, "DELETE", "/api/v1/auth/shops/"+shop, "owner", nil)
	waitAuthorityBlock(t, tx, ctx)
	_, err := tx.ExecContext(ctx, `DELETE FROM shop_members WHERE shop_id=$1 AND user_id='owner'`, shop)
	require.NoError(t, err)
	require.NoError(t, tx.Commit())
	finishInviteRequest(t, done)
	require.GreaterOrEqual(t, resp.Code, 400, resp.Body.String())
	assertDepartureState(t, shop, 1, 1, 1)
}

func TestDepartureCleanupCancellation(t *testing.T) {
	for _, operation := range []string{"leave", "delete"} {
		t.Run(operation, func(t *testing.T) {
			_, shop, _, _ := inviteFixture(t)
			outer, stop := context.WithTimeout(context.Background(), 8*time.Second)
			defer stop()
			tx, err := testDB.BeginTx(outer, nil)
			require.NoError(t, err)
			defer tx.Rollback()
			// Block the aggregate cascade after the request has acquired the Shop/member locks.
			_, err = tx.ExecContext(outer, `LOCK TABLE shop_invite_codes IN SHARE MODE`)
			require.NoError(t, err)
			ctx, cancel := context.WithCancel(outer)
			defer cancel()
			done := make(chan error, 1)
			run := func(ctx context.Context) error {
				if operation == "leave" {
					return members.NewRepository(testDB, nil, nil).LeaveShop(ctx, &bootstrap.User{UserID: "owner"}, shop)
				}
				return core.NewRepository(testDB, nil, nil).DeleteShop(ctx, &bootstrap.User{UserID: "owner"}, shop)
			}
			go func() { done <- run(ctx) }()
			waitAuthorityBlock(t, tx, outer)
			cancel()
			select {
			case err := <-done:
				require.Error(t, err)
			case <-time.After(time.Second):
				t.Fatal("cleanup ignored cancellation")
			}
			require.NoError(t, tx.Rollback())
			assertDepartureState(t, shop, 1, 1, 1)
			var invites int
			require.NoError(t, testDB.QueryRow(`SELECT count(*) FROM shop_invite_codes WHERE shop_id=$1`, shop).Scan(&invites))
			require.Equal(t, 1, invites)
			require.NoError(t, run(outer))
			assertDepartureState(t, shop, 0, 0, 0)
		})
	}
}

func TestDepartureCleanupRollback(t *testing.T) {
	_, shop, _, _ := inviteFixture(t)
	_, err := testDB.Exec(`CREATE TABLE task11_cleanup_blocker (shop_id text REFERENCES shops(id))`)
	require.NoError(t, err)
	t.Cleanup(func() { _, err := testDB.Exec(`DROP TABLE task11_cleanup_blocker`); require.NoError(t, err) })
	_, err = testDB.Exec(`INSERT INTO task11_cleanup_blocker(shop_id) VALUES ($1)`, shop)
	require.NoError(t, err)
	err = members.NewRepository(testDB, nil, nil).LeaveShop(context.Background(), &bootstrap.User{UserID: "owner"}, shop)
	require.Error(t, err)
	assertDepartureState(t, shop, 1, 1, 1)
	_, err = testDB.Exec(`DELETE FROM task11_cleanup_blocker`)
	require.NoError(t, err)
	require.NoError(t, members.NewRepository(testDB, nil, nil).LeaveShop(context.Background(), &bootstrap.User{UserID: "owner"}, shop))
	assertDepartureState(t, shop, 0, 0, 0)
}

func TestExplicitDeleteFailureEnvelope(t *testing.T) {
	for _, actor := range []string{"claimant", "removed-creator"} {
		for _, contract := range []string{"", "2"} {
			t.Run(actor+"/contract-"+contract, func(t *testing.T) {
				r, shop := departureFixture(t, "admin")
				user := actor
				count := 2
				if actor == "removed-creator" {
					user = "owner"
					count = 1
					_, err := testDB.Exec(`DELETE FROM shop_members WHERE shop_id=$1 AND user_id='owner'`, shop)
					require.NoError(t, err)
				}
				req := httptest.NewRequest("DELETE", "/api/v1/auth/shops/"+shop, nil)
				req.Header.Set("X-User-ID", user)
				req.Header.Set("X-MilTech-Shops-Contract", contract)
				resp := httptest.NewRecorder()
				r.ServeHTTP(resp, req)
				var body map[string]any
				require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
				require.Contains(t, body, "data")
				require.Nil(t, body["data"])
				if contract == "2" {
					require.Equal(t, 403, resp.Code)
					require.Equal(t, "denied", body["code"])
				} else {
					require.Equal(t, 500, resp.Code)
					require.NotContains(t, body, "code")
				}
				assertDepartureState(t, shop, 1, count, count)
			})
		}
	}
}
