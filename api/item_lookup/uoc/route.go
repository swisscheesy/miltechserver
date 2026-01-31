package uoc

import (
	"strconv"

	"miltechserver/api/item_lookup/shared"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(router *gin.RouterGroup, service Service) {
	router.GET("/lookup/uoc", func(c *gin.Context) {
		pageStr := c.Query("page")
		page, err := strconv.Atoi(pageStr)
		if err != nil {
			shared.HandleError(c, shared.ErrInvalidPage)
			return
		}

		uocData, err := service.LookupByPage(page)
		if err != nil {
			shared.HandleError(c, err)
			return
		}

		shared.WriteSuccess(c, uocData)
	})

	router.GET("/lookup/uoc/:uoc", func(c *gin.Context) {
		uoc := c.Param("uoc")

		uocData, err := service.LookupSpecific(uoc)
		if err != nil {
			shared.HandleError(c, err)
			return
		}

		shared.WriteSuccess(c, uocData)
	})

	router.GET("/lookup/uoc/by-model/:model", func(c *gin.Context) {
		model := c.Param("model")

		uocData, err := service.LookupByModel(model)
		if err != nil {
			shared.HandleError(c, err)
			return
		}

		shared.WriteSuccess(c, uocData)
	})
	// Legacy route for backward compatibility.
	router.GET("/lookup/uoc/model/:model", func(c *gin.Context) {
		model := c.Param("model")

		uocData, err := service.LookupByModel(model)
		if err != nil {
			shared.HandleError(c, err)
			return
		}

		shared.WriteSuccess(c, uocData)
	})
}
