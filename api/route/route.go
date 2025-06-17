package route

import (
	"database/sql"
	"miltechserver/api/middleware"
	"miltechserver/bootstrap"

	"firebase.google.com/go/v4/auth"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/gin-gonic/gin"
)

func Setup(db *sql.DB, gin *gin.Engine, authClient *auth.Client, env *bootstrap.Env, blobClient *azblob.Client) {
	v1Route := gin.Group("/api/v1")
	v1Route.Use(middleware.ErrorHandler)

	testRoutes := gin.Group("/api/v1/test")
	testRoutes.Use(middleware.AuthenticationMiddleware(authClient))
	NewTestRouter(db, testRoutes)

	v1Route.Use(middleware.LoggerMiddleware())
	// All Public Routes
	NewGeneralQueriesRouter(v1Route, env)
	NewItemQueryRouter(db, v1Route)
	NewItemLookupRouter(db, v1Route)
	NewItemQuickListsRouter(db, v1Route)

	// All Authenticated Routes
	authRoutes := gin.Group("/api/v1/auth")
	authRoutes.Use(middleware.AuthenticationMiddleware(authClient))
	NewUserSavesRouter(db, blobClient, env, authRoutes)
	NewUserGeneralRouter(db, authRoutes)
	NewUserVehicleRouter(db, authRoutes)
}
