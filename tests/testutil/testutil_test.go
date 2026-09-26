package testutil

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"miltechserver/bootstrap"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestDisposableDSNRejectsUnmarkedTargets(t *testing.T) {
	for _, dsn := range []string{"", "postgres://localhost/miltech"} {
		if err := ValidateDisposableTestDSN(dsn, ""); err == nil {
			t.Fatalf("unmarked target accepted")
		}
	}
}

func TestFakeAuthMiddleware_SetsUserFromHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(FakeAuthMiddleware())
	router.GET("/probe", func(c *gin.Context) {
		userValue, exists := c.Get("user")
		require.True(t, exists)

		user, ok := userValue.(*bootstrap.User)
		require.True(t, ok)
		require.Equal(t, "test-uid", user.UserID)
		require.Equal(t, "Test User", user.Username)
		require.Equal(t, "test@example.com", user.Email)
		require.Equal(t, "user", user.Role)

		c.JSON(http.StatusOK, user)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	req.Header.Set("X-User-ID", "test-uid")
	req.Header.Set("X-User-Name", "Test User")
	req.Header.Set("X-User-Email", "test@example.com")
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
}

func TestFakeAuthMiddleware_NoUserIDHeader_SkipsSettingUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(FakeAuthMiddleware())
	router.GET("/probe", func(c *gin.Context) {
		_, exists := c.Get("user")
		require.False(t, exists)
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
}

func TestFakeAuthMiddleware_DefaultsUsernameAndEmail(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(FakeAuthMiddleware())
	router.GET("/probe", func(c *gin.Context) {
		userValue, exists := c.Get("user")
		require.True(t, exists)

		user, ok := userValue.(*bootstrap.User)
		require.True(t, ok)
		require.Equal(t, "test-user", user.Username)
		require.Equal(t, "test-uid@example.com", user.Email)

		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	req.Header.Set("X-User-ID", "test-uid")
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
}

const disposableDSN = "postgres://postgres@127.0.0.1:54321/miltech_test_fixture?sslmode=disable&connect_timeout=5"

var disposableMarker = strings.Repeat("a", 64)

func TestDisposableDSNValidation(t *testing.T) {
	t.Setenv("TEST_DATABASE_URL", disposableDSN)
	require.NoError(t, ValidateDisposableTestDSN(disposableDSN, disposableMarker))
	for _, dsn := range []string{
		"", "host=localhost dbname=miltech_test_fixture",
		strings.Replace(disposableDSN, "127.0.0.1", "example.com", 1),
		strings.Replace(disposableDSN, "miltech_test_fixture", "miltech", 1),
		disposableDSN + "&host=example.com", disposableDSN + "&dbname=miltech",
		disposableDSN + "&sslmode=require", disposableDSN + "#ignored",
		strings.Replace(disposableDSN, ":54321", "", 1),
	} {
		t.Run("reject", func(t *testing.T) {
			t.Setenv("TEST_DATABASE_URL", dsn)
			require.Error(t, ValidateDisposableTestDSN(dsn, disposableMarker))
		})
	}
	require.Error(t, ValidateDisposableTestDSN(disposableDSN, "not-random"))
}

func TestDisposableDSNRejectsConfiguredEffectiveConflict(t *testing.T) {
	t.Setenv("TEST_DATABASE_URL", strings.Replace(disposableDSN, "54321", "54322", 1))
	db, err := OpenDisposableTestDB(disposableDSN, disposableMarker)
	require.Error(t, err)
	require.Nil(t, db)
	require.NotContains(t, err.Error(), "postgres://")
}

// This driver records SQL at the database/sql boundary without any network I/O.
// Returning a DB before the marker SELECT or issuing destructive SQL fails these tests.
type markerDriver struct {
	marker     string
	queryError error
	queries    []string
	closed     bool
}

func (d *markerDriver) Open(string) (driver.Conn, error)             { return d, nil }
func (d *markerDriver) Connect(context.Context) (driver.Conn, error) { return d, nil }
func (d *markerDriver) Driver() driver.Driver                        { return d }
func (d *markerDriver) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("unexpected prepare")
}
func (d *markerDriver) Begin() (driver.Tx, error) { return nil, errors.New("unexpected transaction") }
func (d *markerDriver) Close() error              { d.closed = true; return nil }
func (d *markerDriver) QueryContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Rows, error) {
	d.queries = append(d.queries, query)
	if d.queryError != nil {
		return nil, d.queryError
	}
	return &markerRows{marker: d.marker}, nil
}
func (d *markerDriver) ExecContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Result, error) {
	d.queries = append(d.queries, query)
	return nil, errors.New("destructive SQL before verification")
}

type markerRows struct {
	marker string
	read   bool
}

func (r *markerRows) Columns() []string { return []string{"marker"} }
func (r *markerRows) Close() error      { return nil }
func (r *markerRows) Next(values []driver.Value) error {
	if r.read {
		return io.EOF
	}
	r.read = true
	values[0] = r.marker
	return nil
}
func TestDisposableDBMarkerVerification(t *testing.T) {
	for _, tc := range []struct {
		name, stored string
		queryError   error
		accepted     bool
	}{
		{"matched", disposableMarker, nil, true},
		{"mismatch", strings.Repeat("b", 64), nil, false},
		{"missing table", "", errors.New("relation does not exist; secret diagnostic"), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := &markerDriver{marker: tc.stored, queryError: tc.queryError}
			db := sql.OpenDB(d)
			got, err := verifyAndReturnDisposableDB(db, disposableMarker)
			require.Len(t, d.queries, 1)
			require.Equal(t, "SELECT marker FROM test_infrastructure.disposable_instance WHERE singleton = TRUE", d.queries[0])
			if tc.accepted {
				require.NoError(t, err)
				require.Same(t, db, got)
				require.False(t, d.closed)
				require.NoError(t, got.Close())
			} else {
				require.Error(t, err)
				require.Nil(t, got)
				require.True(t, d.closed)
				require.NotContains(t, err.Error(), "secret diagnostic")
			}
		})
	}
}
