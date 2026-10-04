package shops_test

import (
	"context"
	"database/sql"
	"encoding/json"
	listitems "miltechserver/api/shops/lists/items"
	"miltechserver/api/shops/shared"
	notificationitems "miltechserver/api/shops/vehicles/notifications/items"
	"miltechserver/bootstrap"
	"miltechserver/tests/testutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func batchRemovalFixture(t *testing.T, kind string) (*gin.Engine, string, string, string) {
	t.Helper()
	r, shop, vehicle := atomicFixture(t, "atomic-owner")
	parent := createList(t, r, "atomic-owner", shop)
	if kind == "notification" {
		parent = createNotification(t, r, "atomic-owner", shop, vehicle, "Removal")
	}
	return r, shop, parent, addBatchRemovalItem(t, r, kind, parent, "seed")
}
func addBatchRemovalItem(t *testing.T, r *gin.Engine, kind, parent, niin string) string {
	t.Helper()
	if kind == "list" {
		return createListItem(t, r, "atomic-owner", parent, niin, "Seed")
	}
	rec := doJSONRequest(t, r, http.MethodPost, "/api/v1/auth/shops/notifications/items", map[string]any{"notification_id": parent, "niin": niin, "nomenclature": "Seed", "quantity": 7, "nickname": "", "unit_of_measure": " raw-unit "}, "atomic-owner")
	require.Equal(t, 201, rec.Code, rec.Body.String())
	return decodeMap(t, decodeStandardResponse(t, rec.Body).Data)["id"].(string)
}
func batchRemovalPath(kind string) string {
	if kind == "list" {
		return "/api/v1/auth/shops/lists/items/bulk"
	}
	return "/api/v1/auth/shops/notifications/items/bulk"
}
func assertBatchRemovalCount(t *testing.T, rec *httptest.ResponseRecorder, count int64) {
	t.Helper()
	require.Equal(t, 200, rec.Code, rec.Body.String())
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, map[string]any{"message": "Items removed successfully", "count": float64(count)}, body)
}
func TestBatchRemovalActualCountAndStaleIDs(t *testing.T) {
	for _, kind := range []string{"list", "notification"} {
		for _, contract := range []bool{false, true} {
			for _, scenario := range []string{"first-missing", "nonfirst-missing", "all-missing", "duplicates", "duplicate-and-missing"} {
				label := "legacy"
				if contract {
					label = "contract2"
				}
				t.Run(kind+"/"+label+"/"+scenario, func(t *testing.T) {
					r, _, parent, id := batchRemovalFixture(t, kind)
					missing := uuid.NewString()
					ids := []string{id, missing}
					count := int64(1)
					switch scenario {
					case "first-missing":
						ids = []string{missing, id}
					case "all-missing":
						ids = []string{missing, uuid.NewString()}
						count = 0
					case "duplicates":
						ids = []string{id, id}
					case "duplicate-and-missing":
						ids = []string{id, missing, id}
					}
					before := writerSnapshot(t)
					var rec *httptest.ResponseRecorder
					if contract {
						rec = doContractRequest(t, r, http.MethodDelete, batchRemovalPath(kind), map[string]any{"item_ids": ids}, "atomic-owner")
					} else {
						rec = doJSONRequest(t, r, http.MethodDelete, batchRemovalPath(kind), map[string]any{"item_ids": ids}, "atomic-owner")
					}
					assertBatchRemovalCount(t, rec, count)
					if count == 0 {
						require.Equal(t, before, writerSnapshot(t))
						return
					}
					table := "shop_list_items"
					if kind == "notification" {
						table = "shop_notification_items"
					}
					require.Zero(t, atomicRowCount(t, table, "id=$1", id))
					before = writerSnapshot(t)
					assertBatchRemovalCount(t, doJSONRequest(t, r, http.MethodDelete, batchRemovalPath(kind), map[string]any{"item_ids": ids}, "atomic-owner"), 0)
					require.Equal(t, before, writerSnapshot(t))
					if kind == "notification" {
						var nickname, unit string
						require.NoError(t, testDB.QueryRow(`SELECT nickname,unit_of_measure FROM shop_notification_item_metadata WHERE notification_id=$1 AND niin='seed'`, parent).Scan(&nickname, &unit))
						require.Empty(t, nickname)
						require.Equal(t, " raw-unit ", unit)
						require.Equal(t, 1, atomicRowCount(t, "shop_vehicle_notification_changes", "notification_id=$1 AND change_type='items_removed'", parent))
					}
				})
			}
		}
	}
}

func TestBatchRemovalWholeBatchAuthorization(t *testing.T) {
	for _, kind := range []string{"list", "notification"} {
		for _, scenario := range []string{"foreign-survivor", "mixed-shops-authorized-both", "same-shop-parents", "missing-foreign"} {
			t.Run(kind+"/"+scenario, func(t *testing.T) {
				r, shop, parent, id := batchRemovalFixture(t, kind)
				otherShop := shop
				if scenario != "same-shop-parents" {
					otherShop = createShop(t, r, "atomic-owner", "Other")
				}
				otherParent := createList(t, r, "atomic-owner", otherShop)
				if kind == "notification" {
					var vehicle string
					if otherShop == shop {
						require.NoError(t, testDB.QueryRow(`SELECT vehicle_id FROM shop_vehicle_notifications WHERE id=$1`, parent).Scan(&vehicle))
					} else {
						vehicle = createVehicle(t, r, "atomic-owner", otherShop)
					}
					otherParent = createNotification(t, r, "atomic-owner", otherShop, vehicle, "Other")
				}
				otherID := addBatchRemovalItem(t, r, kind, otherParent, "other")
				actor := "atomic-owner"
				if scenario == "foreign-survivor" || scenario == "missing-foreign" {
					ensureUser(t, testDB, "batch-member")
					_, err := testDB.Exec(`INSERT INTO shop_members(id,shop_id,user_id,role) VALUES($1,$2,'batch-member','admin')`, uuid.NewString(), shop)
					require.NoError(t, err)
					actor = "batch-member"
				}
				if scenario == "missing-foreign" {
					rec := doJSONRequest(t, r, http.MethodDelete, batchRemovalPath(kind), map[string]any{"item_ids": []string{otherID}}, "atomic-owner")
					assertBatchRemovalCount(t, rec, 1)
				}
				before := writerSnapshot(t)
				rec := doContractRequest(t, r, http.MethodDelete, batchRemovalPath(kind), map[string]any{"item_ids": []string{id, otherID}}, actor)
				if scenario == "missing-foreign" {
					assertBatchRemovalCount(t, rec, 1)
				} else if scenario == "same-shop-parents" && kind == "list" {
					assertBatchRemovalCount(t, rec, 2)
				} else {
					require.NotEqual(t, 200, rec.Code, rec.Body.String())
					require.Equal(t, before, writerSnapshot(t))
				}

			})
		}
	}
}

func TestBatchRemovalRace(t *testing.T) {
	for _, kind := range []string{"list", "notification"} {
		for _, boundary := range []string{"shop", "parent", "item"} {
			for _, all := range []bool{false, true} {
				name := "one-deleted"
				if all {
					name = "all-deleted"
				}
				t.Run(kind+"/"+boundary+"/"+name, func(t *testing.T) {
					r, shop, parent, id := batchRemovalFixture(t, kind)
					other := addBatchRemovalItem(t, r, kind, parent, "other")
					if other < id {
						id, other = other, id
					}
					table, parentTable := "shop_list_items", "shop_lists"
					if kind == "notification" {
						table, parentTable = "shop_notification_items", "shop_vehicle_notifications"
					}
					ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
					defer cancel()
					tx, err := testDB.BeginTx(ctx, nil)
					require.NoError(t, err)
					defer tx.Rollback()
					lockTable, lockID := table, id
					if boundary == "shop" {
						lockTable, lockID = "shops", shop
					}
					if boundary == "parent" {
						lockTable, lockID = parentTable, parent
					}
					_, err = tx.ExecContext(ctx, `SELECT id FROM `+lockTable+` WHERE id=$1 FOR UPDATE`, lockID)
					require.NoError(t, err)
					done := make(chan *httptest.ResponseRecorder, 1)
					go func() {
						done <- doJSONRequest(t, r, http.MethodDelete, batchRemovalPath(kind), map[string]any{"item_ids": []string{id, other, id}}, "atomic-owner")
					}()
					waitAuthorityBlock(t, tx, ctx)
					_, err = tx.ExecContext(ctx, `DELETE FROM `+table+` WHERE id=$1`, id)
					require.NoError(t, err)
					if all {
						_, err = tx.ExecContext(ctx, `DELETE FROM `+table+` WHERE id=$1`, other)
						require.NoError(t, err)
					}
					require.NoError(t, tx.Commit())
					count := int64(1)
					if all {
						count = 0
					}
					select {
					case rec := <-done:
						assertBatchRemovalCount(t, rec, count)
					case <-ctx.Done():
						t.Fatal("removal did not finish", ctx.Err())
					}
					require.Zero(t, atomicRowCount(t, table, "id=$1 OR id=$2", id, other))
				})
			}
		}
	}
}

func TestBatchRemovalCurrentAuthority(t *testing.T) {
	for _, kind := range []string{"list", "notification"} {
		for _, change := range []string{"removed", "demoted-admin-only", "admin-only-enabled"} {
			t.Run(kind+"/"+change, func(t *testing.T) {
				r, shop, _, id := batchRemovalFixture(t, kind)
				ensureUser(t, testDB, "batch-member")
				role := "admin"
				if change == "admin-only-enabled" {
					role = "member"
				}
				_, err := testDB.Exec(`INSERT INTO shop_members(id,shop_id,user_id,role) VALUES($1,$2,'batch-member',$3)`, uuid.NewString(), shop, role)
				require.NoError(t, err)
				if change == "demoted-admin-only" {
					_, err = testDB.Exec(`UPDATE shops SET admin_only_lists=true WHERE id=$1`, shop)
					require.NoError(t, err)
				}
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				tx, err := testDB.BeginTx(ctx, nil)
				require.NoError(t, err)
				defer tx.Rollback()
				_, err = tx.ExecContext(ctx, `SELECT id FROM shops WHERE id=$1 FOR UPDATE`, shop)
				require.NoError(t, err)
				done := make(chan *httptest.ResponseRecorder, 1)
				go func() {
					done <- doContractRequest(t, r, http.MethodDelete, batchRemovalPath(kind), map[string]any{"item_ids": []string{id}}, "batch-member")
				}()
				waitAuthorityBlock(t, tx, ctx)
				switch change {
				case "removed":
					_, err = tx.ExecContext(ctx, `DELETE FROM shop_members WHERE shop_id=$1 AND user_id='batch-member'`, shop)
				case "demoted-admin-only":
					_, err = tx.ExecContext(ctx, `UPDATE shop_members SET role='member' WHERE shop_id=$1 AND user_id='batch-member'`, shop)
				case "admin-only-enabled":
					_, err = tx.ExecContext(ctx, `UPDATE shops SET admin_only_lists=true WHERE id=$1`, shop)
				}
				require.NoError(t, err)
				before := writerSnapshot(t)
				require.NoError(t, tx.Commit())
				select {
				case rec := <-done:
					if kind == "notification" && change != "removed" {
						assertBatchRemovalCount(t, rec, 1)
					} else {
						require.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())
						require.Equal(t, before, writerSnapshot(t))
					}
				case <-ctx.Done():
					t.Fatal(ctx.Err())
				}
			})
		}
	}
}

func TestBatchRemovalParentDeletedWhileWaiting(t *testing.T) {
	for _, kind := range []string{"list", "notification"} {
		for _, resource := range []string{"parent", "shop"} {
			t.Run(kind+"/"+resource, func(t *testing.T) {
				r, shop, parent, id := batchRemovalFixture(t, kind)
				table := "shop_lists"
				if kind == "notification" {
					table = "shop_vehicle_notifications"
				}
				if resource == "shop" {
					table, parent = "shops", shop
				}
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				tx, err := testDB.BeginTx(ctx, nil)
				require.NoError(t, err)
				defer tx.Rollback()
				_, err = tx.ExecContext(ctx, `SELECT id FROM `+table+` WHERE id=$1 FOR UPDATE`, parent)
				require.NoError(t, err)
				done := make(chan *httptest.ResponseRecorder, 1)
				go func() {
					done <- doJSONRequest(t, r, http.MethodDelete, batchRemovalPath(kind), map[string]any{"item_ids": []string{id}}, "atomic-owner")
				}()
				waitAuthorityBlock(t, tx, ctx)
				_, err = tx.ExecContext(ctx, `DELETE FROM `+table+` WHERE id=$1`, parent)
				require.NoError(t, err)
				require.NoError(t, tx.Commit())
				select {
				case rec := <-done:
					assertBatchRemovalCount(t, rec, 0)
				case <-ctx.Done():
					t.Fatal(ctx.Err())
				}
			})
		}
	}
}

func TestBatchRemovalActorAndCanceledRequest(t *testing.T) {
	for _, kind := range []string{"list", "notification"} {
		t.Run(kind, func(t *testing.T) {
			r, _, _, id := batchRemovalFixture(t, kind)
			var call func(context.Context, *bootstrap.User, []string) (int64, error)
			if kind == "list" {
				call = listitems.NewService(listitems.NewRepository(testDB), nil, nil, nil).RemoveListItemBatch
			} else {
				call = notificationitems.NewService(notificationitems.NewRepository(testDB)).RemoveNotificationItemList
			}
			before := writerSnapshot(t)
			for _, user := range []*bootstrap.User{nil, {}, {UserID: "atomic-owner"}} {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				count, err := call(ctx, user, []string{id})
				require.Zero(t, count)
				require.ErrorIs(t, err, context.Canceled)
				if user == nil || user.UserID == "" {
					count, err = call(context.Background(), user, []string{uuid.NewString()})
					require.Zero(t, count)
					require.Error(t, err)
				}
			}
			ensureUser(t, testDB, "unrelated-actor")
			count, err := call(context.Background(), &bootstrap.User{UserID: "unrelated-actor"}, []string{uuid.NewString()})
			require.NoError(t, err)
			require.Zero(t, count)
			require.Equal(t, before, writerSnapshot(t))
			rec := doJSONRequest(t, r, http.MethodDelete, batchRemovalPath(kind), map[string]any{"item_ids": []string{uuid.NewString()}}, "")
			require.Equal(t, http.StatusUnauthorized, rec.Code)
		})
	}
}

func TestBatchRemovalListAuthorizationLocksAndStrictMutation(t *testing.T) {
	r, shop, parent, id := batchRemovalFixture(t, "list")
	second := addBatchRemovalItem(t, r, "list", parent, "second")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	tx, err := testDB.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer tx.Rollback()
	survivors, err := shared.AuthorizeExistingListItems(ctx, tx, "atomic-owner", []string{id, uuid.NewString(), second, id})
	require.NoError(t, err)
	expected := []string{id, second}
	if second < id {
		expected = []string{second, id}
	}
	require.Equal(t, expected, survivors)
	done := make(chan error, 1)
	go func() {
		other, e := testDB.BeginTx(ctx, nil)
		if e != nil {
			done <- e
			return
		}
		defer other.Rollback()
		_, _, e = shared.LockShopMutation(ctx, other, shop, "atomic-owner")
		if e == nil {
			e = other.Commit()
		}
		done <- e
	}()
	waitAuthorityBlock(t, tx, ctx)
	_, err = tx.ExecContext(ctx, `DELETE FROM shop_list_items WHERE id=$1 OR id=$2`, id, second)
	require.NoError(t, err)
	require.NoError(t, tx.Commit())
	require.NoError(t, <-done)
	tx, err = testDB.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer tx.Rollback()
	require.Error(t, shared.AuthorizeListMutation(ctx, tx, "atomic-owner", "", nil, []string{id}, false))
}

// A malformed persisted ownership must reject before the first metadata write,
// even when every physical item points to the same notification.
func TestBatchRemovalForeignPhysicalOwner(t *testing.T) {
	r, _, parent, id := batchRemovalFixture(t, "notification")
	other := addBatchRemovalItem(t, r, "notification", parent, "foreign")
	_, err := testDB.Exec(`UPDATE shop_notification_items SET shop_id='historical-other' WHERE id=$1`, other)
	require.NoError(t, err)
	before := writerSnapshot(t)
	rec := doContractRequest(t, r, http.MethodDelete, batchRemovalPath("notification"), map[string]any{"item_ids": []string{id, other}}, "atomic-owner")
	require.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())
	require.Equal(t, before, writerSnapshot(t))
}

func TestBatchRemovalPhysicalCancellation(t *testing.T) {
	for _, kind := range []string{"list", "notification"} {
		for _, boundary := range []string{"pool", "shop", "parent", "item"} {
			// Task 18 already covers list repository pool and Shop waits; this adds
			// the newly changed service path and notification transaction boundaries.
			t.Run(kind+"/"+boundary, func(t *testing.T) {
				_, shop, parent, id := batchRemovalFixture(t, kind)
				db, err := testutil.OpenDisposableTestDB(os.Getenv("TEST_DATABASE_URL"), os.Getenv("TEST_DATABASE_MARKER"))
				require.NoError(t, err)
				defer db.Close()
				db.SetMaxOpenConns(1)
				var call func(context.Context, *bootstrap.User, []string) (int64, error)
				if kind == "list" {
					call = listitems.NewService(listitems.NewRepository(db), nil, nil, nil).RemoveListItemBatch
				} else {
					call = notificationitems.NewService(notificationitems.NewRepository(db)).RemoveNotificationItemList
				}
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				var release func()
				var blocker *sql.Tx
				if boundary == "pool" {
					conn, e := db.Conn(ctx)
					require.NoError(t, e)
					release = func() { require.NoError(t, conn.Close()) }
				} else {
					blocker, err = testDB.BeginTx(ctx, nil)
					require.NoError(t, err)
					table, target := "shops", shop
					if boundary == "parent" {
						table, target = "shop_lists", parent
						if kind == "notification" {
							table = "shop_vehicle_notifications"
						}
					}
					if boundary == "item" {
						table, target = "shop_list_items", id
						if kind == "notification" {
							table = "shop_notification_items"
						}
					}
					_, err = blocker.ExecContext(ctx, `SELECT id FROM `+table+` WHERE id=$1 FOR UPDATE`, target)
					require.NoError(t, err)
					release = func() { require.NoError(t, blocker.Rollback()) }
				}
				defer release()
				before := writerSnapshot(t)
				requestCtx, cancelRequest := context.WithCancel(ctx)
				defer cancelRequest()
				type result struct {
					count int64
					err   error
				}
				done := make(chan result, 1)
				go func() {
					count, e := call(requestCtx, &bootstrap.User{UserID: "atomic-owner"}, []string{id})
					done <- result{count, e}
				}()
				if boundary == "pool" {
					require.Eventually(t, func() bool { return db.Stats().WaitCount > 0 }, time.Second, time.Millisecond)
				} else {
					waitAuthorityBlock(t, blocker, ctx)
				}
				cancelRequest()
				select {
				case got := <-done:
					require.Zero(t, got.count)
					require.Error(t, got.err)
				case <-ctx.Done():
					t.Fatal("canceled removal kept waiting", ctx.Err())
				}
				require.Equal(t, before, writerSnapshot(t))
			})
		}
	}
}

func TestBatchRemovalMissingAfterAuthorityChange(t *testing.T) {
	for _, kind := range []string{"list", "notification"} {
		for _, change := range []string{"removed", "admin-only"} {
			t.Run(kind+"/"+change, func(t *testing.T) {
				r, shop, _, id := batchRemovalFixture(t, kind)
				ensureUser(t, testDB, "batch-member")
				_, err := testDB.Exec(`INSERT INTO shop_members(id,shop_id,user_id,role) VALUES($1,$2,'batch-member','member')`, uuid.NewString(), shop)
				require.NoError(t, err)
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				tx, err := testDB.BeginTx(ctx, nil)
				require.NoError(t, err)
				defer tx.Rollback()
				_, err = tx.ExecContext(ctx, `SELECT id FROM shops WHERE id=$1 FOR UPDATE`, shop)
				require.NoError(t, err)
				done := make(chan *httptest.ResponseRecorder, 1)
				go func() {
					done <- doJSONRequest(t, r, http.MethodDelete, batchRemovalPath(kind), map[string]any{"item_ids": []string{id}}, "batch-member")
				}()
				waitAuthorityBlock(t, tx, ctx)
				table := "shop_list_items"
				if kind == "notification" {
					table = "shop_notification_items"
				}
				_, err = tx.ExecContext(ctx, `DELETE FROM `+table+` WHERE id=$1`, id)
				require.NoError(t, err)
				if change == "removed" {
					_, err = tx.ExecContext(ctx, `DELETE FROM shop_members WHERE shop_id=$1 AND user_id='batch-member'`, shop)
				} else {
					_, err = tx.ExecContext(ctx, `UPDATE shops SET admin_only_lists=true WHERE id=$1`, shop)
				}
				require.NoError(t, err)
				require.NoError(t, tx.Commit())
				select {
				case rec := <-done:
					assertBatchRemovalCount(t, rec, 0)
				case <-ctx.Done():
					t.Fatal(ctx.Err())
				}
			})
		}
	}
}
