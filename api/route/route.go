package route

import (
	"github.com/gin-gonic/gin"
	"miltechserver/api/middleware"
	"miltechserver/prisma/db"
)

func Setup(db *db.PrismaClient, gin *gin.Engine) {
	v1Route := gin.Group("/api/v1")
	v1Route.Use(middleware.ErrorHandler)
	//v1Route.Use(middleware.LoggerMiddleware())
	// All Public Routes
	NewItemQueryRouter(db, v1Route)
}
