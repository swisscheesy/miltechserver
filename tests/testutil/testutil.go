// Package testutil holds test infrastructure shared across the
// tests/<domain> integration test packages: the integration test
// database verification and a fake-auth middleware that stands in for real
// Firebase-authenticated requests.
package testutil

import (
	"context"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"errors"
	_ "github.com/lib/pq"
	"miltechserver/bootstrap"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// ValidateDisposableTestDSN rejects implicit configuration and target overrides.
// A valid URL is only a preliminary check: the stored random marker is required too.
func ValidateDisposableTestDSN(dsn string, marker string) error {
	if dsn == "" {
		return errors.New("TEST_DATABASE_URL is required; use scripts/test-shops-isolated.sh")
	}
	if configured := os.Getenv("TEST_DATABASE_URL"); configured != "" && configured != dsn {
		return errors.New("configured and effective test database URLs differ")
	}
	decoded, err := hex.DecodeString(marker)
	if err != nil || len(decoded) != 32 {
		return errors.New("TEST_DATABASE_MARKER must be a 32-byte random hex token")
	}
	u, err := url.Parse(dsn)
	if err != nil {
		return errors.New("invalid test database URL")
	}
	if (u.Scheme != "postgres" && u.Scheme != "postgresql") || u.Fragment != "" || u.User == nil || u.User.Username() == "" {
		return errors.New("explicit PostgreSQL test database URL required")
	}
	if u.Hostname() != "127.0.0.1" && u.Hostname() != "::1" {
		return errors.New("test database must use a loopback IP")
	}
	port, err := strconv.Atoi(u.Port())
	if err != nil || port < 1 || port > 65535 {
		return errors.New("explicit test database port required")
	}
	name := strings.TrimPrefix(u.Path, "/")
	if !strings.HasPrefix(name, "miltech_test_") || strings.ContainsAny(name, "/\\") {
		return errors.New("disposable test database name required")
	}
	query, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		return errors.New("invalid test database options")
	}
	for key, values := range query {
		if (key != "sslmode" && key != "connect_timeout") || len(values) != 1 {
			return errors.New("unsupported or duplicate test database option")
		}
	}
	if query.Get("sslmode") != "disable" || query.Get("connect_timeout") != "5" {
		return errors.New("explicit sslmode=disable and connect_timeout=5 required for disposable database")
	}
	return nil
}

// OpenDisposableTestDB returns a database only after verifying instance ownership.
func OpenDisposableTestDB(dsn, marker string) (*sql.DB, error) {
	if err := ValidateDisposableTestDSN(dsn, marker); err != nil {
		return nil, err
	}
	return openAndVerifyDisposableDB(dsn, marker)
}

func openAndVerifyDisposableDB(dsn, marker string) (*sql.DB, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, errors.New("could not open disposable test database")
	}
	return verifyAndReturnDisposableDB(db, marker)
}

func verifyAndReturnDisposableDB(db *sql.DB, marker string) (*sql.DB, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var stored string
	err := db.QueryRowContext(ctx, "SELECT marker FROM test_infrastructure.disposable_instance WHERE singleton = TRUE").Scan(&stored)
	if err != nil || subtle.ConstantTimeCompare([]byte(stored), []byte(marker)) != 1 {
		_ = db.Close()
		// Driver diagnostics can contain connection details. Keep refusal errors secret-free.
		return nil, errors.New("disposable test database marker verification failed")
	}
	return db, nil
}

// FakeAuthMiddleware reads the X-User-ID/X-User-Name/X-User-Email
// headers and sets a *bootstrap.User on the gin context under the
// "user" key, replicating the shape of real Firebase-authenticated
// requests for integration tests without requiring a real token.
//
// This is a faithful copy of the testUserMiddleware() implementation
// previously duplicated identically across tests/shops/helpers_test.go
// and five other domain helpers_test.go files — same header names,
// same context key, same User struct construction.
func FakeAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetHeader("X-User-ID")
		if userID == "" {
			c.Next()
			return
		}

		user := &bootstrap.User{
			UserID:   userID,
			Username: c.GetHeader("X-User-Name"),
			Email:    c.GetHeader("X-User-Email"),
			Role:     "user",
		}

		if user.Username == "" {
			user.Username = "test-user"
		}
		if user.Email == "" {
			user.Email = userID + "@example.com"
		}

		c.Set("user", user)
		c.Next()
	}
}
