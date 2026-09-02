// Package testutil holds test infrastructure shared across the
// tests/<domain> integration test packages: the integration test
// database DSN and a fake-auth middleware that stands in for real
// Firebase-authenticated requests.
package testutil

import (
	"miltechserver/bootstrap"

	"github.com/gin-gonic/gin"
)

// TestDSN is the connection string for the shared integration test
// database. This is the confirmed-correct, intentional test host —
// do not change it without updating every domain that imports it.
const TestDSN = "postgres://postgres:potato123@192.168.20.70/miltech_ng_test?sslmode=disable"

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
