package controller

import (
	"github.com/gin-gonic/gin"
	"miltechserver/api/response"
	"miltechserver/api/service"
)

type ItemQueryController struct {
	ItemQueryService    service.ItemShortService
	ItemDetailedService service.ItemDetailedService
}

func NewItemQueryController(itemQueryService service.ItemShortService, itemDetailedService service.ItemDetailedService) *ItemQueryController {
	return &ItemQueryController{ItemQueryService: itemQueryService, ItemDetailedService: itemDetailedService}
}

func (controller *ItemQueryController) FindShort(c *gin.Context) {
	method := c.Query("method")
	value := c.Query("value")

	switch method {
	case "niin":
		result, err := controller.ItemQueryService.FindShortByNiin(value)
		if err != nil {
			c.Error(err)
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
			c.Error(err)
		} else {
			c.JSON(200, response.StandardResponse{
				Status:  200,
				Message: "",
				Data:    result,
			})
		}
	}
}

func (controller *ItemQueryController) FindShortByNiin(c *gin.Context, niin string) {
	result, err := controller.ItemQueryService.FindShortByNiin(niin)

	if err != nil {
		c.Error(err)
	} else {
		c.JSON(200, response.StandardResponse{
			Status:  200,
			Message: "",
			Data:    result,
		})
	}

}

func (controller *ItemQueryController) FindDetailed(c *gin.Context) {
	//niin := c.Query("niin")
	//itemData, err := controller.ItemDetailedService.FindDetailedItem(c, niin)
	//
	//if err != nil {
	//	c.Error(err)
	//} else {
	//	c.JSON(200, response.StandardResponse{
	//		Status:  200,
	//		Message: "",
	//		Data:    itemData,
	//	})
	//}

}

func (controller *ItemQueryController) FindDetailedTest(c *gin.Context) {
	niin := c.Query("niin")
	itemData, err := controller.ItemDetailedService.GetDetailedItemTest(niin)

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

func (controller *ItemQueryController) FindShortByPart(c *gin.Context, part string) {
	//result, err := controller.ItemQueryService.FindShortByPart(c, part)
	//
	//if err != nil {
	//	c.Error(err)
	//} else {
	//	c.JSON(200, response.StandardResponse{
	//		Status:  200,
	//		Message: "",
	//		Data:    result,
	//	})
	//}
}
