package controller

import (
	"github.com/gin-gonic/gin"
	"log/slog"
	"miltechserver/api/service"
	"miltechserver/bootstrap"
)

type UserSavesController struct {
	UserSavesService service.UserSavesService
}

func NewUserSavesController(userSavesService service.UserSavesService) *UserSavesController {
	return &UserSavesController{UserSavesService: userSavesService}
}

func (controller *UserSavesController) GetQuickSaveItemsByUser(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request %s")
		return
	}

	result, err := controller.UserSavesService.GetQuickSaveItemsByUser(c, user)

	if err != nil {
		c.Error(err)
	} else {
		c.JSON(200, result)
	}
}
