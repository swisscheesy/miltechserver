package middleware

import (
	"errors"
	"github.com/gin-gonic/gin"
	"log"
	"miltechserver/model/response"
	"miltechserver/prisma/db"
)

func ErrorHandler(c *gin.Context) {
	c.Next()

	for _, err := range c.Errors {
		log.Default().Println(err)
		switch {
		case errors.Is(err.Err, db.ErrNotFound):
			c.JSON(404, response.StandardResponse{
				Status:  404,
				Message: err.Error(),
				Data:    nil,
			})
		default:
			c.JSON(500, response.StandardResponse{
				Status:  500,
				Message: err.Error(),
				Data:    nil,
			})
		}
	}
}
