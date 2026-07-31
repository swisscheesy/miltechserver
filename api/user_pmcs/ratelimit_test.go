package user_pmcs

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"

	"miltechserver/api/user_pmcs/shared"
	"miltechserver/bootstrap"
)

type fakeClock struct {
	mu  sync.Mutex
	now time.Time
}

func (clock *fakeClock) Now() time.Time {
	clock.mu.Lock()
	defer clock.mu.Unlock()
	return clock.now
}

func (clock *fakeClock) Advance(duration time.Duration) {
	clock.mu.Lock()
	defer clock.mu.Unlock()
	clock.now = clock.now.Add(duration)
}

func TestKeyedLimiterSeparatesKeysAndCleansIdleEntries(t *testing.T) {
	clock := &fakeClock{now: time.Unix(1_000, 0)}
	limiter := newKeyedLimiter(
		rate.Limit(1),
		1,
		15*time.Minute,
		clock.Now,
		defaultLimiterFactory,
	)

	require.True(t, limiter.allow("first"))
	require.False(t, limiter.allow("first"))
	require.True(t, limiter.allow("second"))
	require.Equal(t, 2, keyedLimiterEntryCount(limiter))

	clock.Advance(16 * time.Minute)
	require.True(t, limiter.allow("third"))
	require.Equal(t, 1, keyedLimiterEntryCount(limiter))
}

func TestKeyedLimiterCreatesOneBucketUnderConcurrentAccess(t *testing.T) {
	clock := &fakeClock{now: time.Unix(2_000, 0)}
	var factoryCalls atomic.Int32
	factory := func(limit rate.Limit, burst int) *rate.Limiter {
		factoryCalls.Add(1)
		return rate.NewLimiter(limit, burst)
	}
	limiter := newKeyedLimiter(
		rate.Limit(100),
		200,
		15*time.Minute,
		clock.Now,
		factory,
	)

	var waitGroup sync.WaitGroup
	results := make(chan bool, 100)
	for range 100 {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			results <- limiter.allow("shared-user")
		}()
	}
	waitGroup.Wait()
	close(results)

	for allowed := range results {
		require.True(t, allowed)
	}
	require.Equal(t, int32(1), factoryCalls.Load())
	require.Equal(t, 1, keyedLimiterEntryCount(limiter))
}

func TestPublicLimiterReturnsStable429EnvelopePerClientIP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	clock := &fakeClock{now: time.Unix(3_000, 0)}
	config := shared.DefaultConfig()
	config.PublicRequestsPerSecond = 1
	config.PublicRequestBurst = 1
	limiters := newOperationalLimiters(config, clock.Now, defaultLimiterFactory)

	router := gin.New()
	router.GET("/api/v1/user-pmcs/community", limiters.publicMiddleware(), func(context *gin.Context) {
		context.Status(http.StatusNoContent)
	})

	first := performLimiterRequest(router, http.MethodGet, "/api/v1/user-pmcs/community", "192.0.2.1:1000", "")
	require.Equal(t, http.StatusNoContent, first.Code)

	second := performLimiterRequest(router, http.MethodGet, "/api/v1/user-pmcs/community", "192.0.2.1:2000", "")
	require.Equal(t, http.StatusTooManyRequests, second.Code)
	var envelope map[string]any
	require.NoError(t, json.Unmarshal(second.Body.Bytes(), &envelope))
	require.Equal(t, float64(http.StatusTooManyRequests), envelope["status"])
	require.Nil(t, envelope["data"])
	errorBody, ok := envelope["error"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "rate_limited", errorBody["code"])

	otherIP := performLimiterRequest(router, http.MethodGet, "/api/v1/user-pmcs/community", "192.0.2.2:1000", "")
	require.Equal(t, http.StatusNoContent, otherIP.Code)
}

func TestCommunityReleaseUsesSeparateUserAndIPBuckets(t *testing.T) {
	gin.SetMode(gin.TestMode)
	clock := &fakeClock{now: time.Unix(4_000, 0)}
	config := shared.DefaultConfig()
	config.AuthenticatedMutationsPerSecond = 100
	config.AuthenticatedMutationBurst = 100
	config.ReleasesPerUserPerHour = 1
	config.ReleaseUserBurst = 1
	config.ReleasesPerIPPerHour = 1
	config.ReleaseIPBurst = 1
	limiters := newOperationalLimiters(config, clock.Now, defaultLimiterFactory)

	router := gin.New()
	router.Use(func(context *gin.Context) {
		context.Set("user", &bootstrap.User{UserID: context.GetHeader("X-Test-UID")})
		context.Next()
	})
	router.PUT(
		"/api/v1/auth/user-pmcs/checklists/:checklist_id/community-releases/:revision_id",
		limiters.authenticatedMiddleware(),
		func(context *gin.Context) {
			context.Status(http.StatusNoContent)
		},
	)

	first := performLimiterRequest(router, http.MethodPut, "/api/v1/auth/user-pmcs/checklists/one/community-releases/one", "192.0.2.10:1000", "user-1")
	require.Equal(t, http.StatusNoContent, first.Code)

	sameIPDifferentUser := performLimiterRequest(router, http.MethodPut, "/api/v1/auth/user-pmcs/checklists/two/community-releases/two", "192.0.2.10:2000", "user-2")
	require.Equal(t, http.StatusTooManyRequests, sameIPDifferentUser.Code)

	sameUserDifferentIP := performLimiterRequest(router, http.MethodPut, "/api/v1/auth/user-pmcs/checklists/three/community-releases/three", "192.0.2.11:1000", "user-1")
	require.Equal(t, http.StatusTooManyRequests, sameUserDifferentIP.Code)
}

func keyedLimiterEntryCount(limiter *keyedLimiter) int {
	limiter.mu.Lock()
	defer limiter.mu.Unlock()
	return len(limiter.entries)
}

func performLimiterRequest(
	router http.Handler,
	method string,
	path string,
	remoteAddress string,
	uid string,
) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, nil)
	request.RemoteAddr = remoteAddress
	if uid != "" {
		request.Header.Set("X-Test-UID", uid)
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}
