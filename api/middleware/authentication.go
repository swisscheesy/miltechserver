package middleware

import (
	"github.com/gin-gonic/gin"
	"log/slog"
	"strings"
	"time"
)

func AuthenticationMiddleware(c *gin.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()

		header := c.Request.Header.Get("Authorization")
		if header == "" {
			slog.Error("No Authorization header found")
			c.JSON(401, gin.H{"message": "No Authorization header found"})
		}
		idToken := strings.Split(header, "Bearer ")
		if len(idToken) != 2 {
			slog.Error("Invalid Authorization header")
			c.JSON(401, gin.H{"message": "Invalid Authorization header"})
		}

		tokenID := idToken[1]

		token, err :=

			c.Next()
	}
}
