package messages

import (
	"encoding/base64"
	"encoding/json"
	"net/http/httptest"
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
