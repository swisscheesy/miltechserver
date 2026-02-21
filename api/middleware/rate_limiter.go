package middleware

import (
	"log/slog"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// ipLimiter holds a per-IP token-bucket rate limiter.
type ipLimiter struct {
	limiter *rate.Limiter
}

var (
	ipLimiters sync.Map // map[string]*ipLimiter
)

// getLimiter returns (or creates) a rate limiter for the given IP address.
func getLimiter(ip string) *rate.Limiter {
	v, _ := ipLimiters.LoadOrStore(ip, &ipLimiter{
		// 2 tokens/second sustained, burst of up to 10 requests.
		limiter: rate.NewLimiter(2, 10),
	})
	return v.(*ipLimiter).limiter
}

// RateLimiter returns a Gin middleware that limits requests per client IP.
// Sustained rate: 2 req/s. Burst: 10 requests.
// Responds with 429 Too Many Requests when the bucket is exhausted.
func RateLimiter() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		limiter := getLimiter(ip)

		if !limiter.Allow() {
			slog.Warn("Rate limit exceeded", "ip", ip, "path", c.FullPath())
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "Too many requests. Please wait before retrying.",
			})
			return
		}

		c.Next()
	}
}
