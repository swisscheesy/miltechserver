package route

import (
	"net/http"

	"miltechserver/api/response"
	"miltechserver/bootstrap"

	"github.com/gin-gonic/gin"
)

func NewGeneralRouter(group *gin.RouterGroup, env *bootstrap.Env) {
	group.GET("/version", func(c *gin.Context) {
		// Kept as a raw StandardResponse literal: response.OK hardcodes an empty
		// Message and has no parameter to carry this success text.
		c.JSON(http.StatusOK, response.StandardResponse{
			Status:  http.StatusOK,
			Message: "Mobile app version",
			Data:    env.MobileAppVersion,
		})
	})
}
