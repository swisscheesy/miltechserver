package route

import (
	"database/sql"
	"miltechserver/api/middleware"
	"miltechserver/bootstrap"
	"net/http"
	"strings"

	"firebase.google.com/go/v4/auth"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/gin-gonic/gin"
)

func Setup(db *sql.DB, router *gin.Engine, authClient *auth.Client, env *bootstrap.Env, blobClient *azblob.Client) {
	v1Route := router.Group("/api/v1")
	v1Route.Use(middleware.ErrorHandler)

	testRoutes := router.Group("/api/v1/test")
	testRoutes.Use(middleware.AuthenticationMiddleware(authClient))
	NewTestRouter(db, testRoutes)

	v1Route.Use(middleware.LoggerMiddleware())
	// All Public Routes
	NewGeneralQueriesRouter(v1Route, env)
	NewItemQueryRouter(db, v1Route)
	NewItemLookupRouter(db, v1Route)
	NewItemQuickListsRouter(db, v1Route)
	NewEICRouter(db, v1Route)

	// All Authenticated Routes
	authRoutes := router.Group("/api/v1/auth")
	authRoutes.Use(middleware.AuthenticationMiddleware(authClient))
	NewUserSavesRouter(db, blobClient, env, authRoutes)
	NewUserGeneralRouter(db, authRoutes)
	NewUserVehicleRouter(db, authRoutes)
	NewShopsRouter(db, env, authRoutes)
	NewEquipmentServicesRouter(db, env, authRoutes)

	// Mixed Routes
	NewMaterialImagesRouter(db, blobClient, env, authClient, v1Route, authRoutes)

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
