package route

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
)

func NewTestRouter(db *sql.DB, group *gin.RouterGroup) {

	group.GET("/", func(c *gin.Context) {
		user, ok := c.Get("user")
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "You have access to this route", "user": user})
	})

	group.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "Hello World",
		})
	})
}
