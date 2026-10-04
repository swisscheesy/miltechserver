package shops_test

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// legacyMessageKeys is the released message JSON field set. Sync rows reuse the
// same response type, so both must stay exactly this set (no insertion_number).
var legacyMessageKeys = []string{"author_username", "created_at", "id", "is_edited", "message", "parent_id", "shop_id", "updated_at", "user_id"}

type messageSyncPage struct {
	Rows        []map[string]interface{} `json:"rows"`
	OlderCursor *string                  `json:"older_cursor"`
	HasOlder    bool                     `json:"has_older"`
	Watermark   string                   `json:"watermark"`
	NextCursor  *string                  `json:"next_cursor"`
	HasMore     bool                     `json:"has_more"`
	NextAfter   string                   `json:"next_after"`
	Through     string                   `json:"through"`
	MissingIDs  []string                 `json:"missing_ids"`
}

type failureBody struct {
	Status int             `json:"status"`
	Code   string          `json:"code"`
	Data   json.RawMessage `json:"data"`
}

func messageSyncPath(shopID, operation string, query url.Values) string {
	path := "/api/v1/auth/shops/" + shopID + "/messages-v2/" + operation
	if len(query) > 0 {
		path += "?" + query.Encode()
	}
	return path
}

// decodeSyncPage requires a 200 and decodes into typed fields, so a numeric
// watermark/next_after (instead of a decimal string) fails the decode.
func decodeSyncPage(t *testing.T, w *httptest.ResponseRecorder) messageSyncPage {
	t.Helper()
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var page messageSyncPage
	require.NoError(t, json.Unmarshal(decodeStandardResponse(t, w.Body).Data, &page), w.Body.String())
	require.NotNil(t, page.Rows, "rows must be an array, never null: %s", w.Body.String())
	for _, row := range page.Rows {
		requireMessageKeys(t, row)
	}
	return page
}

func requireFailure(t *testing.T, w *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	require.Equal(t, status, w.Code, w.Body.String())
	var body failureBody
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body), w.Body.String())
	require.Equal(t, code, body.Code, w.Body.String())
	require.NotContains(t, w.Body.String(), "missing_ids", "a failed read must never report IDs as missing")
	require.NotContains(t, w.Body.String(), "rows", "a failed read must never return rows")
}

func requireMessageKeys(t *testing.T, message map[string]interface{}) {
	t.Helper()
	keys := make([]string, 0, len(message))
	for key := range message {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	require.Equal(t, legacyMessageKeys, keys, "message JSON must not gain insertion_number (jet model must stay unregenerated)")
}

func rowIDs(rows []map[string]interface{}) []string {
	ids := make([]string, len(rows))
	for i, row := range rows {
		ids[i] = row["id"].(string)
	}
	return ids
}

// setMessageCreatedAt pins created_at so ordering assertions never depend on
// the wall clock of successive requests.
func setMessageCreatedAt(t *testing.T, messageID string, createdAt time.Time) {
	t.Helper()
	result, err := testDB.Exec(`UPDATE public.shop_messages SET created_at=$2 WHERE id=$1`, messageID, createdAt)
	require.NoError(t, err)
	affected, err := result.RowsAffected()
	require.NoError(t, err)
	require.Equal(t, int64(1), affected)
}

func insertMessageAt(t *testing.T, shopID, userID, body string, createdAt time.Time) string {
	t.Helper()
	id := uuid.NewString()
	_, err := testDB.Exec(
		`INSERT INTO public.shop_messages (id, shop_id, user_id, message, created_at, updated_at, is_edited)
		 VALUES ($1, $2, $3, $4, $5, $5, false)`,
		id, shopID, userID, body, createdAt)
	require.NoError(t, err)
	return id
}

func addShopMember(t *testing.T, shopID, userID string) {
	t.Helper()
	_, err := testDB.Exec(`INSERT INTO public.shop_members (id, shop_id, user_id, role) VALUES ($1, $2, $3, 'member')`, uuid.NewString(), shopID, userID)
	require.NoError(t, err)
}

func reconcile(t *testing.T, router *gin.Engine, shopID string, ids []string, userID string) *httptest.ResponseRecorder {
	t.Helper()
	return doContractRequest(t, router, http.MethodPost, messageSyncPath(shopID, "reconcile", nil), map[string]interface{}{"ids": ids}, userID)
}

func catchUp(t *testing.T, router *gin.Engine, shopID, after string, through *string, limit int, userID string) messageSyncPage {
	t.Helper()
	query := url.Values{"after": {after}}
	if through != nil {
		query.Set("through", *through)
	}
	if limit > 0 {
		query.Set("limit", fmt.Sprint(limit))
	}
	return decodeSyncPage(t, doContractRequest(t, router, http.MethodGet, messageSyncPath(shopID, "catch-up", query), nil, userID))
}

func legacyDeleteMessage(t *testing.T, router *gin.Engine, messageID, userID string) {
	t.Helper()
	resp := doJSONRequest(t, router, http.MethodDelete, "/api/v1/auth/shops/messages/"+messageID, nil, userID)
	require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
}

func TestMessageSyncRoutesInitialReturnsWatermark(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "user-1")
	router := newTestRouterWithFlags(t, false, true)
	shopID := createShop(t, router, "user-1", "Initial")

	base := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	first := createMessage(t, router, "user-1", shopID, "first")
	second := createMessage(t, router, "user-1", shopID, "second")
	third := createMessage(t, router, "user-1", shopID, "third")
	for i, id := range []string{first, second, third} {
		setMessageCreatedAt(t, id, base.Add(time.Duration(i)*time.Second))
	}

	w := doContractRequest(t, router, http.MethodGet, messageSyncPath(shopID, "initial", url.Values{"limit": {"2"}}), nil, "user-1")
	t.Logf("SAMPLE initial: %s", w.Body.String())
	page := decodeSyncPage(t, w)
	require.Equal(t, []string{third, second}, rowIDs(page.Rows), "newest first")
	require.True(t, page.HasOlder)
	require.NotNil(t, page.OlderCursor)
	require.NotEmpty(t, *page.OlderCursor)
	require.Equal(t, "3", page.Watermark)
	require.Contains(t, w.Body.String(), `"watermark":"3"`, "watermark must be a decimal string")
	require.Equal(t, "test-user", page.Rows[0]["author_username"])

	w = doContractRequest(t, router, http.MethodGet, messageSyncPath(shopID, "history", url.Values{"cursor": {*page.OlderCursor}, "limit": {"2"}}), nil, "user-1")
	t.Logf("SAMPLE history: %s", w.Body.String())
	older := decodeSyncPage(t, w)
	require.Equal(t, []string{first}, rowIDs(older.Rows))
	require.False(t, older.HasMore)
	require.Nil(t, older.NextCursor)
}

// A Shop created after migration 018 has no counter row until its first
// message; its initial page must be an empty snapshot at watermark 0.
func TestMessageSyncRoutesInitialEmptyShopStartsAtZero(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "user-1")
	router := newTestRouterWithFlags(t, false, true)
	shopID := createShop(t, router, "user-1", "Empty")

	page := decodeSyncPage(t, doContractRequest(t, router, http.MethodGet, messageSyncPath(shopID, "initial", nil), nil, "user-1"))
	require.Empty(t, page.Rows)
	require.False(t, page.HasOlder)
	require.Nil(t, page.OlderCursor)
	require.Equal(t, "0", page.Watermark)

	caught := catchUp(t, router, shopID, "0", nil, 0, "user-1")
	require.Empty(t, caught.Rows)
	require.Equal(t, "0", caught.NextAfter)
	require.Equal(t, "0", caught.Through)
	require.False(t, caught.HasMore)

	created := createMessage(t, router, "user-1", shopID, "first ever")
	caught = catchUp(t, router, shopID, "0", nil, 0, "user-1")
	require.Equal(t, []string{created}, rowIDs(caught.Rows))
	require.Equal(t, "1", caught.Through)
}

// A row the migration could not number must fail the read as unavailable
// rather than be served without a position or crash the scan as a 500.
func TestMessageSyncRoutesUnnumberedRowFailsClosed(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "user-1")
	router := newTestRouterWithFlags(t, false, true)
	shopID := createShop(t, router, "user-1", "Unnumbered")
	messageID := createMessage(t, router, "user-1", shopID, "orphan")
	_, err := testDB.Exec(`UPDATE public.shop_messages SET insertion_number=NULL WHERE id=$1`, messageID)
	require.NoError(t, err)

	requireFailure(t, doContractRequest(t, router, http.MethodGet, messageSyncPath(shopID, "initial", nil), nil, "user-1"), http.StatusServiceUnavailable, "unsupported_contract")
	requireFailure(t, reconcile(t, router, shopID, []string{messageID}, "user-1"), http.StatusServiceUnavailable, "unsupported_contract")
}

// A shop that has messages but lost its counter row must never be served with
// an invented watermark. (No messages and no counter is a normal new shop and
// starts at 0; see TestMessageSyncRoutesInitialEmptyShopStartsAtZero.)
func TestMessageSyncRoutesMissingCounterWithMessagesFailsClosed(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "user-1")
	router := newTestRouterWithFlags(t, false, true)
	shopID := createShop(t, router, "user-1", "Lost counter")
	createMessage(t, router, "user-1", shopID, "has a message")
	result, err := testDB.Exec(`DELETE FROM public.shop_message_counters WHERE shop_id=$1`, shopID)
	require.NoError(t, err)
	affected, err := result.RowsAffected()
	require.NoError(t, err)
	require.Equal(t, int64(1), affected, "the counter row must exist before it is deleted")

	requireFailure(t, doContractRequest(t, router, http.MethodGet, messageSyncPath(shopID, "initial", nil), nil, "user-1"), http.StatusServiceUnavailable, "unsupported_contract")
	requireFailure(t, doContractRequest(t, router, http.MethodGet, messageSyncPath(shopID, "catch-up", url.Values{"after": {"0"}}), nil, "user-1"), http.StatusServiceUnavailable, "unsupported_contract")
}

func TestMessageSyncRoutesCatchUpBurstOver100HoldsBound(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "user-1")
	router := newTestRouterWithFlags(t, false, true)
	shopID := createShop(t, router, "user-1", "Burst")

	watermark := decodeSyncPage(t, doContractRequest(t, router, http.MethodGet, messageSyncPath(shopID, "initial", nil), nil, "user-1")).Watermark
	require.Equal(t, "0", watermark)
	for i := 1; i <= 250; i++ {
		require.Equal(t, int64(i), insertMessageNumber(t, testDB, shopID, "user-1"))
	}

	seen := map[string]bool{}
	after := watermark
	var through *string
	pages := 0
	for {
		query := url.Values{"after": {after}}
		if through != nil {
			query.Set("through", *through)
		}
		w := doContractRequest(t, router, http.MethodGet, messageSyncPath(shopID, "catch-up", query), nil, "user-1")
		if pages == 0 {
			t.Logf("SAMPLE catch-up: %s", w.Body.String())
		}
		page := decodeSyncPage(t, w)
		pages++
		require.LessOrEqual(t, len(page.Rows), 100)
		if through == nil {
			held := page.Through
			through = &held
		}
		require.Equal(t, *through, page.Through, "bound must be held for the whole cycle")
		for _, id := range rowIDs(page.Rows) {
			require.False(t, seen[id], "row %s returned twice", id)
			seen[id] = true
		}
		after = page.NextAfter
		if !page.HasMore {
			break
		}
		require.Less(t, pages, 10, "catch-up did not terminate")
	}
	require.Equal(t, "250", *through)
	require.Equal(t, "250", after)
	require.Equal(t, 3, pages)
	require.Len(t, seen, 250)

	var stored int
	require.NoError(t, testDB.QueryRow(`SELECT count(*) FROM public.shop_messages WHERE shop_id=$1`, shopID).Scan(&stored))
	require.Equal(t, 250, stored)

	// The 251st insert lands after the held bound: the finished cycle must not
	// see it, and only the next cycle (fresh bound) returns it.
	require.Equal(t, int64(251), insertMessageNumber(t, testDB, shopID, "user-1"))
	held := catchUp(t, router, shopID, after, through, 0, "user-1")
	require.Empty(t, held.Rows)
	require.Equal(t, "250", held.NextAfter)
	require.Equal(t, "250", held.Through)
	require.False(t, held.HasMore)

	next := catchUp(t, router, shopID, after, nil, 0, "user-1")
	require.Len(t, next.Rows, 1)
	require.False(t, seen[next.Rows[0]["id"].(string)])
	require.Equal(t, "251", next.Through)
	require.Equal(t, "251", next.NextAfter)
	require.False(t, next.HasMore)

	// A bound beyond the committed counter is rejected, never fabricated.
	future := "252"
	requireFailure(t, doContractRequest(t, router, http.MethodGet, messageSyncPath(shopID, "catch-up", url.Values{"after": {"251"}, "through": {future}}), nil, "user-1"), http.StatusConflict, "message_sync_reset_required")
}

func TestMessageSyncRoutesCatchUpSkipsDeletedGaps(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "user-1")
	router := newTestRouterWithFlags(t, false, true)
	shopID := createShop(t, router, "user-1", "Gaps")

	ids := make([]string, 5)
	for i := range ids {
		ids[i] = createMessage(t, router, "user-1", shopID, fmt.Sprintf("message %d", i+1))
	}
	legacyDeleteMessage(t, router, ids[1], "user-1")
	legacyDeleteMessage(t, router, ids[2], "user-1")

	w := doContractRequest(t, router, http.MethodGet, messageSyncPath(shopID, "catch-up", url.Values{"after": {"0"}}), nil, "user-1")
	t.Logf("SAMPLE catch-up with gaps: %s", w.Body.String())
	page := decodeSyncPage(t, w)
	require.Equal(t, []string{ids[0], ids[3], ids[4]}, rowIDs(page.Rows), "insertion order")
	require.Equal(t, "5", page.NextAfter)
	require.Equal(t, "5", page.Through)
	require.False(t, page.HasMore)

	// One row per page: next_after jumps over the gap and still reaches the bound.
	var collected []string
	after := "0"
	var through *string
	var afters []string
	for i := 0; i < 5; i++ {
		page := catchUp(t, router, shopID, after, through, 1, "user-1")
		if through == nil {
			held := page.Through
			through = &held
		}
		collected = append(collected, rowIDs(page.Rows)...)
		after = page.NextAfter
		afters = append(afters, after)
		if !page.HasMore {
			break
		}
	}
	require.Equal(t, []string{ids[0], ids[3], ids[4]}, collected)
	require.Equal(t, []string{"1", "4", "5"}, afters)

	// A range made only of deleted numbers still advances to the bound.
	emptyRange := catchUp(t, router, shopID, "1", nil, 0, "user-1")
	require.Equal(t, []string{ids[3], ids[4]}, rowIDs(emptyRange.Rows))
	throughThree := "3"
	allDeleted := catchUp(t, router, shopID, "1", &throughThree, 0, "user-1")
	require.Empty(t, allDeleted.Rows)
	require.Equal(t, "3", allDeleted.NextAfter)
	require.False(t, allDeleted.HasMore)
}

func TestMessageSyncRoutesHistoryEqualTimestampsAndDeletedAnchor(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "user-1")
	router := newTestRouterWithFlags(t, false, true)
	shopID := createShop(t, router, "user-1", "History")

	// Microsecond-exact so the value round-trips through timestamptz unchanged.
	sameInstant := time.Date(2026, 3, 1, 12, 0, 0, 123456000, time.UTC)
	all := make([]string, 6)
	for i := range all {
		all[i] = insertMessageAt(t, shopID, "user-1", fmt.Sprintf("tied %d", i), sameInstant)
	}
	// Ties are broken by id DESC.
	expected := append([]string(nil), all...)
	sort.Sort(sort.Reverse(sort.StringSlice(expected)))

	initial := decodeSyncPage(t, doContractRequest(t, router, http.MethodGet, messageSyncPath(shopID, "initial", url.Values{"limit": {"2"}}), nil, "user-1"))
	require.Equal(t, expected[:2], rowIDs(initial.Rows))
	require.True(t, initial.HasOlder)
	require.NotNil(t, initial.OlderCursor)

	// Delete the anchor (last row of page 1) before asking for page 2.
	_, err := testDB.Exec(`DELETE FROM public.shop_messages WHERE id=$1`, expected[1])
	require.NoError(t, err)

	w := doContractRequest(t, router, http.MethodGet, messageSyncPath(shopID, "history", url.Values{"cursor": {*initial.OlderCursor}, "limit": {"2"}}), nil, "user-1")
	page2 := decodeSyncPage(t, w)
	require.Equal(t, expected[2:4], rowIDs(page2.Rows))
	require.True(t, page2.HasMore)
	require.NotNil(t, page2.NextCursor)

	page3 := decodeSyncPage(t, doContractRequest(t, router, http.MethodGet, messageSyncPath(shopID, "history", url.Values{"cursor": {*page2.NextCursor}, "limit": {"2"}}), nil, "user-1"))
	require.Equal(t, expected[4:6], rowIDs(page3.Rows))
	require.False(t, page3.HasMore)
	require.Nil(t, page3.NextCursor)

	union := append(append([]string{expected[0]}, rowIDs(page2.Rows)...), rowIDs(page3.Rows)...)
	remaining := append([]string{expected[0]}, expected[2:]...)
	require.Equal(t, remaining, union, "no row lost or repeated")

	// A cursor minted for one Shop is rejected for another.
	otherShop := createShop(t, router, "user-1", "Other")
	requireFailure(t, doContractRequest(t, router, http.MethodGet, messageSyncPath(otherShop, "history", url.Values{"cursor": {*initial.OlderCursor}}), nil, "user-1"), http.StatusBadRequest, "invalid")
}

func TestMessageSyncRoutesReconcile(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "user-1")
	ensureUser(t, testDB, "user-2")
	router := newTestRouterWithFlags(t, false, true)
	shopID := createShop(t, router, "user-1", "Reconcile")
	foreignShop := createShop(t, router, "user-2", "Foreign")

	edited := createMessage(t, router, "user-1", shopID, "before edit")
	untouched := createMessage(t, router, "user-1", shopID, "untouched")
	deleted := createMessage(t, router, "user-1", shopID, "to delete")
	foreign := createMessage(t, router, "user-2", foreignShop, "foreign secret content")
	random := uuid.NewString()

	updateResp := doJSONRequest(t, router, http.MethodPut, "/api/v1/auth/shops/messages", map[string]interface{}{"message_id": edited, "message": "after edit"}, "user-1")
	require.Equal(t, http.StatusOK, updateResp.Code, updateResp.Body.String())
	legacyDeleteMessage(t, router, deleted, "user-1")

	ids := []string{edited, deleted, foreign, random, untouched, edited, strings.ToUpper(untouched)}
	w := reconcile(t, router, shopID, ids, "user-1")
	t.Logf("SAMPLE reconcile: %s", w.Body.String())
	page := decodeSyncPage(t, w)
	require.NotContains(t, w.Body.String(), "foreign secret content")
	require.ElementsMatch(t, []string{edited, untouched}, rowIDs(page.Rows), "duplicates collapse to one row each")
	require.Len(t, page.Rows, 2)
	require.Equal(t, []string{deleted, foreign, random}, page.MissingIDs, "foreign and nonexistent IDs are indistinguishable")
	for _, row := range page.Rows {
		require.Equal(t, shopID, row["shop_id"])
		if row["id"] == edited {
			require.Equal(t, "after edit", row["message"])
			require.Equal(t, true, row["is_edited"])
		} else {
			require.Equal(t, "untouched", row["message"])
			require.Equal(t, false, row["is_edited"])
		}
	}

	empty := decodeSyncPage(t, reconcile(t, router, shopID, []string{}, "user-1"))
	require.Empty(t, empty.Rows)
	require.NotNil(t, empty.MissingIDs)
	require.Empty(t, empty.MissingIDs)

	tooMany := make([]string, 101)
	for i := range tooMany {
		tooMany[i] = uuid.NewString()
	}
	requireFailure(t, reconcile(t, router, shopID, tooMany, "user-1"), http.StatusBadRequest, "invalid")
	requireFailure(t, reconcile(t, router, shopID, []string{"not-a-uuid"}, "user-1"), http.StatusBadRequest, "invalid")

	// Non-members are denied outright, never told that every ID is missing, and
	// an existing foreign Shop is indistinguishable from a nonexistent one.
	requireFailure(t, reconcile(t, router, shopID, []string{edited, random}, "user-2"), http.StatusForbidden, "denied")
	requireFailure(t, reconcile(t, router, uuid.NewString(), []string{edited}, "user-2"), http.StatusForbidden, "denied")
	for _, operation := range []string{"initial", "catch-up"} {
		query := url.Values{}
		if operation == "catch-up" {
			query.Set("after", "0")
		}
		requireFailure(t, doContractRequest(t, router, http.MethodGet, messageSyncPath(shopID, operation, query), nil, "user-2"), http.StatusForbidden, "denied")
		requireFailure(t, doContractRequest(t, router, http.MethodGet, messageSyncPath(uuid.NewString(), operation, query), nil, "user-2"), http.StatusForbidden, "denied")
	}
}

// The physical parent_id FK is ON DELETE CASCADE (the source migration said
// SET NULL): deleting a parent deletes its replies, which must then reconcile
// as missing.
func TestMessageSyncRoutesReconcileParentDeleteCascades(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "user-1")
	router := newTestRouterWithFlags(t, false, true)
	shopID := createShop(t, router, "user-1", "Cascade")

	parent := createMessage(t, router, "user-1", shopID, "parent")
	replyResp := doJSONRequest(t, router, http.MethodPost, "/api/v1/auth/shops/messages", map[string]interface{}{"shop_id": shopID, "message": "reply", "parent_id": parent}, "user-1")
	require.Equal(t, http.StatusCreated, replyResp.Code, replyResp.Body.String())
	reply := decodeMap(t, decodeStandardResponse(t, replyResp.Body).Data)["id"].(string)

	before := decodeSyncPage(t, reconcile(t, router, shopID, []string{parent, reply}, "user-1"))
	require.Len(t, before.Rows, 2)
	for _, row := range before.Rows {
		if row["id"] == reply {
			require.Equal(t, parent, row["parent_id"])
		}
	}

	legacyDeleteMessage(t, router, parent, "user-1")
	after := decodeSyncPage(t, reconcile(t, router, shopID, []string{parent, reply}, "user-1"))
	require.Empty(t, after.Rows)
	require.Equal(t, []string{parent, reply}, after.MissingIDs)
}

func TestMessageSyncRoutesMemberRemovedBetweenChunks(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "user-1")
	ensureUser(t, testDB, "user-2")
	router := newTestRouterWithFlags(t, false, true)
	shopID := createShop(t, router, "user-1", "Chunks")
	addShopMember(t, shopID, "user-2")
	firstChunk := createMessage(t, router, "user-1", shopID, "chunk one")
	secondChunk := createMessage(t, router, "user-1", shopID, "chunk two")

	first := decodeSyncPage(t, reconcile(t, router, shopID, []string{firstChunk}, "user-2"))
	require.Equal(t, []string{firstChunk}, rowIDs(first.Rows))
	require.Empty(t, first.MissingIDs)

	_, err := testDB.Exec(`DELETE FROM public.shop_members WHERE shop_id=$1 AND user_id=$2`, shopID, "user-2")
	require.NoError(t, err)

	requireFailure(t, reconcile(t, router, shopID, []string{secondChunk}, "user-2"), http.StatusForbidden, "denied")
	// The first chunk's result is still what the server holds for that ID.
	stillThere := decodeSyncPage(t, reconcile(t, router, shopID, []string{firstChunk}, "user-1"))
	require.Equal(t, rowIDs(first.Rows), rowIDs(stillThere.Rows))
}

func validMessageCursor(t *testing.T, shopID string) string {
	t.Helper()
	payload, err := json.Marshal(map[string]interface{}{"v": 2, "shop_id": shopID, "created_at": "2026-01-01T00:00:00Z", "id": uuid.NewString()})
	require.NoError(t, err)
	return base64.RawURLEncoding.EncodeToString(payload)
}

// Every request here is otherwise valid, so the only reason to refuse is the flag.
func syncRequestsForShop(t *testing.T, shopID string) []struct {
	method, path string
	body         interface{}
} {
	t.Helper()
	return []struct {
		method, path string
		body         interface{}
	}{
		{http.MethodGet, messageSyncPath(shopID, "initial", nil), nil},
		{http.MethodGet, messageSyncPath(shopID, "history", url.Values{"cursor": {validMessageCursor(t, shopID)}}), nil},
		{http.MethodGet, messageSyncPath(shopID, "catch-up", url.Values{"after": {"0"}}), nil},
		{http.MethodPost, messageSyncPath(shopID, "reconcile", nil), map[string]interface{}{"ids": []string{uuid.NewString()}}},
	}
}

func TestMessageSyncRoutesClosedWhenFlagOff(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "user-1")
	router := newTestRouterWithFlags(t, false, false)
	shopID := createShop(t, router, "user-1", "Flag off")
	createMessage(t, router, "user-1", shopID, "exists")

	for _, request := range syncRequestsForShop(t, shopID) {
		w := doContractRequest(t, router, request.method, request.path, request.body, "user-1")
		requireFailure(t, w, http.StatusServiceUnavailable, "unsupported_contract")
	}
}

func TestMessageSyncRoutesRequireContractHeader(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "user-1")
	router := newTestRouterWithFlags(t, false, true)
	shopID := createShop(t, router, "user-1", "Header")

	for _, request := range syncRequestsForShop(t, shopID) {
		missing := doJSONRequest(t, router, request.method, request.path, request.body, "user-1")
		requireFailure(t, missing, http.StatusBadRequest, "unsupported_contract")
		for _, selector := range []string{"1", "99", "2.0"} {
			w := doJSONRequestWithHeaders(t, router, request.method, request.path, request.body, "user-1", map[string]string{"X-MilTech-Shops-Contract": selector})
			requireFailure(t, w, http.StatusBadRequest, "unsupported_contract")
		}
		// Sanity: the same request with the right selector succeeds.
		ok := doContractRequest(t, router, request.method, request.path, request.body, "user-1")
		require.Equal(t, http.StatusOK, ok.Code, "%s %s: %s", request.method, request.path, ok.Body.String())
	}
}

func TestMessageSyncLegacyCompatibility(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "user-1")
	router := newTestRouterWithFlags(t, false, true)
	shopID := createShop(t, router, "user-1", "Legacy")

	// Released-client request: no contract header, legacy body.
	resp := doJSONRequest(t, router, http.MethodPost, "/api/v1/auth/shops/messages",
		map[string]interface{}{"shop_id": shopID, "message": "hello"}, "user-1")
	require.Equal(t, http.StatusCreated, resp.Code, resp.Body.String())
	created := decodeStandardResponse(t, resp.Body)
	require.Equal(t, http.StatusCreated, created.Status)
	require.Equal(t, "Message created successfully", created.Message)
	data := decodeMap(t, created.Data)
	requireMessageKeys(t, data)
	require.Equal(t, false, data["is_edited"])
	require.Nil(t, data["parent_id"])
	messageID := data["id"].(string)

	var number int64
	require.NoError(t, testDB.QueryRow(`SELECT insertion_number FROM public.shop_messages WHERE id=$1`, messageID).Scan(&number))
	require.Equal(t, int64(1), number)

	replyResp := doJSONRequest(t, router, http.MethodPost, "/api/v1/auth/shops/messages",
		map[string]interface{}{"shop_id": shopID, "message": "reply", "parent_id": messageID}, "user-1")
	require.Equal(t, http.StatusCreated, replyResp.Code, replyResp.Body.String())
	reply := decodeMap(t, decodeStandardResponse(t, replyResp.Body).Data)
	requireMessageKeys(t, reply)
	require.Equal(t, messageID, reply["parent_id"])
	require.NoError(t, testDB.QueryRow(`SELECT insertion_number FROM public.shop_messages WHERE id=$1`, reply["id"]).Scan(&number))
	require.Equal(t, int64(2), number)

	listResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/"+shopID+"/messages", nil, "user-1")
	require.Equal(t, http.StatusOK, listResp.Code, listResp.Body.String())
	list := decodeSlice(t, decodeStandardResponse(t, listResp.Body).Data)
	require.Len(t, list, 2)
	for _, message := range list {
		requireMessageKeys(t, message.(map[string]interface{}))
	}

	pagedResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/"+shopID+"/messages/paginated?page=1&limit=1", nil, "user-1")
	require.Equal(t, http.StatusOK, pagedResp.Code, pagedResp.Body.String())
	paged := decodeMap(t, decodeStandardResponse(t, pagedResp.Body).Data)
	pagedMessages := paged["messages"].([]interface{})
	require.Len(t, pagedMessages, 1)
	requireMessageKeys(t, pagedMessages[0].(map[string]interface{}))
	_, hasPagination := paged["pagination"]
	require.True(t, hasPagination)

	updateResp := doJSONRequest(t, router, http.MethodPut, "/api/v1/auth/shops/messages", map[string]interface{}{"message_id": messageID, "message": "edited"}, "user-1")
	require.Equal(t, http.StatusOK, updateResp.Code, updateResp.Body.String())
	require.Equal(t, "Message updated successfully", decodeMap(t, decodeStandardResponse(t, updateResp.Body).Data)["message"])

	listResp = doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/"+shopID+"/messages", nil, "user-1")
	require.Equal(t, http.StatusOK, listResp.Code)
	for _, message := range decodeSlice(t, decodeStandardResponse(t, listResp.Body).Data) {
		row := message.(map[string]interface{})
		requireMessageKeys(t, row)
		if row["id"] == messageID {
			require.Equal(t, "edited", row["message"])
			require.Equal(t, true, row["is_edited"])
		}
	}
	// Editing never renumbers.
	require.NoError(t, testDB.QueryRow(`SELECT insertion_number FROM public.shop_messages WHERE id=$1`, messageID).Scan(&number))
	require.Equal(t, int64(1), number)

	deleteResp := doJSONRequest(t, router, http.MethodDelete, "/api/v1/auth/shops/messages/"+reply["id"].(string), nil, "user-1")
	require.Equal(t, http.StatusOK, deleteResp.Code, deleteResp.Body.String())
	require.Equal(t, "Message deleted successfully", decodeMap(t, decodeStandardResponse(t, deleteResp.Body).Data)["message"])

	// Legacy failures keep the legacy envelope: no typed code without the selector.
	denied := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/"+shopID+"/messages", nil, "outsider")
	require.NotEqual(t, http.StatusOK, denied.Code)
	var deniedBody failureBody
	require.NoError(t, json.Unmarshal(denied.Body.Bytes(), &deniedBody))
	require.Empty(t, deniedBody.Code)
}
