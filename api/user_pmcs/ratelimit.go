package user_pmcs

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"

	"miltechserver/api/user_pmcs/shared"
	"miltechserver/bootstrap"
)

const (
	limiterCleanupInterval = time.Minute
	maxCleanupEntries      = 128
)

type limiterFactory func(rate.Limit, int) *rate.Limiter

type limiterEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type keyedLimiter struct {
	mu          sync.Mutex
	entries     map[string]*limiterEntry
	limit       rate.Limit
	burst       int
	idleTTL     time.Duration
	now         func() time.Time
	factory     limiterFactory
	nextCleanup time.Time
}

func defaultLimiterFactory(limit rate.Limit, burst int) *rate.Limiter {
	return rate.NewLimiter(limit, burst)
}

func newKeyedLimiter(
	limit rate.Limit,
	burst int,
	idleTTL time.Duration,
	now func() time.Time,
	factory limiterFactory,
) *keyedLimiter {
	if now == nil {
		now = time.Now
	}
	if factory == nil {
		factory = defaultLimiterFactory
	}
	currentTime := now()
	return &keyedLimiter{
		entries:     make(map[string]*limiterEntry),
		limit:       limit,
		burst:       burst,
		idleTTL:     idleTTL,
		now:         now,
		factory:     factory,
		nextCleanup: currentTime.Add(limiterCleanupInterval),
	}
}

func (limiter *keyedLimiter) allow(key string) bool {
	if key == "" {
		return true
	}

	currentTime := limiter.now()
	limiter.mu.Lock()
	defer limiter.mu.Unlock()

	if !currentTime.Before(limiter.nextCleanup) {
		limiter.cleanupIdleEntries(currentTime)
		limiter.nextCleanup = currentTime.Add(limiterCleanupInterval)
	}

	entry, exists := limiter.entries[key]
	if !exists {
		entry = &limiterEntry{
			limiter: limiter.factory(limiter.limit, limiter.burst),
		}
		limiter.entries[key] = entry
	}
	entry.lastSeen = currentTime
	return entry.limiter.AllowN(currentTime, 1)
}

func (limiter *keyedLimiter) cleanupIdleEntries(currentTime time.Time) {
	inspected := 0
	for key, entry := range limiter.entries {
		if inspected == maxCleanupEntries {
			return
		}
		inspected++
		if currentTime.Sub(entry.lastSeen) >= limiter.idleTTL {
			delete(limiter.entries, key)
		}
	}
}

type operationalLimiters struct {
	publicRequests          *keyedLimiter
	authenticatedReads      *keyedLimiter
	authenticatedMutations  *keyedLimiter
	communityReleasesByUser *keyedLimiter
	communityReleasesByIP   *keyedLimiter
}

func newOperationalLimiters(
	config shared.Config,
	now func() time.Time,
	factory limiterFactory,
) *operationalLimiters {
	idleTTL := time.Duration(config.LimiterIdleMinutes) * time.Minute
	return &operationalLimiters{
		publicRequests: newKeyedLimiter(
			rate.Limit(config.PublicRequestsPerSecond),
			config.PublicRequestBurst,
			idleTTL,
			now,
			factory,
		),
		authenticatedReads: newKeyedLimiter(
			rate.Limit(config.AuthenticatedReadsPerSecond),
			config.AuthenticatedReadBurst,
			idleTTL,
			now,
			factory,
		),
		authenticatedMutations: newKeyedLimiter(
			rate.Limit(config.AuthenticatedMutationsPerSecond),
			config.AuthenticatedMutationBurst,
			idleTTL,
			now,
			factory,
		),
		communityReleasesByUser: newKeyedLimiter(
			perHour(config.ReleasesPerUserPerHour),
			config.ReleaseUserBurst,
			idleTTL,
			now,
			factory,
		),
		communityReleasesByIP: newKeyedLimiter(
			perHour(config.ReleasesPerIPPerHour),
			config.ReleaseIPBurst,
			idleTTL,
			now,
			factory,
		),
	}
}

func perHour(requests int) rate.Limit {
	return rate.Limit(float64(requests) / time.Hour.Seconds())
}

func (limiters *operationalLimiters) publicMiddleware() gin.HandlerFunc {
	return func(context *gin.Context) {
		if !limiters.publicRequests.allow(context.ClientIP()) {
			writeRateLimitExceeded(context)
			return
		}
		context.Next()
	}
}

func (limiters *operationalLimiters) authenticatedMiddleware() gin.HandlerFunc {
	return func(context *gin.Context) {
		uid := verifiedUID(context)
		if uid == "" {
			context.Next()
			return
		}

		if context.Request.Method == http.MethodGet {
			if !limiters.authenticatedReads.allow(uid) {
				writeRateLimitExceeded(context)
				return
			}
			context.Next()
			return
		}

		if !limiters.authenticatedMutations.allow(uid) {
			writeRateLimitExceeded(context)
			return
		}
		if isCommunityRelease(context) {
			if !limiters.communityReleasesByIP.allow(context.ClientIP()) ||
				!limiters.communityReleasesByUser.allow(uid) {
				writeRateLimitExceeded(context)
				return
			}
		}
		context.Next()
	}
}

func verifiedUID(context *gin.Context) string {
	value, exists := context.Get("user")
	if !exists {
		return ""
	}
	user, ok := value.(*bootstrap.User)
	if !ok || user == nil || strings.TrimSpace(user.UserID) == "" {
		return ""
	}
	return user.UserID
}

func isCommunityRelease(context *gin.Context) bool {
	return context.Request.Method == http.MethodPut &&
		strings.HasSuffix(
			context.FullPath(),
			"/checklists/:checklist_id/community-releases/:revision_id",
		)
}

func writeRateLimitExceeded(context *gin.Context) {
	context.Abort()
	shared.WriteAPIError(
		context,
		shared.NewRateLimited("request rate limit exceeded", nil),
	)
}
