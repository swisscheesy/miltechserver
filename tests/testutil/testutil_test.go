package testutil

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"miltechserver/bootstrap"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestTestDSN_IsSet(t *testing.T) {
	require.NotEmpty(t, TestDSN)
	require.Contains(t, TestDSN, "192.168.20.70")
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
