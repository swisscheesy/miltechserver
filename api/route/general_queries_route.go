package route

import (
	"github.com/gin-gonic/gin"
	"miltechserver/api/response"
	"miltechserver/bootstrap"
)

func NewGeneralQueriesRouter(group *gin.RouterGroup, env *bootstrap.Env) {

	group.GET("/general/db_date", func(c *gin.Context) {
		c.JSON(200, response.StandardResponse{
			Status:  200,
			Message: "FedLog Database Date",
			Data:    env.DBDate,
		})
	})

}
