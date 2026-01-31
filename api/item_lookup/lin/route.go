package lin

import (
	"strconv"

	"miltechserver/api/item_lookup/shared"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(router *gin.RouterGroup, service Service) {
	router.GET("/lookup/lin", func(c *gin.Context) {
		pageStr := c.Query("page")
		page, err := strconv.Atoi(pageStr)
		if err != nil {
			shared.HandleError(c, shared.ErrInvalidPage)
			return
		}

		linData, err := service.LookupByPage(page)
		if err != nil {
			shared.HandleError(c, err)
			return
		}

		shared.WriteSuccess(c, linData)
	})

	router.GET("/lookup/lin/by-niin/:niin", func(c *gin.Context) {
		niin := c.Param("niin")

		linData, err := service.LookupByNIIN(niin)
		if err != nil {
			shared.HandleError(c, err)
			return
		}

		shared.WriteSuccess(c, linData)
	})
	// Legacy route for backward compatibility.
	router.GET("/lookup/lin/lin/:niin", func(c *gin.Context) {
		niin := c.Param("niin")

		linData, err := service.LookupByNIIN(niin)
		if err != nil {
			shared.HandleError(c, err)
			return
		}

		shared.WriteSuccess(c, linData)
	})

	router.GET("/lookup/niin/by-lin/:lin", func(c *gin.Context) {
		lin := c.Param("lin")

		niinData, err := service.LookupNIINByLIN(lin)
		if err != nil {
			shared.HandleError(c, err)
			return
		}

		shared.WriteSuccess(c, niinData)
	})
	// Legacy route for backward compatibility.
	router.GET("/lookup/lin/niin/:lin", func(c *gin.Context) {
		lin := c.Param("lin")

		niinData, err := service.LookupNIINByLIN(lin)
		if err != nil {
			shared.HandleError(c, err)
			return
		}

		shared.WriteSuccess(c, niinData)
	})
}
