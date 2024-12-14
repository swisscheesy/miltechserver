package controller

import (
	"github.com/gin-gonic/gin"
	"miltechserver/api/service"
)

type UserSavesController struct {
	UserSavesService service.UserSavesService
}

func NewUserSavesController(userSavesService service.UserSavesService) *UserSavesController {
	return &UserSavesController{UserSavesService: userSavesService}
}

func (controller *UserSavesController) GetQuickSaveItemsByUser(c *gin.Context) {
	user, ok := c.Get("user")

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		return
	}

	userId := user.(string)
	result, err := controller.UserSavesService.GetQuickSaveItemsByUser(c, userId)

	if err != nil {
		c.Error(err)
	} else {
		c.JSON(200, result)
	}
}
