package controller

import (
	"log/slog"
	"miltechserver/api/auth"
	"miltechserver/api/request"
	"miltechserver/api/service"
	"miltechserver/bootstrap"

	"github.com/gin-gonic/gin"
)

type UserGeneralController struct {
	UserGeneralService service.UserGeneralService
}

func NewUserGeneralController(userGeneralService service.UserGeneralService) *UserGeneralController {
	return &UserGeneralController{UserGeneralService: userGeneralService}
}

func (controller *UserGeneralController) UpsertUser(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)
	userDto := auth.UserDto{}

	if err := c.ShouldBindJSON(&userDto); err != nil {
		c.JSON(400, gin.H{"message": "invalid request body"})
		slog.Info("Invalid request body", "error", err)
		return
	}

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request %s")
		return
	} else {
		err := controller.UserGeneralService.UpsertUser(user, userDto)

		if err != nil {
			c.Error(err)
		} else {
			c.Status(200)
		}
	}
}

func (controller *UserGeneralController) DeleteUser(c *gin.Context) {
	_, ok := c.Get("user")
	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	var deleteRequest request.UserDeleteRequest
	if err := c.ShouldBindJSON(&deleteRequest); err != nil {
		c.JSON(400, gin.H{"message": "invalid request body"})
		slog.Info("Invalid request body", "error", err)
		return
	}

	err := controller.UserGeneralService.DeleteUser(deleteRequest.UID)
	if err != nil {
		if err.Error() == "user not found" {
			c.JSON(404, gin.H{"message": "user not found"})
			slog.Info("User not found", "uid", deleteRequest.UID)
		} else {
			c.Error(err)
		}
		return
	}

	c.JSON(200, gin.H{"message": "user deleted successfully"})
	slog.Info("User deleted successfully", "uid", deleteRequest.UID)
}
