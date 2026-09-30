package messages

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"miltechserver/api/response"
	"miltechserver/api/shops/shared"
	"miltechserver/bootstrap"
)

func TestMessageSyncRoutesFailClosedAndValidate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	group := router.Group("/api/v1/auth", shared.ContractMiddleware, func(c *gin.Context) {
		if c.GetHeader("Test-Anonymous") == "" {
			c.Set("user", &bootstrap.User{UserID: "user"})
		}
	})
	RegisterRoutes(group, NewService(nil, nil))
	timestamp := time.Now().UTC()
	row := response.ShopMessageResponse{}
	row.ID = syncID
	row.CreatedAt = &timestamp
	cursor, err := encodeMessageCursor(syncShop, row)
	if err != nil {
		t.Fatal(err)
	}
	manyIDs := `{"ids":[` + strings.Repeat(`"`+syncID+`",`, 100) + `"` + syncID + `"]}`
	cases := []struct {
		name, method, path, body, contract string
		status                             int
		code                               string
	}{
		{"initial unavailable", "GET", "initial", "", "2", 503, "unsupported_contract"},
		{"history unavailable", "GET", "history?cursor=" + cursor, "", "2", 503, "unsupported_contract"},
		{"catch up unavailable", "GET", "catch-up?after=0", "", "2", 503, "unsupported_contract"},
		{"reconcile unavailable", "POST", "reconcile", `{"ids":[]}`, "2", 503, "unsupported_contract"},
		{"no contract", "GET", "initial", "", "", 400, "unsupported_contract"},
		{"unknown contract", "GET", "initial", "", "3", 400, "unsupported_contract"},
		{"bad cursor", "GET", "history?cursor=bad", "", "2", 400, "invalid"},
		{"zero limit", "GET", "initial?limit=0", "", "2", 400, "invalid"},
		{"over limit", "GET", "initial?limit=101", "", "2", 400, "invalid"},
		{"malformed URL limit", "GET", "initial?limit=%ZZ", "", "2", 400, "invalid"},
		{"empty limit", "GET", "initial?limit=", "", "2", 400, "invalid"},
		{"repeated limit", "GET", "initial?limit=1&limit=2", "", "2", 400, "invalid"},
		{"unknown query", "GET", "initial?before_id=x", "", "2", 400, "invalid"},
		{"missing after", "GET", "catch-up", "", "2", 400, "invalid"},
		{"negative after", "GET", "catch-up?after=-1", "", "2", 400, "invalid"},
		{"overflow", "GET", "catch-up?after=9223372036854775808", "", "2", 400, "invalid"},
		{"decimal", "GET", "catch-up?after=1.0", "", "2", 400, "invalid"},
		{"reversed bound", "GET", "catch-up?after=3&through=2", "", "2", 400, "invalid"},
		{"empty through", "GET", "catch-up?after=0&through=", "", "2", 400, "invalid"},
		{"too many IDs before dedup", "POST", "reconcile", manyIDs, "2", 400, "invalid"},
		{"invalid ID", "POST", "reconcile", `{"ids":["bad"]}`, "2", 400, "invalid"},
		{"missing IDs", "POST", "reconcile", `{}`, "2", 400, "invalid"},
		{"unknown field", "POST", "reconcile", `{"ids":[],"foreign":true}`, "2", 400, "invalid"},
		{"trailing JSON", "POST", "reconcile", `{"ids":[]} {}`, "2", 400, "invalid"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, "/api/v1/auth/shops/"+syncShop+"/messages-v2/"+tc.path, strings.NewReader(tc.body))
			req.Header.Set(shared.ContractHeader, tc.contract)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			var body struct {
				Code string `json:"code"`
			}
			json.Unmarshal(w.Body.Bytes(), &body)
			if w.Code != tc.status || body.Code != tc.code {
				t.Fatalf("got %d %s", w.Code, w.Body.String())
			}
		})
	}
	req := httptest.NewRequest("GET", "/api/v1/auth/shops/"+syncShop+"/messages-v2/initial", nil)
	req.Header.Set("Test-Anonymous", "1")
	req.Header.Set(shared.ContractHeader, "2")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != 401 {
		t.Fatalf("anonymous: %d", w.Code)
	}
}
func TestMessageSyncCursorRejectsWrongShopVersionAndMalformed(t *testing.T) {
	for _, raw := range []string{
		`{"v":1,"shop_id":"` + syncShop + `","created_at":"2026-01-01T00:00:00Z","id":"` + syncID + `"}`,
		`{"v":2,"shop_id":"` + syncOtherID + `","created_at":"2026-01-01T00:00:00Z","id":"` + syncID + `"}`,
		`{"v":2,"shop_id":"` + syncShop + `","created_at":"2026-01-01T00:00:00Z","id":"bad"}`,
		`{"v":2,"shop_id":"` + syncShop + `","id":"` + syncID + `"}`,
	} {
		if _, err := decodeMessageCursor(syncShop, base64.RawURLEncoding.EncodeToString([]byte(raw))); err == nil {
			t.Fatalf("accepted %s", raw)
		}
	}
}

// fakeSyncRepository embeds Repository so only the sync reader methods need
// implementing; any legacy call would panic on the nil embedded interface.
type fakeSyncRepository struct {
	Repository
	calls []string
	args  []any

	initial   *MessageInitial
	history   *MessageHistory
	catchUp   *MessageCatchUp
	reconcile *MessageReconcile
	err       error
}

func (fake *fakeSyncRepository) InitialMessages(_ context.Context, user *bootstrap.User, shopID string, limit int) (*MessageInitial, error) {
	fake.calls = append(fake.calls, "initial")
	fake.args = []any{user, shopID, limit}
	return fake.initial, fake.err
}

func (fake *fakeSyncRepository) MessageHistory(_ context.Context, user *bootstrap.User, shopID, cursor string, limit int) (*MessageHistory, error) {
	fake.calls = append(fake.calls, "history")
	fake.args = []any{user, shopID, cursor, limit}
	return fake.history, fake.err
}

func (fake *fakeSyncRepository) CatchUpMessages(_ context.Context, user *bootstrap.User, shopID, after string, through *string, limit int) (*MessageCatchUp, error) {
	fake.calls = append(fake.calls, "catch-up")
	fake.args = []any{user, shopID, after, through, limit}
	return fake.catchUp, fake.err
}

func (fake *fakeSyncRepository) ReconcileMessages(_ context.Context, user *bootstrap.User, shopID string, ids []string) (*MessageReconcile, error) {
	fake.calls = append(fake.calls, "reconcile")
	fake.args = []any{user, shopID, ids}
	return fake.reconcile, fake.err
}

func assertSyncUnavailable(t *testing.T, name string, err error) {
	t.Helper()
	var failure *shared.Failure
	if !errors.As(err, &failure) || failure.Status != 503 {
		t.Fatalf("%s: expected 503 failure, got %v", name, err)
	}
}

func invokeAllSyncOperations(service *ServiceImpl, user *bootstrap.User) map[string]error {
	through := "9"
	ids := []string{syncID}
	results := map[string]error{}
	_, results["initial"] = service.InitialMessages(context.Background(), user, syncShop, 7)
	_, results["history"] = service.MessageHistory(context.Background(), user, syncShop, "cursor", 7)
	_, results["catch-up"] = service.CatchUpMessages(context.Background(), user, syncShop, "3", &through, 7)
	_, results["reconcile"] = service.ReconcileMessages(context.Background(), user, syncShop, ids)
	return results
}

func TestMessageSyncServiceFailsClosedWhenFlagOff(t *testing.T) {
	fake := &fakeSyncRepository{}
	for name, err := range invokeAllSyncOperations(NewService(fake, nil), &bootstrap.User{UserID: "user"}) {
		assertSyncUnavailable(t, name, err)
	}
	if len(fake.calls) != 0 {
		t.Fatalf("reader reached while flag off: %v", fake.calls)
	}
}

func TestMessageSyncServiceFailsClosedWhenRepositoryHasNoReader(t *testing.T) {
	// A nil repository and a legacy-only repository both lack SyncReader.
	for name, err := range invokeAllSyncOperations(NewService(nil, nil).WithMessageSync(true), &bootstrap.User{UserID: "user"}) {
		assertSyncUnavailable(t, name, err)
	}
}

func TestMessageSyncServiceDelegatesUnchangedWhenFlagOn(t *testing.T) {
	user := &bootstrap.User{UserID: "user"}
	through := "9"
	ids := []string{syncID}
	readerErr := errors.New("reader failed")
	fake := &fakeSyncRepository{
		initial:   &MessageInitial{Watermark: "5"},
		history:   &MessageHistory{HasMore: true},
		catchUp:   &MessageCatchUp{NextAfter: "4"},
		reconcile: &MessageReconcile{MissingIDs: []string{syncID}},
	}
	service := NewService(fake, nil).WithMessageSync(true)

	initial, err := service.InitialMessages(context.Background(), user, syncShop, 7)
	if err != nil || initial != fake.initial || !reflect.DeepEqual(fake.args, []any{user, syncShop, 7}) {
		t.Fatalf("initial: %v %v %v", initial, err, fake.args)
	}
	history, err := service.MessageHistory(context.Background(), user, syncShop, "cursor", 7)
	if err != nil || history != fake.history || !reflect.DeepEqual(fake.args, []any{user, syncShop, "cursor", 7}) {
		t.Fatalf("history: %v %v %v", history, err, fake.args)
	}
	catchUp, err := service.CatchUpMessages(context.Background(), user, syncShop, "3", &through, 7)
	if err != nil || catchUp != fake.catchUp || !reflect.DeepEqual(fake.args, []any{user, syncShop, "3", &through, 7}) {
		t.Fatalf("catch-up: %v %v %v", catchUp, err, fake.args)
	}
	reconcile, err := service.ReconcileMessages(context.Background(), user, syncShop, ids)
	if err != nil || reconcile != fake.reconcile || !reflect.DeepEqual(fake.args, []any{user, syncShop, ids}) {
		t.Fatalf("reconcile: %v %v %v", reconcile, err, fake.args)
	}

	fake.err = readerErr
	for name, err := range invokeAllSyncOperations(service, user) {
		if !errors.Is(err, readerErr) {
			t.Fatalf("%s: reader error not returned: %v", name, err)
		}
	}
}

func TestMessageSyncFlagSurvivesWithAuthorization(t *testing.T) {
	auth := shared.NewShopAuthorization(nil)
	enabled := NewService(&fakeSyncRepository{}, nil).WithMessageSync(true)
	if !enabled.WithAuthorization(auth).(*ServiceImpl).messageSyncEnabled {
		t.Fatal("enabled flag lost by WithAuthorization")
	}
	disabled := NewService(&fakeSyncRepository{}, nil)
	if disabled.WithAuthorization(auth).(*ServiceImpl).messageSyncEnabled {
		t.Fatal("disabled service became enabled")
	}
	if enabled.WithMessageSync(false).messageSyncEnabled != false || !enabled.messageSyncEnabled {
		t.Fatal("WithMessageSync must return a copy")
	}
}
