package route

import (
	"github.com/gin-gonic/gin"
	"miltechserver/bootstrap"
	"miltechserver/prisma/db"
	"time"
)

func Setup(env *bootstrap.Env, timeout time.Duration, db *db.PrismaClient, gin *gin.Engine) {
	publicRouter := gin.Group("")
	// All Public Routes
	NewDebugRouter(env, timeout, db, publicRouter)
	NewItemQueryRouter(db, publicRouter)
}
