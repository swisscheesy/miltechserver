package cage

import (
	"miltechserver/api/item_lookup/shared"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(router *gin.RouterGroup, service Service) {
	router.GET("/lookup/cage/:cage", func(c *gin.Context) {
		cage := c.Param("cage")

		cageData, err := service.LookupByCode(cage)
		if err != nil {
			shared.HandleError(c, err)
			return
		}

		shared.WriteSuccess(c, cageData)
	})
}
