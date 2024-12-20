package route

import (
	"database/sql"
	"firebase.google.com/go/v4/auth"
	"github.com/gin-gonic/gin"
	"miltechserver/api/middleware"
)

func Setup(db *sql.DB, gin *gin.Engine, authClient *auth.Client) {
	v1Route := gin.Group("/api/v1")
	v1Route.Use(middleware.ErrorHandler)

	testRoutes := gin.Group("/api/v1/test")
	testRoutes.Use(middleware.AuthenticationMiddleware(authClient))
	NewTestRouter(db, testRoutes)

	//v1Route.Use(middleware.LoggerMiddleware())
	// All Public Routes
	NewItemQueryRouter(db, v1Route)
	NewItemLookupRouter(db, v1Route)

	// All Authenticated Routes
	authRoutes := gin.Group("/api/v1/auth")
	authRoutes.Use(middleware.AuthenticationMiddleware(authClient))
	//NewUserSavesRouter(db, authRoutes)
}
