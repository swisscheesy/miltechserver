package substitute

import (
	"miltechserver/api/item_lookup/shared"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(router *gin.RouterGroup, service Service) {
	router.GET("/lookup/substitute-lin", func(c *gin.Context) {
		substituteData, err := service.LookupAll()
		if err != nil {
			shared.HandleError(c, err)
			return
		}

		shared.WriteSuccess(c, substituteData)
	})
}
