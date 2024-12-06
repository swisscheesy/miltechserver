package middleware

import (
	"context"
	"firebase.google.com/go/v4/auth"
	"github.com/gin-gonic/gin"
	"log/slog"
	"strings"
	"time"
)

func AuthenticationMiddleware(client *auth.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()

		header := c.Request.Header.Get("Authorization")
		if header == "" {
			slog.Error("No Authorization header found")
			c.JSON(401, gin.H{"message": "No Authorization header found"})
			c.Abort()
			return
		}
		idToken := strings.Split(header, "Bearer ")
		if len(idToken) != 2 || len(idToken) == 0 {
			slog.Error("Invalid Authorization header")
			c.JSON(401, gin.H{"message": "Invalid Authorization header"})
			c.Abort()
			return
		}

		tokenID := idToken[1]

		_, err := client.VerifyIDToken(context.Background(), tokenID)
		if err != nil {
			slog.Error("Invalid token: %v", err)
			c.JSON(401, gin.H{"message": "Invalid token"})
			c.Abort()
			return
		}
		slog.Info("Auth time:", time.Since(startTime))

		c.Next()
	}
}
