package route

import (
	"github.com/gin-gonic/gin"
	"miltechserver/api/middleware"
	"miltechserver/prisma/db"
)

func Setup(db *db.PrismaClient, gin *gin.Engine) {
	publicRouter := gin.Group("")
	publicRouter.Use(middleware.ErrorHandler)
	//publicRouter.Use(middleware.LoggerMiddleware())
	// All Public Routes
	NewItemQueryRouter(db, publicRouter)
}
