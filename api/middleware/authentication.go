package middleware

import (
	"context"
	"log/slog"
	"miltechserver/api/response"
	"miltechserver/bootstrap"
	"net/http"
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
			abortAuthenticationFailure(c, "No Authorization header found")
			return
		}
		idToken := strings.Split(header, "Bearer ")
		if len(idToken) != 2 || len(idToken) == 0 {
			slog.Error("", "auth_error", "invalid authorization header")
			abortAuthenticationFailure(c, "Invalid Authorization header")
			return
		}

		tokenID := idToken[1]

		token, err := client.VerifyIDToken(context.Background(), tokenID)
		if err != nil {
			slog.Error("Invalid token: ", "auth_error", err)
			abortAuthenticationFailure(c, "Invalid token")
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
		abortAuthenticationFailure(c, "Email not found in token")
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

func abortAuthenticationFailure(c *gin.Context, message string) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, response.StandardResponse{
		Status:  http.StatusUnauthorized,
		Message: message,
		Data:    nil,
	})
}
