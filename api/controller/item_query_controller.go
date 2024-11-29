package controller

import (
	"github.com/gin-gonic/gin"
	"miltechserver/model/response"
	"miltechserver/service"
)

type ItemQueryController struct {
	ItemQueryService service.ItemQueryService
}

func NewItemQueryController(itemQueryService service.ItemQueryService) *ItemQueryController {
	return &ItemQueryController{ItemQueryService: itemQueryService}
}

func (controller *ItemQueryController) FindShort(c *gin.Context) {
	method := c.Query("method")
	value := c.Query("value")

	if method == "niin" {
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

	} else if method == "part" {
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
