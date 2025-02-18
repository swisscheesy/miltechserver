package controller

import (
	"github.com/gin-gonic/gin"
	"miltechserver/api/response"
	"miltechserver/api/service"
	"strings"
)

type ItemQueryController struct {
	ItemQueryService    service.ItemShortService
	ItemDetailedService service.ItemDetailedService
}

func NewItemQueryController(itemQueryService service.ItemShortService, itemDetailedService service.ItemDetailedService) *ItemQueryController {
	return &ItemQueryController{ItemQueryService: itemQueryService, ItemDetailedService: itemDetailedService}
}

// FindShort handles the request to find a short item by method and value.
// \param c - the Gin context for the request.
func (controller *ItemQueryController) FindShort(c *gin.Context) {
	method := c.Query("method")
	value := c.Query("value")

	switch method {
	case "niin":
		result, err := controller.ItemQueryService.FindShortByNiin(value)
		if err != nil {
			if strings.Contains(err.Error(), "no item") {
				c.JSON(404, response.NoItemFoundResponseMessage())
			} else {
				c.JSON(500, response.InternalErrorResponseMessage())
			}
		} else {
			c.JSON(200, response.StandardResponse{
				Status:  200,
				Message: "",
				Data:    result,
			})
		}
	case "part":
		result, err := controller.ItemQueryService.FindShortByPart(value)
		if err != nil {
			if strings.Contains(err.Error(), "no item") {
				c.JSON(404, response.NoItemFoundResponseMessage())
			} else {
				c.JSON(500, response.InternalErrorResponseMessage())
			}
		} else {

			c.JSON(200, response.StandardResponse{
				Status:  200,
				Message: "",
				Data:    result,
			})
		}
	}
}

// FindDetailed handles the request to find a detailed item by NIIN.
// \param c - the Gin context for the request.
func (controller *ItemQueryController) FindDetailed(c *gin.Context) {
	niin := c.Query("niin")
	itemData, err := controller.ItemDetailedService.FindDetailedItem(niin)

	if err != nil {
		c.Error(err)
	} else {
		c.JSON(200, response.StandardResponse{
			Status:  200,
			Message: "",
			Data:    itemData,
		})
	}

}
