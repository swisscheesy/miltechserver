package controller

import (
	"miltechserver/api/response"
	"miltechserver/api/service"

	"github.com/gin-gonic/gin"
)

type QuickListsController struct {
	QuickListsService service.ItemQuickListsService
}

func NewQuickListsController(quickListsService service.ItemQuickListsService) *QuickListsController {
	return &QuickListsController{QuickListsService: quickListsService}
}

func (controller *QuickListsController) QueryQuickListClothing(c *gin.Context) {
	clothingData, err := controller.QuickListsService.GetQuickListClothing()

	if err != nil {
		c.JSON(500, response.InternalErrorResponseMessage())
	} else {
		c.JSON(200, response.StandardResponse{
			Status:  200,
			Message: "",
			Data:    clothingData,
		})
	}
}

func (controller *QuickListsController) QueryQuickListWheels(c *gin.Context) {
	wheelsData, err := controller.QuickListsService.GetQuickListWheels()

	if err != nil {
		c.JSON(500, response.InternalErrorResponseMessage())
	} else {
		c.JSON(200, response.StandardResponse{
			Status:  200,
			Message: "",
			Data:    wheelsData,
		})
	}
}

func (controller *QuickListsController) QueryQuickListBatteries(c *gin.Context) {
	batteriesData, err := controller.QuickListsService.GetQuickListBatteries()

	if err != nil {
		c.JSON(500, response.InternalErrorResponseMessage())
	} else {
		c.JSON(200, response.StandardResponse{
			Status:  200,
			Message: "",
			Data:    batteriesData,
		})
	}
}
