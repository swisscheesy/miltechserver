package middleware

import (
	"log/slog"

	"firebase.google.com/go/v4/auth"
	"github.com/gin-gonic/gin"
)

// OptionalAuthMiddleware accepts anonymous requests and invalid optional tokens.
// Verified tokens must pass the same current-account checks as required auth.
func OptionalAuthMiddleware(client *auth.Client) gin.HandlerFunc {
	if client == nil {
		return optionalAuthMiddleware(nil)
	}
	return optionalAuthMiddleware(client)
}

func optionalAuthMiddleware(client identityClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenID, hasBearer := bearerToken(c.GetHeader("Authorization"))
		if !hasBearer {
			c.Next()
			return
		}
		if client == nil {
			abortIdentityError(c, errIdentityUnavailable)
			return
		}
		token, err := client.VerifyIDToken(c.Request.Context(), tokenID)
		if err != nil {
			if classifyIdentityError(err) == errIdentityUnavailable {
				abortIdentityError(c, errIdentityUnavailable)
				return
			}
			slog.Debug("Optional auth: invalid token")
			c.Next()
			return
		}
		user, err := processToken(c.Request.Context(), client, token)
		if err != nil {
			abortIdentityError(c, err)
			return
		}
		c.Set("user", user)
		c.Next()
	}
}
