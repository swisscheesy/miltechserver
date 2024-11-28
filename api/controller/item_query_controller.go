package controller

import (
	"github.com/gin-gonic/gin"
	"miltechserver/service"
)

type ItemQueryController struct {
	ItemQueryService service.ItemQueryService
}

func NewItemQueryController(itemQueryService service.ItemQueryService) *ItemQueryController {
	return &ItemQueryController{ItemQueryService: itemQueryService}
}

func (controller *ItemQueryController) FindShortByNiin(c *gin.Context) {
	niin := c.GetString("niin")
	result, err := controller.ItemQueryService.FindShortByNiin(c, niin)

	if err != nil {

	}
	//webResponse := response.StandardResponse{
	//	Code:    200,
	//	Data:    result,
	//	Message: "Ok",
	//}

	c.JSON(200, result)

}
