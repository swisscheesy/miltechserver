package route

import (
	"database/sql"
	"miltechserver/api/analytics"
	"miltechserver/api/eic"
	"miltechserver/api/equipment_services"
	"miltechserver/api/item_comments"
	"miltechserver/api/item_lookup"
	"miltechserver/api/item_query"
	"miltechserver/api/library"
	"miltechserver/api/material_images"
	"miltechserver/api/middleware"
	"miltechserver/api/quick_lists"
	"miltechserver/api/user_general"
	"miltechserver/api/user_saves"
	"miltechserver/api/user_vehicles"
	"miltechserver/bootstrap"
	"net/http"
	"strings"

	"firebase.google.com/go/v4/auth"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/gin-gonic/gin"
)

func Setup(db *sql.DB, router *gin.Engine, authClient *auth.Client, env *bootstrap.Env, blobClient *azblob.Client, blobCredential *azblob.SharedKeyCredential) {
	v1Route := router.Group("/api/v1")
	v1Route.Use(middleware.ErrorHandler)

	testRoutes := router.Group("/api/v1/test")
	testRoutes.Use(middleware.AuthenticationMiddleware(authClient))
	NewTestRouter(db, testRoutes)

	v1Route.Use(middleware.LoggerMiddleware())
	// All Public Routes
	NewGeneralRouter(v1Route, env)
	NewGeneralQueriesRouter(v1Route, env)
	item_query.RegisterRoutes(item_query.Dependencies{DB: db}, v1Route)
	item_lookup.RegisterRoutes(item_lookup.Dependencies{DB: db}, v1Route)
	quick_lists.RegisterRoutes(quick_lists.Dependencies{DB: db}, v1Route)
	eic.RegisterRoutes(eic.Dependencies{DB: db}, v1Route)

	// All Authenticated Routes
	authRoutes := router.Group("/api/v1/auth")
	authRoutes.Use(middleware.AuthenticationMiddleware(authClient))
	user_saves.RegisterRoutes(user_saves.Dependencies{
		DB:         db,
		BlobClient: blobClient,
		Env:        env,
	}, authRoutes)
	user_general.RegisterRoutes(user_general.Dependencies{DB: db}, authRoutes)
	user_vehicles.RegisterRoutes(user_vehicles.Dependencies{DB: db}, authRoutes)
	NewShopsRouter(db, blobClient, env, authRoutes)
	equipment_services.RegisterRoutes(equipment_services.Dependencies{DB: db}, authRoutes)
	item_comments.RegisterRoutes(item_comments.Dependencies{DB: db}, v1Route, authRoutes)

	// Mixed Routes (both public and authenticated endpoints)
	material_images.RegisterRoutes(material_images.Dependencies{
		DB:         db,
		BlobClient: blobClient,
		Env:        env,
		AuthClient: authClient,
	}, v1Route, authRoutes)
	analyticsService := analytics.New(db)
	library.RegisterRoutes(library.Dependencies{
		DB:             db,
		BlobClient:     blobClient,
		BlobCredential: blobCredential,
		Env:            env,
		Analytics:      analyticsService,
	}, v1Route, authRoutes)

	// Serve static assets (CSS, JS, images, etc.)
	router.Static("/_app", "./static/_app")
	router.Static("/assets", "./static/assets")
	router.StaticFile("/favicon.ico", "./static/favicon.ico")
	router.StaticFile("/favicon.svg", "./static/favicon.svg")

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
