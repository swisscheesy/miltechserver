package middleware

import (
	"github.com/gin-gonic/gin"
	"log"
	"miltechserver/api/response"
	"strings"
)

// TODO: Don't think this really does anything anymore.
func ErrorHandler(c *gin.Context) {
	c.Next()

	for _, err := range c.Errors {
		log.Default().Println(err)
		if strings.Contains(err.Error(), "no item found") {
			c.JSON(404, response.StandardResponse{
				Status:  404,
				Message: err.Error(),
				Data:    nil,
			})
			return
		} else {
			c.JSON(500, response.StandardResponse{
				Status:  500,
				Message: err.Error(),
				Data:    nil,
			})
		}
	}
}
