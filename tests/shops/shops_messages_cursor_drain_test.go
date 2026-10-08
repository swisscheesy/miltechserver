package shops_test

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/request"
	"miltechserver/api/response"
	"miltechserver/api/shops/messages"
	"miltechserver/api/shops/shared"
	"miltechserver/bootstrap"
	"miltechserver/tests/testutil"
)

func TestLegacyCursorDrainsTiesAndBurst(t *testing.T) {
	for _, tied := range []bool{true, false} {
		for _, direction := range []string{"before_id", "after_id"} {
			t.Run(fmt.Sprintf("ties=%t/%s", tied, direction), func(t *testing.T) {
				clearShopTables(t, testDB)
				ensureUser(t, testDB, "user-1")
				ensureUser(t, testDB, "outsider")
				router := newTestRouter(t)
				shop := createShop(t, router, "user-1", "Drain")
				base := time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
				ids := make([]string, 12)
				for i := range ids {
					ids[i] = fmt.Sprintf("00000000-0000-0000-0000-%012d", i)
					stamp := base
					if !tied {
						stamp = base.Add(time.Duration(i) * time.Second)
					}
					_, err := testDB.Exec(`INSERT INTO shop_messages (id,shop_id,user_id,message,created_at) VALUES ($1,$2,$3,$4,$5)`, ids[i], shop, "user-1", "fixed fixture", stamp)
					require.NoError(t, err)
				}
				// Remove the opposite sentinel so exactly ten rows must be collected.
				anchor, opposite := ids[0], ids[11]
				if direction == "before_id" {
					anchor, opposite = ids[11], ids[0]
				}
				_, err := testDB.Exec(`DELETE FROM shop_messages WHERE id=$1`, opposite)
				require.NoError(t, err)
				seen := map[string]bool{}
				var collected []string
				cursor := anchor
				for page := 0; page < 7; page++ {
					res := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/"+shop+"/messages/paginated?"+direction+"="+cursor+"&limit=2", nil, "user-1")
					require.Equal(t, 200, res.Code, res.Body.String())
					data := decodeMap(t, decodeStandardResponse(t, res.Body).Data)
					require.NotContains(t, data, "pagination")
					rows := data["messages"].([]interface{})
					require.Len(t, rows, 2)
					pageIDs := []string{}
					for _, raw := range rows {
						row := raw.(map[string]interface{})
						require.Len(t, row, 9)
						require.Contains(t, row, "parent_id")
						require.Nil(t, row["parent_id"])
						requireAuthorUsername(t, row, "test-user")
						id := row["id"].(string)
						require.False(t, seen[id], "duplicate %s", id)
						seen[id] = true
						collected = append(collected, id)
						pageIDs = append(pageIDs, id)
					}
					require.Greater(t, pageIDs[0], pageIDs[1], "legacy presentation is descending")
					next, ok := data["next_cursor"].(string)
					if !ok {
						break
					}
					boundary := pageIDs[1]
					if direction == "after_id" {
						boundary = pageIDs[0]
					}
					require.Equal(t, boundary, next)
					cursor = next
				}
				require.Len(t, seen, 10)
				require.ElementsMatch(t, ids[1:11], collected)
				denied := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/"+shop+"/messages/paginated?"+direction+"="+anchor, nil, "outsider")
				require.NotEqual(t, 200, denied.Code)
				_, err = testDB.Exec(`DELETE FROM shop_messages WHERE id=$1`, anchor)
				require.NoError(t, err)
				missing := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/"+shop+"/messages/paginated?"+direction+"="+anchor, nil, "user-1")
				require.Equal(t, 500, missing.Code)
				require.Equal(t, messages.LegacyCursorReloadMessage, decodeStandardResponse(t, missing.Body).Message)
				service := messages.NewService(messages.NewRepository(testDB, nil, nil), shared.NewShopAuthorization(testDB))
				_, deletedLegacyAnchorError := service.GetShopMessagesPaginated(context.Background(), &bootstrap.User{UserID: "user-1"}, shop, request.GetShopMessagesPaginatedRequest{BeforeID: &anchor, Limit: 2})
				require.Error(t, deletedLegacyAnchorError)
				var failure *shared.Failure
				require.ErrorAs(t, deletedLegacyAnchorError, &failure)
				require.Equal(t, "reset_required", failure.Code)
			})
		}
	}
}

type messageReadAuthorization struct{ shared.ShopAuthorization }

func (messageReadAuthorization) IsUserMemberOfShop(context.Context, *bootstrap.User, string) (bool, error) {
	return true, nil
}

func TestLegacyMessageServicePhysicalPoolCancellation(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "user-1")
	router := newTestRouter(t)
	shop := createShop(t, router, "user-1", "Cancellation")
	anchor := createMessage(t, router, "user-1", shop, "anchor")
	db, err := testutil.OpenDisposableTestDB(os.Getenv("TEST_DATABASE_URL"), os.Getenv("TEST_DATABASE_MARKER"))
	require.NoError(t, err)
	defer db.Close()
	db.SetMaxOpenConns(1)
	repo := messages.NewRepository(db, nil, nil)
	service := messages.NewService(repo, messageReadAuthorization{})
	anchorRow, err := messages.NewRepository(testDB, nil, nil).GetShopMessageByID(context.Background(), &bootstrap.User{UserID: "user-1"}, anchor)
	require.NoError(t, err)
	laterReads := messages.NewService(&messageLaterReadsRepository{Repository: repo, anchor: anchorRow}, messageReadAuthorization{})
	user := &bootstrap.User{UserID: "user-1"}
	calls := map[string]func(context.Context) error{
		"cursor-query": func(ctx context.Context) error {
			_, e := laterReads.GetShopMessagesPaginated(ctx, user, shop, request.GetShopMessagesPaginatedRequest{AfterID: &anchor, Limit: 2})
			return e
		},
		"count-query": func(ctx context.Context) error {
			_, e := laterReads.GetShopMessagesPaginated(ctx, user, shop, request.GetShopMessagesPaginatedRequest{Page: 1, Limit: 2})
			return e
		},
		"all": func(ctx context.Context) error { _, e := service.GetShopMessages(ctx, user, shop); return e },
		"page": func(ctx context.Context) error {
			_, e := service.GetShopMessagesPaginated(ctx, user, shop, request.GetShopMessagesPaginatedRequest{Page: 1, Limit: 2})
			return e
		},
		"anchor": func(ctx context.Context) error {
			_, e := service.GetShopMessagesPaginated(ctx, user, shop, request.GetShopMessagesPaginatedRequest{AfterID: &anchor, Limit: 2})
			return e
		},
	}
	for name, call := range calls {
		t.Run(name, func(t *testing.T) {
			conn, err := db.Conn(context.Background())
			require.NoError(t, err)
			ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
			defer cancel()
			done := make(chan error, 1)
			go func() { done <- call(ctx) }()
			var result error
			returned := false
			select {
			case result = <-done:
				returned = true
			case <-time.After(250 * time.Millisecond):
			}
			require.NoError(t, conn.Close())
			if !returned {
				select {
				case result = <-done:
				case <-time.After(time.Second):
					t.Fatal("read did not return after pool release")
				}
			}
			require.True(t, returned, "read ignored deadline during physical pool wait")
			require.ErrorIs(t, result, context.DeadlineExceeded)
		})
	}
}

func TestLegacyMessageRepositoryPhysicalCancellation(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "user-1")
	router := newTestRouter(t)
	shop := createShop(t, router, "user-1", "Cancellation")
	anchor := createMessage(t, router, "user-1", shop, "anchor")
	db, err := testutil.OpenDisposableTestDB(os.Getenv("TEST_DATABASE_URL"), os.Getenv("TEST_DATABASE_MARKER"))
	require.NoError(t, err)
	defer db.Close()
	repo := messages.NewRepository(db, nil, nil)
	user := &bootstrap.User{UserID: "user-1"}
	stamp := time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
	calls := map[string]func(context.Context) error{
		"all": func(ctx context.Context) error { _, e := repo.GetShopMessages(ctx, user, shop); return e },
		"page": func(ctx context.Context) error {
			_, e := repo.GetShopMessagesPaginated(ctx, user, shop, 0, 2)
			return e
		},
		"cursor": func(ctx context.Context) error {
			_, e := repo.GetShopMessagesByCursor(ctx, user, shop, anchor, stamp, true, 2)
			return e
		},
		"count": func(ctx context.Context) error { _, e := repo.GetShopMessagesCount(ctx, user, shop); return e },
		"byID":  func(ctx context.Context) error { _, e := repo.GetShopMessageByID(ctx, user, anchor); return e },
	}
	for _, phase := range []string{"pool", "query"} {
		for name, call := range calls {
			t.Run(phase+"/"+name, func(t *testing.T) {
				release := func() {}
				if phase == "pool" {
					db.SetMaxOpenConns(1)
					conn, e := db.Conn(context.Background())
					require.NoError(t, e)
					release = func() { require.NoError(t, conn.Close()) }
				} else {
					tx, e := testDB.BeginTx(context.Background(), nil)
					require.NoError(t, e)
					_, e = tx.Exec(`LOCK TABLE shop_messages IN ACCESS EXCLUSIVE MODE`)
					require.NoError(t, e)
					release = func() { require.NoError(t, tx.Rollback()) }
				}
				ctx, cancel := context.WithTimeout(context.Background(), 75*time.Millisecond)
				defer cancel()
				done := make(chan error, 1)
				go func() { done <- call(ctx) }()
				var result error
				returned := false
				select {
				case result = <-done:
					returned = true
				case <-time.After(350 * time.Millisecond):
				}
				release()
				if !returned {
					select {
					case result = <-done:
					case <-time.After(time.Second):
						t.Fatal("read did not finish")
					}
				}
				require.True(t, returned, "request must end before pool/table lock release")
				require.Error(t, result)
				require.ErrorIs(t, ctx.Err(), context.DeadlineExceeded)
				if phase == "pool" {
					require.ErrorIs(t, result, context.DeadlineExceeded)
				}
			})
		}
	}
}

// Only the scheduling seam is substituted: both writes use the real repository,
// Shop lock, asset transaction and database. This pins a limitation, not a clock guarantee.
type delayedMessageRepository struct {
	messages.Repository
	prepared chan model.ShopMessages
	release  chan struct{}
}

func (r *delayedMessageRepository) CreateShopMessage(ctx context.Context, user *bootstrap.User, message model.ShopMessages) (*response.ShopMessageResponse, error) {
	r.prepared <- message
	select {
	case <-r.release:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	return r.Repository.CreateShopMessage(ctx, user, message)
}
func TestLegacyCursorCurrentWriterCommitOrderLimitation(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "user-1")
	router := newTestRouter(t)
	shop := createShop(t, router, "user-1", "Delayed creation")
	repo := messages.NewRepository(testDB, nil, nil)
	delayed := &delayedMessageRepository{Repository: repo, prepared: make(chan model.ShopMessages, 1), release: make(chan struct{})}
	service := messages.NewService(delayed, messageReadAuthorization{})
	user := &bootstrap.User{UserID: "user-1"}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() {
		_, e := service.CreateShopMessage(ctx, user, model.ShopMessages{ShopID: shop, Message: "delayed current writer"})
		done <- e
	}()
	var prepared model.ShopMessages
	select {
	case prepared = <-delayed.prepared:
	case <-ctx.Done():
		t.Fatal("writer did not prepare")
	}
	newer, err := messages.NewService(repo, messageReadAuthorization{}).CreateShopMessage(ctx, user, model.ShopMessages{ShopID: shop, Message: "committed anchor"})
	require.NoError(t, err)
	// Observe the committed newer anchor before allowing the older prepared write.
	current, err := service.GetShopMessages(ctx, user, shop)
	require.NoError(t, err)
	require.Len(t, current, 1)
	require.Equal(t, newer.ID, current[0].ID)
	close(delayed.release)
	require.NoError(t, <-done)
	late, err := repo.GetShopMessageByID(ctx, user, prepared.ID)
	require.NoError(t, err)
	require.True(t, late.CreatedAt.Before(*newer.CreatedAt))
	page, err := service.GetShopMessagesPaginated(ctx, user, shop, request.GetShopMessagesPaginatedRequest{AfterID: &newer.ID, Limit: 2})
	require.NoError(t, err)
	require.Empty(t, page.Messages, "legacy timestamp cursors cannot see an older timestamp committed after the observed anchor")
	all, err := service.GetShopMessages(ctx, user, shop)
	require.NoError(t, err)
	require.Len(t, all, 2, "the late row exists; only tuple catch-up omits it")
}

func TestLegacyCursorAnchorScopeAndExactText(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "user-1")
	router := newTestRouter(t)
	shop := createShop(t, router, "user-1", "Anchor scope")
	otherShop := createShop(t, router, "user-1", "Other scope")
	anchor := "ABCDEF00-0000-0000-0000-000000000000"
	stamp := time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
	_, err := testDB.Exec(`INSERT INTO shop_messages(id,shop_id,user_id,message,created_at) VALUES($1,$2,$3,$4,$5)`, anchor, shop, "user-1", "text identity", stamp)
	require.NoError(t, err)
	for _, tc := range []struct {
		name, shop, id string
		status         int
		message        string
	}{
		{"exact", shop, anchor, 200, ""},
		{"case is distinct", shop, strings.ToLower(anchor), 500, messages.LegacyCursorReloadMessage},
		{"foreign anchor", otherShop, anchor, 500, "cursor message does not belong to this shop"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			res := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/"+tc.shop+"/messages/paginated?after_id="+tc.id+"&limit=2", nil, "user-1")
			require.Equal(t, tc.status, res.Code, res.Body.String())
			decoded := decodeStandardResponse(t, res.Body)
			require.Equal(t, tc.message, decoded.Message)
			if tc.status == 200 {
				data := decodeMap(t, decoded.Data)
				require.Empty(t, data["messages"])
			}
		})
	}
}

type messageLaterReadsRepository struct {
	messages.Repository
	anchor *response.ShopMessageResponse
}

func (r *messageLaterReadsRepository) GetShopMessageByID(context.Context, *bootstrap.User, string) (*response.ShopMessageResponse, error) {
	return r.anchor, nil
}
func (r *messageLaterReadsRepository) GetShopMessagesPaginated(context.Context, *bootstrap.User, string, int, int) ([]response.ShopMessageResponse, error) {
	return []response.ShopMessageResponse{*r.anchor}, nil
}
