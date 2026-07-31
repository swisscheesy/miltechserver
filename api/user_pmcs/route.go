package user_pmcs

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"

	"miltechserver/api/user_pmcs/community"
	"miltechserver/api/user_pmcs/owned"
	"miltechserver/api/user_pmcs/persistence"
	"miltechserver/api/user_pmcs/shared"
	"miltechserver/api/user_pmcs/subscriptions"
	"miltechserver/api/user_pmcs/sync"
)

type Dependencies struct {
	DB     *sql.DB
	Config shared.Config
}

func RegisterRoutes(
	deps Dependencies,
	publicGroup *gin.RouterGroup,
	authGroup *gin.RouterGroup,
) {
	store := persistence.NewStore(
		deps.DB,
		deps.Config.TransactionMaxAttempts,
	)
	ownedService := owned.NewService(
		owned.NewRepository(store, deps.Config),
		deps.Config,
	)
	communityService := community.NewService(
		community.NewRepository(store, deps.Config),
		deps.Config,
	)
	subscriptionService := subscriptions.NewService(
		subscriptions.NewRepository(store, deps.Config),
		deps.Config,
	)
	syncService := sync.NewService(sync.NewRepository(store), deps.Config)

	observer := defaultObserver()
	limiters := newOperationalLimiters(
		deps.Config,
		time.Now,
		defaultLimiterFactory,
	)
	publicRoutes := publicGroup.Group(
		"",
		observeRequests(observer, time.Now),
		limiters.publicMiddleware(),
	)
	authRoutes := authGroup.Group(
		"",
		observeRequests(observer, time.Now),
		limiters.authenticatedMiddleware(),
	)
	compressedAuthRoutes := authGroup.Group(
		"",
		observeRequests(observer, time.Now),
		limiters.authenticatedMiddleware(),
		gzipGETResponses(),
	)

	community.RegisterPublicRoutes(publicRoutes, communityService)
	community.RegisterRoutes(authRoutes, communityService)
	sync.RegisterRoutes(authRoutes, syncService, deps.Config)
	owned.RegisterRoutes(compressedAuthRoutes, ownedService, deps.Config)
	subscriptions.RegisterRoutes(compressedAuthRoutes, subscriptionService)
}

func gzipGETResponses() gin.HandlerFunc {
	gzipMiddleware := gzip.Gzip(gzip.DefaultCompression)
	return func(context *gin.Context) {
		if context.Request.Method != http.MethodGet {
			context.Next()
			return
		}
		gzipMiddleware(context)
	}
}
