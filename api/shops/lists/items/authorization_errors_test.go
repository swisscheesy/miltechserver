package items

import (
	"bytes"
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"

	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/middleware"
	"miltechserver/api/response"
	"miltechserver/api/shops/shared"
	"miltechserver/bootstrap"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// This connector injects query failures without opening a network connection.
type authorizationFailureConnector struct {
	failAt int
	cause  error
}

func (c authorizationFailureConnector) Connect(context.Context) (driver.Conn, error) {
	return &authorizationFailureConn{config: c}, nil
}
func (c authorizationFailureConnector) Driver() driver.Driver { return authorizationFailureDriver{} }

type authorizationFailureDriver struct{}

func (authorizationFailureDriver) Open(string) (driver.Conn, error) {
	return nil, errors.New("connector required")
}

type authorizationFailureConn struct {
	config  authorizationFailureConnector
	queries int
}

func (*authorizationFailureConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("unexpected prepare")
}
func (*authorizationFailureConn) Close() error              { return nil }
func (*authorizationFailureConn) Begin() (driver.Tx, error) { return authorizationFailureTx{}, nil }

type authorizationFailureTx struct{}

func (authorizationFailureTx) Commit() error   { return nil }
func (authorizationFailureTx) Rollback() error { return nil }
func (c *authorizationFailureConn) QueryContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Rows, error) {
	c.queries++
	if c.queries == c.config.failAt {
		return nil, c.config.cause
	}
	var values []driver.Value
	switch {
	case strings.HasPrefix(query, "SELECT role "):
		values = []driver.Value{"member"}
	case strings.HasPrefix(query, "SELECT admin_only_lists "):
		values = []driver.Value{false}
	case strings.HasPrefix(query, "SELECT list_id "):
		values = []driver.Value{"list"}
	case strings.HasPrefix(query, "SELECT shop_id,"):
		values = []driver.Value{"shop", "user"}
	case strings.HasPrefix(query, "SELECT shop_id "):
		values = []driver.Value{"shop"}
	default:
		return nil, fmt.Errorf("unexpected query: %s", query)
	}
	return &authorizationFailureRows{values: values}, nil
}

type authorizationFailureRows struct {
	values []driver.Value
	read   bool
}

func (r *authorizationFailureRows) Columns() []string {
	columns := make([]string, len(r.values))
	for i := range columns {
		columns[i] = fmt.Sprint(i)
	}
	return columns
}
func (*authorizationFailureRows) Close() error { return nil }
func (r *authorizationFailureRows) Next(dest []driver.Value) error {
	if r.read {
		return io.EOF
	}
	copy(dest, r.values)
	r.read = true
	return nil
}

type authorizationFailureService struct {
	Service
	authorize func() error
	lastError error
}

func (s *authorizationFailureService) AddListItemBatch(context.Context, *bootstrap.User, []model.ShopListItems) ([]response.ShopListItemWithUsername, error) {
	s.lastError = s.authorize()
	if s.lastError == nil {
		return nil, errors.New("injected failure was not reached")
	}
	return nil, fmt.Errorf("failed to add list items: %w", s.lastError)
}

func TestAuthorizationFailuresKeepDatabaseDetailsOutOfHandlerResponses(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, resource := range []string{"list-item", "message", "vehicle"} {
		stages := 6
		if resource != "list-item" {
			stages = 4
		}
		for stage := 1; stage <= stages; stage++ {
			for _, cause := range []error{sql.ErrNoRows, errors.New("pq: connection failed password=private-driver-marker SELECT role FROM shop_members")} {
				t.Run(fmt.Sprintf("%s/stage-%d/%v", resource, stage, errors.Is(cause, sql.ErrNoRows)), func(t *testing.T) {
					db := sql.OpenDB(authorizationFailureConnector{failAt: stage, cause: cause})
					defer db.Close()
					tx, err := db.Begin()
					require.NoError(t, err)
					defer tx.Rollback()
					var logs bytes.Buffer
					originalLogger := slog.Default()
					slog.SetDefault(slog.New(slog.NewTextHandler(&logs, nil)))
					defer slog.SetDefault(originalLogger)
					service := &authorizationFailureService{authorize: func() error {
						if resource == "list-item" {
							return shared.AuthorizeListMutation(context.Background(), tx, "user", "", nil, []string{"item"}, false)
						}
						return shared.AuthorizeOwnedMutation(context.Background(), tx, "user", resource, "id", true)
					}}
					handler := Handler{service: service}
					router := gin.New()
					router.Use(middleware.ErrorHandler)
					router.Use(func(c *gin.Context) { c.Set("user", &bootstrap.User{UserID: "user"}) })
					router.POST("/items/bulk", handler.AddListItemBatch)
					req := httptest.NewRequest("POST", "/items/bulk", strings.NewReader(`{"list_id":"list","items":[{"list_id":"list","niin":"1","nomenclature":"item","quantity":1}]}`))
					req.Header.Set("Content-Type", "application/json")
					out := httptest.NewRecorder()
					router.ServeHTTP(out, req)
					require.Equal(t, 500, out.Code)
					var body response.StandardResponse
					require.NoError(t, json.Unmarshal(out.Body.Bytes(), &body))
					require.Equal(t, 500, body.Status)
					require.Nil(t, body.Data)
					require.NotContains(t, out.Body.String(), cause.Error())
					require.NotContains(t, out.Body.String(), "private-driver-marker")
					require.NotContains(t, out.Body.String(), "SELECT ")
					require.NotContains(t, out.Body.String(), "sql:")
					require.ErrorIs(t, service.lastError, cause, "diagnostic cause must remain available")
					if errors.Is(cause, sql.ErrNoRows) {
						missingMessages := []string{"list item not found", "list not found", "shop not found", "access denied: not a member of this shop", "list not found", "list item not found"}
						if resource != "list-item" {
							missingMessages = []string{"resource not found", "shop not found", "access denied: not a member of this shop", "resource not found"}
						}
						require.Equal(t, "failed to add list items: "+missingMessages[stage-1], body.Message)
					} else {
						require.Equal(t, "failed to add list items: failed to verify shop authorization", body.Message)
						require.Contains(t, logs.String(), cause.Error())
					}
				})
			}
		}
	}
}
