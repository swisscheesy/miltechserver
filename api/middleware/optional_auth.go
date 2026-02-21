package middleware

import (
	"context"
	"log/slog"
	"strings"

	"firebase.google.com/go/v4/auth"
	"github.com/gin-gonic/gin"
)

// OptionalAuthMiddleware attempts to verify the Bearer token.
// If valid, sets *bootstrap.User in context via ProcessToken. If missing or invalid, continues without user.
func OptionalAuthMiddleware(client *auth.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.Request.Header.Get("Authorization")
		if header == "" {
			c.Next()
			return
		}

		parts := strings.Split(header, "Bearer ")
		if len(parts) != 2 {
			c.Next()
			return
		}

		tokenID := parts[1]
		token, err := client.VerifyIDToken(context.Background(), tokenID)
		if err != nil {
			slog.Debug("Optional auth: invalid token", "error", err)
			c.Next()
			return
		}

		ProcessToken(c, client, token)
	}
}
