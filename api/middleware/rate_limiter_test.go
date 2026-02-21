package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestRateLimiterAllowsNormalTraffic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/test", RateLimiter(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	// Within the burst limit (10), all requests should succeed.
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "192.0.2.1:1234"
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		require.Equal(t, http.StatusOK, resp.Code, "request %d should be allowed", i+1)
	}
}

func TestRateLimiterBlocksAfterBurst(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/test", RateLimiter(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	// Use a unique IP so this test has its own fresh limiter.
	uniqueIP := "192.0.2.99:5678"

	// Exhaust the burst (10 tokens).
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = uniqueIP
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
	}

	// The 11th request should be rate-limited.
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = uniqueIP
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusTooManyRequests, resp.Code)
}
