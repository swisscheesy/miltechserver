package middleware

import (
	"context"
	"log/slog"
	"miltechserver/bootstrap"
	"strings"
	"time"

	"firebase.google.com/go/v4/auth"
	"github.com/gin-gonic/gin"
)

func AuthenticationMiddleware(client *auth.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()

		header := c.Request.Header.Get("Authorization")
		if header == "" {
			slog.Error("", "auth_error", "no authorization header found")
			c.JSON(401, gin.H{"message": "No Authorization header found"})
			c.Abort()
			return
		}
		idToken := strings.Split(header, "Bearer ")
		if len(idToken) != 2 || len(idToken) == 0 {
			slog.Error("", "auth_error", "invalid authorization header")
			c.JSON(401, gin.H{"message": "Invalid Authorization header"})
			c.Abort()
			return
		}

		tokenID := idToken[1]

		token, err := client.VerifyIDToken(context.Background(), tokenID)
		if err != nil {
			slog.Error("Invalid token: ", "auth_error", err)
			c.JSON(401, gin.H{"message": "Invalid token"})
			c.Abort()
			return
		}

		slog.Info("Auth process completed ", "auth_time", time.Since(startTime))

		ProcessToken(c, client, token)
		//c.Next()
	}
}

func ProcessToken(c *gin.Context, auth *auth.Client, token *auth.Token) {
	email, ok := token.Claims["email"].(string)
	if !ok {
		slog.Error("", "auth_error", "email not found in token")
		c.JSON(401, gin.H{"message": "Email not found in token"})
		c.Abort()
		return
	}
	username, err := auth.GetUser(context.Background(), token.UID)
	if err != nil {
		slog.Error("Error getting user: ", "error", err)
	}

	// role, ok := token.Claims["role"].(string)

	user := &bootstrap.User{
		UserID:   token.UID,
		Username: username.DisplayName,
		Email:    email,
	}
	c.Set("user", user)

	c.Next()

}
