package route

import (
	"firebase.google.com/go/v4/auth"
	"github.com/gin-gonic/gin"
	"miltechserver/api/middleware"
	"miltechserver/prisma/db"
)

func Setup(db *db.PrismaClient, gin *gin.Engine, authClient *auth.Client) {
	v1Route := gin.Group("/api/v1")
	v1Route.Use(middleware.ErrorHandler)

	testRoutes := gin.Group("/api/v1/test")
	testRoutes.Use(middleware.AuthenticationMiddleware(authClient))
	NewTestRouter(db, testRoutes)

	//v1Route.Use(middleware.LoggerMiddleware())
	// All Public Routes
	NewItemQueryRouter(db, v1Route)
	NewItemLookupRouter(db, v1Route)
}
