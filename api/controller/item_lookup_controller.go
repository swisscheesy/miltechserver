package controller

import (
	"github.com/gin-gonic/gin"
	"miltechserver/api/service"
	"miltechserver/model/response"
	"strconv"
)

type ItemLookupController struct {
	ItemLookupService service.ItemLookupService
}

func NewItemLookupController(itemLookupService service.ItemLookupService) *ItemLookupController {
	return &ItemLookupController{ItemLookupService: itemLookupService}
}

func (controller *ItemLookupController) LookupLINByPage(c *gin.Context) {
	pageStr := c.Query("page")
	page, err := strconv.Atoi(pageStr)
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid page number"})
		return
	}

	linData, err := controller.ItemLookupService.LookupLINByPage(c, page)

	c.JSON(200, response.StandardResponse{
		Status:  200,
		Message: "",
		Data:    linData,
	})
}

func (controller *ItemLookupController) LookupLINByNIIN(c *gin.Context) {
	niin := c.Param("niin")

	linData, err := controller.ItemLookupService.LookupLINByNIIN(c, niin)

	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	retCount := len(linData)

	c.JSON(200, response.StandardResponse{
		Status:  200,
		Message: "",
		Data: response.LinSearchResponse{
			Count: retCount,
			Lins:  linData,
		},
	})
}
