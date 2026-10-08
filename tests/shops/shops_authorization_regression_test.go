package shops_test

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"miltechserver/api/shops/shared"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestFormerCreatorCannotDeleteList(t *testing.T) {
	if shared.CanDeleteList(false, false, false, true) {
		t.Fatal("authorship must not outlive membership")
	}
}

func authorizationMember(t *testing.T, shopID, userID, role string) {
	t.Helper()
	ensureUser(t, testDB, userID)
	_, err := testDB.Exec(`INSERT INTO shop_members (id,shop_id,user_id,role,joined_at) VALUES ($1,$2,$3,$4,NOW())`, uuid.NewString(), shopID, userID, role)
	require.NoError(t, err)
}

func TestListAuthorizationCurrentMembers(t *testing.T) {
	for _, only := range []bool{false, true} {
		t.Run(fmt.Sprint(only), func(t *testing.T) {
			clearShopTables(t, testDB)
			ensureUser(t, testDB, "admin")
			r := newTestRouter(t)
			shop := createShop(t, r, "admin", "authorization")
			authorizationMember(t, shop, "member", "member")
			authorizationMember(t, shop, "other", "member")
			own := createList(t, r, "member", shop)
			another := createList(t, r, "other", shop)
			item := createListItem(t, r, "member", own, "123", "item")
			setting := doJSONRequest(t, r, http.MethodPut, "/api/v1/auth/shops/"+shop+"/settings", map[string]interface{}{"admin_only_lists": only}, "admin")
			require.Equal(t, 200, setting.Code)
			for _, action := range []struct {
				method, path string
				body         interface{}
			}{
				{"POST", "/api/v1/auth/shops/lists", map[string]interface{}{"shop_id": shop, "description": "new"}},
				{"PUT", "/api/v1/auth/shops/lists", map[string]interface{}{"list_id": another, "description": "renamed"}},
				{"PUT", "/api/v1/auth/shops/lists/items", map[string]interface{}{"item_id": item, "niin": "123", "nomenclature": "changed", "quantity": 2}},
				{"POST", "/api/v1/auth/shops/lists/items", map[string]interface{}{"list_id": another, "niin": "123", "nomenclature": "added", "quantity": 1}},
				{"DELETE", "/api/v1/auth/shops/lists/items", map[string]interface{}{"item_id": item}},
			} {
				resp := doJSONRequest(t, r, action.method, action.path, action.body, "member")
				if only {
					require.GreaterOrEqual(t, resp.Code, 400, resp.Body.String())
				} else {
					require.Less(t, resp.Code, 300, resp.Body.String())
					require.NotEmpty(t, decodeStandardResponse(t, resp.Body).Data)
				}
			}
			denied := doJSONRequest(t, r, "DELETE", "/api/v1/auth/shops/lists", map[string]interface{}{"list_id": another}, "member")
			require.GreaterOrEqual(t, denied.Code, 400)
			ownResp := doJSONRequest(t, r, "DELETE", "/api/v1/auth/shops/lists", map[string]interface{}{"list_id": own}, "member")
			if only {
				require.GreaterOrEqual(t, ownResp.Code, 400)
			} else {
				require.Equal(t, 200, ownResp.Code)
			}
			adminResp := doJSONRequest(t, r, "DELETE", "/api/v1/auth/shops/lists", map[string]interface{}{"list_id": another}, "admin")
			require.Equal(t, 200, adminResp.Code)
		})
	}
}

func TestBatchAuthorizationEveryResource(t *testing.T) {
	for _, scenario := range []string{"same-shop", "mixed-shop", "missing", "foreign", "admin-only"} {
		t.Run(scenario, func(t *testing.T) {
			clearShopTables(t, testDB)
			ensureUser(t, testDB, "admin")
			r := newTestRouter(t)
			shop := createShop(t, r, "admin", "first")
			otherShop := createShop(t, r, "admin", "second")
			authorizationMember(t, shop, "member", "member")
			first := createList(t, r, "admin", shop)
			second := createList(t, r, "admin", shop)
			foreign := createList(t, r, "admin", otherShop)
			target := second
			if scenario == "mixed-shop" || scenario == "foreign" {
				target = foreign
			}
			if scenario == "missing" {
				target = uuid.NewString()
			}
			if scenario == "admin-only" {
				_, err := testDB.Exec(`UPDATE shops SET admin_only_lists=true WHERE id=$1`, shop)
				require.NoError(t, err)
			}
			if scenario == "mixed-shop" {
				authorizationMember(t, otherShop, "member", "member")
			}
			a := createListItem(t, r, "admin", first, "1", "first")
			b := createListItem(t, r, "admin", second, "2", "second")
			if target == foreign {
				b = createListItem(t, r, "admin", foreign, "3", "foreign")
			}
			if scenario == "missing" {
				b = uuid.NewString()
			}
			before := 0
			require.NoError(t, testDB.QueryRow(`SELECT count(*) FROM shop_list_items`).Scan(&before))
			add := doJSONRequest(t, r, "POST", "/api/v1/auth/shops/lists/items/bulk", map[string]interface{}{"list_id": first, "items": []map[string]interface{}{
				{"list_id": first, "niin": "4", "nomenclature": "batch-a", "quantity": 1},
				{"list_id": target, "niin": "5", "nomenclature": "batch-b", "quantity": 1},
			}}, "member")
			if scenario == "same-shop" {
				require.Equal(t, 201, add.Code, add.Body.String())
				require.Len(t, decodeSlice(t, decodeStandardResponse(t, add.Body).Data), 2)
			} else {
				require.GreaterOrEqual(t, add.Code, 400)
				if scenario == "missing" {
					body := decodeStandardResponse(t, add.Body)
					require.Equal(t, 500, add.Code)
					require.Equal(t, 500, body.Status)
					require.Equal(t, "null", string(body.Data))
					require.Contains(t, body.Message, "list not found")
					require.NotContains(t, body.Message, "sql:")
					require.NotContains(t, body.Message, "pq:")
				}
			}
			after := 0
			require.NoError(t, testDB.QueryRow(`SELECT count(*) FROM shop_list_items`).Scan(&after))
			if scenario == "same-shop" {
				require.Equal(t, before+2, after)
			} else {
				require.Equal(t, before, after)
			}
			remove := doJSONRequest(t, r, "DELETE", "/api/v1/auth/shops/lists/items/bulk", map[string]interface{}{"item_ids": []string{a, b}}, "member")
			if scenario == "same-shop" {
				require.Equal(t, 200, remove.Code)
				require.Contains(t, remove.Body.String(), `"count":2`)
			} else if scenario == "missing" {
				assertBatchRemovalCount(t, remove, 1)
				require.Zero(t, atomicRowCount(t, "shop_list_items", "id=$1", a))
			} else {
				require.GreaterOrEqual(t, remove.Code, 400)
			}
			count := 0
			require.NoError(t, testDB.QueryRow(`SELECT count(*) FROM shop_list_items`).Scan(&count))
			if scenario == "same-shop" {
				require.Equal(t, after-2, count)
			} else if scenario == "missing" {
				require.Equal(t, after-1, count)
			} else {
				require.Equal(t, after, count)
			}
		})
	}
}

func TestFormerMemberAuthorizationMutations(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "admin")
	r := newTestRouter(t)
	shop := createShop(t, r, "admin", "former")
	authorizationMember(t, shop, "former", "member")
	list := createList(t, r, "former", shop)
	item := createListItem(t, r, "former", list, "1", "original")
	message := createMessage(t, r, "former", shop, "original")
	vehicle := createVehicle(t, r, "former", shop)
	var auditsBefore int
	require.NoError(t, testDB.QueryRow(`SELECT count(*) FROM shop_vehicle_notification_changes`).Scan(&auditsBefore))
	_, err := testDB.Exec(`DELETE FROM shop_members WHERE shop_id=$1 AND user_id=$2`, shop, "former")
	require.NoError(t, err)
	for _, action := range []struct {
		method, path string
		body         interface{}
	}{
		{"PUT", "/api/v1/auth/shops/lists", map[string]interface{}{"list_id": list, "description": "changed"}},
		{"DELETE", "/api/v1/auth/shops/lists", map[string]interface{}{"list_id": list}},
		{"DELETE", "/api/v1/auth/shops/lists/items", map[string]interface{}{"item_id": item}},
		{"PUT", "/api/v1/auth/shops/messages", map[string]interface{}{"message_id": message, "message": "changed"}},
		{"DELETE", "/api/v1/auth/shops/messages/" + message, nil},
		{"DELETE", "/api/v1/auth/shops/vehicles/" + vehicle, nil},
	} {
		resp := doJSONRequest(t, r, action.method, action.path, action.body, "former")
		require.GreaterOrEqual(t, resp.Code, 400, resp.Body.String())
	}
	var text string
	require.NoError(t, testDB.QueryRow(`SELECT message FROM shop_messages WHERE id=$1`, message).Scan(&text))
	require.Equal(t, "original", text)
	var count int
	require.NoError(t, testDB.QueryRow(`SELECT count(*) FROM shop_vehicle WHERE id=$1`, vehicle).Scan(&count))
	require.Equal(t, 1, count)
	var auditsAfter int
	require.NoError(t, testDB.QueryRow(`SELECT count(*) FROM shop_vehicle_notification_changes`).Scan(&auditsAfter))
	require.Equal(t, auditsBefore, auditsAfter)
}

func TestMessageAuthorizationAdminDeleteAuthorEdit(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "admin")
	r := newTestRouter(t)
	shop := createShop(t, r, "admin", "messages")
	authorizationMember(t, shop, "author", "member")
	message := createMessage(t, r, "author", shop, "original")
	edit := map[string]interface{}{"message_id": message, "message": "changed"}
	denied := doJSONRequest(t, r, "PUT", "/api/v1/auth/shops/messages", edit, "admin")
	require.GreaterOrEqual(t, denied.Code, 400)
	allowed := doJSONRequest(t, r, "PUT", "/api/v1/auth/shops/messages", edit, "author")
	require.Equal(t, 200, allowed.Code)
	deleted := doJSONRequest(t, r, "DELETE", "/api/v1/auth/shops/messages/"+message, nil, "admin")
	require.Equal(t, 200, deleted.Code)
}

func TestAuthorizationMembershipLockSerializesRemoval(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "admin")
	r := newTestRouter(t)
	shop := createShop(t, r, "admin", "race")
	authorizationMember(t, shop, "member", "member")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	first, err := testDB.Conn(ctx)
	require.NoError(t, err)
	defer first.Close()
	second, err := testDB.Conn(ctx)
	require.NoError(t, err)
	defer second.Close()
	tx, err := first.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer tx.Rollback()
	_, err = shared.RequireShopMember(ctx, tx, shop, "member")
	require.NoError(t, err)
	var pid int
	require.NoError(t, second.QueryRowContext(ctx, `SELECT pg_backend_pid()`).Scan(&pid))
	done := make(chan error, 1)
	go func() {
		_, err := second.ExecContext(ctx, `DELETE FROM shop_members WHERE shop_id=$1 AND user_id=$2`, shop, "member")
		done <- err
	}()
	require.Eventually(t, func() bool {
		var blocked bool
		err := first.QueryRowContext(ctx, `SELECT cardinality(pg_blocking_pids($1)) > 0`, pid).Scan(&blocked)
		return err == nil && blocked
	}, time.Second, 10*time.Millisecond)
	require.NoError(t, tx.Commit())
	require.NoError(t, <-done)
}

func TestAuthorizationRoleRemovalRacingWrite(t *testing.T) {
	for _, remove := range []bool{false, true} {
		t.Run(fmt.Sprint(remove), func(t *testing.T) {
			clearShopTables(t, testDB)
			ensureUser(t, testDB, "admin")
			r := newTestRouter(t)
			shop := createShop(t, r, "admin", "role race")
			authorizationMember(t, shop, "writer", "admin")
			list := createList(t, r, "writer", shop)
			_, err := testDB.Exec(`UPDATE shops SET admin_only_lists=true WHERE id=$1`, shop)
			require.NoError(t, err)
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			conn, err := testDB.Conn(ctx)
			require.NoError(t, err)
			defer conn.Close()
			tx, err := conn.BeginTx(ctx, nil)
			require.NoError(t, err)
			defer tx.Rollback()
			if remove {
				_, err = tx.ExecContext(ctx, `DELETE FROM shop_members WHERE shop_id=$1 AND user_id=$2`, shop, "writer")
			} else {
				_, err = tx.ExecContext(ctx, `UPDATE shop_members SET role='member' WHERE shop_id=$1 AND user_id=$2`, shop, "writer")
			}
			require.NoError(t, err)
			req, err := http.NewRequestWithContext(ctx, "PUT", "/api/v1/auth/shops/lists", strings.NewReader(fmt.Sprintf(`{"list_id":%q,"description":"must not persist"}`, list)))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-User-ID", "writer")
			resp := httptest.NewRecorder()
			done := make(chan struct{})
			go func() { defer close(done); r.ServeHTTP(resp, req) }()
			// PostgreSQL's blocker list is the barrier: the request has reached its
			// transactional authorization and is waiting for the role change to finish.
			require.Eventually(t, func() bool {
				var blocked bool
				err := tx.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM pg_stat_activity WHERE pg_backend_pid() = ANY(pg_blocking_pids(pid)))`).Scan(&blocked)
				return err == nil && blocked
			}, 2*time.Second, 10*time.Millisecond)
			require.NoError(t, tx.Commit())
			select {
			case <-done:
			case <-ctx.Done():
				t.Fatal("write did not finish after role lock released")
			}
			require.GreaterOrEqual(t, resp.Code, 400, resp.Body.String())
			var description string
			require.NoError(t, testDB.QueryRow(`SELECT description FROM shop_lists WHERE id=$1`, list).Scan(&description))
			require.Equal(t, "Test list", description)
		})
	}
}

func TestAuthorizationForeignSingleResources(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "admin")
	ensureUser(t, testDB, "outsider")
	r := newTestRouter(t)
	shop := createShop(t, r, "admin", "foreign")
	list := createList(t, r, "admin", shop)
	item := createListItem(t, r, "admin", list, "1", "original")
	for _, action := range []struct {
		method, path string
		body         interface{}
	}{
		{"POST", "/api/v1/auth/shops/lists", map[string]interface{}{"shop_id": shop, "description": "bad"}},
		{"PUT", "/api/v1/auth/shops/lists", map[string]interface{}{"list_id": list, "description": "bad"}},
		{"DELETE", "/api/v1/auth/shops/lists", map[string]interface{}{"list_id": list}},
		{"POST", "/api/v1/auth/shops/lists/items", map[string]interface{}{"list_id": list, "niin": "2", "nomenclature": "bad", "quantity": 1}},
		{"PUT", "/api/v1/auth/shops/lists/items", map[string]interface{}{"item_id": item, "niin": "2", "nomenclature": "bad", "quantity": 2}},
		{"DELETE", "/api/v1/auth/shops/lists/items", map[string]interface{}{"item_id": item}},
	} {
		resp := doJSONRequest(t, r, action.method, action.path, action.body, "outsider")
		require.GreaterOrEqual(t, resp.Code, 400)
	}
	var description, nomenclature string
	require.NoError(t, testDB.QueryRow(`SELECT description FROM shop_lists WHERE id=$1`, list).Scan(&description))
	require.Equal(t, "Test list", description)
	require.NoError(t, testDB.QueryRow(`SELECT nomenclature FROM shop_list_items WHERE id=$1`, item).Scan(&nomenclature))
	require.Equal(t, "original", nomenclature)
}
