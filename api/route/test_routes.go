package route

import (
	"github.com/gin-gonic/gin"
	"miltechserver/prisma/db"
	"net/http"
)

func NewTestRouter(db *db.PrismaClient, group *gin.RouterGroup) {

	group.GET("/", func(c *gin.Context) {
		user, ok := c.Get("user")
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "You have access to this route", "user": user})
	})

	//router.GET("/item_query", func(c *gin.Context) {
	//	c.JSON(200, gin.H{
	//		"message": "Hello World",
	//	})
	//})
}
