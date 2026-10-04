package middleware

import (
	"context"
	"errors"
	"log/slog"
	"miltechserver/api/response"
	"miltechserver/api/shops/shared"
	"miltechserver/bootstrap"
	"net/http"
	"strings"
	"time"

	"firebase.google.com/go/v4/auth"
	"github.com/gin-gonic/gin"
)

type identityClient interface {
	VerifyIDToken(context.Context, string) (*auth.Token, error)
	GetUser(context.Context, string) (*auth.UserRecord, error)
}

var (
	errUnauthorizedIdentity = errors.New("unauthorized identity")
	errMissingTokenEmail    = errors.New("email not found in token")
	errIdentityUnavailable  = errors.New("identity service unavailable")
)

func AuthenticationMiddleware(client *auth.Client) gin.HandlerFunc {
	if client == nil {
		return authenticationMiddleware(nil)
	}
	return authenticationMiddleware(client)
}

func authenticationMiddleware(client identityClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()
		header := c.GetHeader("Authorization")
		if header == "" {
			abortAuthenticationFailure(c, "No Authorization header found")
			return
		}
		tokenID, hasBearer := bearerToken(header)
		if !hasBearer {
			abortAuthenticationFailure(c, "Invalid Authorization header")
			return
		}
		if client == nil {
			abortIdentityError(c, errIdentityUnavailable)
			return
		}
		token, err := client.VerifyIDToken(c.Request.Context(), tokenID)
		if err != nil {
			abortIdentityError(c, classifyIdentityError(err))
			return
		}
		user, err := processToken(c.Request.Context(), client, token)
		if err != nil {
			abortIdentityError(c, err)
			return
		}
		c.Set("user", user)
		slog.Info("Auth process completed", "auth_time", time.Since(startTime))
		c.Next()
	}
}

func ProcessToken(c *gin.Context, client *auth.Client, token *auth.Token) {
	var identity identityClient
	if client != nil {
		identity = client
	}
	user, err := processToken(c.Request.Context(), identity, token)
	if err != nil {
		abortIdentityError(c, err)
		return
	}
	c.Set("user", user)
	c.Next()
}

func processToken(ctx context.Context, client identityClient, token *auth.Token) (*bootstrap.User, error) {
	if token == nil || token.UID == "" {
		return nil, errUnauthorizedIdentity
	}
	email, hasEmail := token.Claims["email"].(string)
	if !hasEmail || email == "" {
		return nil, errMissingTokenEmail
	}
	if client == nil {
		return nil, errIdentityUnavailable
	}
	currentUser, err := client.GetUser(ctx, token.UID)
	if err != nil {
		return nil, classifyIdentityError(err)
	}
	if currentUser == nil || currentUser.UserInfo == nil || currentUser.UID != token.UID || currentUser.Disabled {
		return nil, errUnauthorizedIdentity
	}
	// Match the pinned SDK checked verifier while reusing the display-name lookup.
	if token.IssuedAt*1000 < currentUser.TokensValidAfterMillis {
		return nil, errUnauthorizedIdentity
	}
	return &bootstrap.User{UserID: token.UID, Username: currentUser.DisplayName, Email: email}, nil
}

func classifyIdentityError(err error) error {
	if auth.IsIDTokenInvalid(err) || auth.IsUserNotFound(err) {
		return errUnauthorizedIdentity
	}
	return errIdentityUnavailable
}

func abortIdentityError(c *gin.Context, err error) {
	if errors.Is(err, errIdentityUnavailable) {
		slog.Warn("Authentication denied", "category", "identity_unavailable")
		abortAuthenticationResponse(c, http.StatusServiceUnavailable, "unavailable", "Authentication service unavailable")
		return
	}
	if errors.Is(err, errMissingTokenEmail) {
		abortAuthenticationFailure(c, "Email not found in token")
		return
	}
	abortAuthenticationFailure(c, "Invalid token")
}

func abortAuthenticationFailure(c *gin.Context, message string) {
	slog.Warn("Authentication denied", "category", "unauthorized")
	abortAuthenticationResponse(c, http.StatusUnauthorized, "unauthorized", message)
}

func abortAuthenticationResponse(c *gin.Context, status int, code, message string) {
	// FullPath is the matched server route, not the untrusted request URL.
	// Equipment-service routes are registered under this same Shops namespace.
	route := c.FullPath()
	if c.GetHeader(shared.ContractHeader) == "2" && (route == "/api/v1/auth/shops" || strings.HasPrefix(route, "/api/v1/auth/shops/")) {
		c.AbortWithStatusJSON(status, response.StandardResponse{Status: status, Code: code, Message: code})
		return
	}
	c.AbortWithStatusJSON(status, response.StandardResponse{Status: status, Message: message, Data: nil})
}

func bearerToken(header string) (string, bool) {
	token, hasBearer := strings.CutPrefix(header, "Bearer ")
	return token, hasBearer && token != "" && !strings.ContainsAny(token, " \t\r\n")
}
