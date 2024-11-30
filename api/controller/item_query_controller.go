package controller

import (
	"github.com/gin-gonic/gin"
	"miltechserver/api/service"
	"miltechserver/model"
	"miltechserver/model/response"
)

type ItemQueryController struct {
	ItemQueryService service.ItemShortService
}

func NewItemQueryController(itemQueryService service.ItemShortService) *ItemQueryController {
	return &ItemQueryController{ItemQueryService: itemQueryService}
}

func (controller *ItemQueryController) FindShort(c *gin.Context) {
	method := c.Query("method")
	value := c.Query("value")

	switch method {
	case "niin":
		result, err := controller.ItemQueryService.FindShortByNiin(c, value)
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
		result, err := controller.ItemQueryService.FindShortByPart(c, value)
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
	result, err := controller.ItemQueryService.FindShortByNiin(c, niin)

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
	niin := c.Query("niin")
	_, err := controller.ItemQueryService.FindAmdfData(c, niin)

	if err != nil {
		c.Error(err)
	} else {
		amdf := model.DetailedItem{}
		c.JSON(200, response.StandardResponse{
			Status:  200,
			Message: "",
			Data:    amdf,
		})
	}

}

func (controller *ItemQueryController) FindShortByPart(c *gin.Context, part string) {
	result, err := controller.ItemQueryService.FindShortByPart(c, part)

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
