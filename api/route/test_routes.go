package route

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"

	"miltechserver/api/response"
)

func NewTestRouter(db *sql.DB, group *gin.RouterGroup) {

	group.GET("/", func(c *gin.Context) {
		user, ok := c.Get("user")
		if !ok {
			response.Error(c, http.StatusUnauthorized, "unauthorized")
			return
		}
		// Kept as a raw gin.H{} response: flat multi-field body ("message" +
		// "user") that response.OK()'s single data field cannot represent
		// without nesting it under "data", changing this diagnostic route's
		// existing body shape.
		c.JSON(http.StatusOK, gin.H{"message": "You have access to this route", "user": user})
	})

	group.GET("/test", func(c *gin.Context) {
		// Kept as a raw gin.H{} response: a flat debug body, not the
		// response.OK()/response.Error() envelope shape; wrapping it would
		// nest it under "data", changing this diagnostic route's body.
		c.JSON(200, gin.H{
			"message": "Hello World",
		})
	})
}
