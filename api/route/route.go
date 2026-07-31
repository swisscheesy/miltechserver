package route

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"

	"miltechserver/api/analytics"
	"miltechserver/api/docs_equipment"
	"miltechserver/api/eic"
	"miltechserver/api/equipment_services"
	"miltechserver/api/item_comments"
	"miltechserver/api/item_lookup"
	"miltechserver/api/item_query"
	"miltechserver/api/library"
	"miltechserver/api/material_images"
	"miltechserver/api/middleware"
	"miltechserver/api/pmcs_sbs_progress"
	"miltechserver/api/pol_products"
	"miltechserver/api/quick_lists"
	"miltechserver/api/sb_700_20"
	"miltechserver/api/tmde"
	"miltechserver/api/user_general"
	"miltechserver/api/user_pmcs"
	userpmcsshared "miltechserver/api/user_pmcs/shared"
	"miltechserver/api/user_saves"
	"miltechserver/api/user_suggestions"
	"miltechserver/api/user_vehicles"
	"miltechserver/bootstrap"

	"firebase.google.com/go/v4/auth"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/gin-gonic/gin"
)

func NewEngine() *gin.Engine {
	router := gin.New()
	router.Use(gin.LoggerWithFormatter(formatAccessLog))
	router.Use(gin.Recovery())
	return router
}

func formatAccessLog(params gin.LogFormatterParams) string {
	path := params.Path
	if isUserPmcsAccessPath(params.Request.URL.Path) {
		path = params.Request.URL.Path
	}

	return fmt.Sprintf(
		"[GIN] %v | %3d | %13v | %15s | %-7s %#v\n%s",
		params.TimeStamp.Format("2006/01/02 - 15:04:05"),
		params.StatusCode,
		params.Latency,
		params.ClientIP,
		params.Method,
		path,
		params.ErrorMessage,
	)
}

func isUserPmcsAccessPath(path string) bool {
	for _, prefix := range []string{
		"/api/v1/user-pmcs",
		"/api/v1/auth/user-pmcs",
	} {
		if path == prefix || strings.HasPrefix(path, prefix+"/") {
			return true
		}
	}
	return false
}

func Setup(db *sql.DB, router *gin.Engine, authClient *auth.Client, env *bootstrap.Env, blobClient *azblob.Client) {
	v1Route := router.Group("/api/v1")
	v1Route.Use(middleware.ErrorHandler)

	testRoutes := router.Group("/api/v1/test")
	testRoutes.Use(middleware.AuthenticationMiddleware(authClient))
	NewTestRouter(db, testRoutes)

	authRoutes := router.Group("/api/v1/auth")
	authRoutes.Use(middleware.AuthenticationMiddleware(authClient))
	userPmcsConfig := userpmcsshared.DefaultConfig()
	if env != nil {
		var err error
		userPmcsConfig, err = userpmcsshared.ConfigFromEnv(env)
		if err != nil {
			panic(err)
		}
	}
	user_pmcs.RegisterRoutes(user_pmcs.Dependencies{
		DB:     db,
		Config: userPmcsConfig,
	}, v1Route, authRoutes)

	v1Route.Use(middleware.LoggerMiddleware())
	// All Public Routes
	NewGeneralRouter(v1Route, env)
	NewGeneralQueriesRouter(v1Route, env)
	item_query.RegisterRoutes(item_query.Dependencies{DB: db}, v1Route)
	item_lookup.RegisterRoutes(item_lookup.Dependencies{DB: db}, v1Route)
	quick_lists.RegisterRoutes(quick_lists.Dependencies{DB: db}, v1Route)
	pol_products.RegisterRoutes(pol_products.Dependencies{DB: db}, v1Route)
	eic.RegisterRoutes(eic.Dependencies{DB: db}, v1Route)
	tmde.RegisterRoutes(tmde.Dependencies{DB: db}, v1Route)
	sb_700_20.RegisterRoutes(sb_700_20.Dependencies{DB: db}, v1Route)
	docs_equipment.RegisterRoutes(docs_equipment.Dependencies{DB: db, BlobClient: blobClient}, v1Route)

	// All Authenticated Routes
	user_saves.RegisterRoutes(user_saves.Dependencies{
		DB:         db,
		BlobClient: blobClient,
		Env:        env,
	}, authRoutes)
	user_general.RegisterRoutes(user_general.Dependencies{DB: db}, authRoutes)
	user_vehicles.RegisterRoutes(user_vehicles.Dependencies{DB: db}, authRoutes)
	NewShopsRouter(db, blobClient, env, authRoutes)
	equipment_services.RegisterRoutes(equipment_services.Dependencies{DB: db}, authRoutes)
	pmcs_sbs_progress.RegisterRoutes(pmcs_sbs_progress.Dependencies{DB: db}, authRoutes)
	item_comments.RegisterRoutes(item_comments.Dependencies{DB: db}, v1Route, authRoutes)
	user_suggestions.RegisterRoutes(user_suggestions.Dependencies{
		DB:         db,
		AuthClient: authClient,
	}, v1Route, authRoutes)

	// Mixed Routes (both public and authenticated endpoints)
	material_images.RegisterRoutes(material_images.Dependencies{
		DB:         db,
		BlobClient: blobClient,
		Env:        env,
		AuthClient: authClient,
	}, v1Route, authRoutes)
	analyticsService := analytics.New(db)
	library.RegisterRoutes(library.Dependencies{
		DB:         db,
		BlobClient: blobClient,
		Env:        env,
		Analytics:  analyticsService,
	}, v1Route, authRoutes)

	// Serve static assets (CSS, JS, images, etc.)
	router.Static("/_app", "./static/_app")
	router.Static("/assets", "./static/assets")
	router.StaticFile("/favicon.ico", "./static/favicon.ico")
	router.StaticFile("/favicon.svg", "./static/favicon.svg")
	router.StaticFile("/app-ads.txt", "./static/app-ads.txt")

	// Explicitly serve the frontend at root path
	router.GET("/", func(c *gin.Context) {
		c.File("./static/index.html")
	})

	// SPA fallback - serve index.html for all other non-API routes
	router.NoRoute(func(c *gin.Context) {
		// Don't serve the SPA for API routes
		if strings.HasPrefix(c.Request.URL.Path, "/api") {
			c.JSON(http.StatusNotFound, gin.H{"error": "API route not found"})
			return
		}
		// Serve the SPA for all other routes
		c.File("./static/index.html")
	})
}
