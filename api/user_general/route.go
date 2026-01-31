package user_general

import (
	"database/sql"
	"errors"
	"log/slog"

	"github.com/gin-gonic/gin"

	"miltechserver/api/auth"
	"miltechserver/bootstrap"
)

type Dependencies struct {
	DB *sql.DB
}

type Handler struct {
	service Service
}

func RegisterRoutes(deps Dependencies, router *gin.RouterGroup) {
	repo := NewRepository(deps.DB)
	svc := NewService(repo)
	registerHandlers(router, svc)
}

func registerHandlers(router *gin.RouterGroup, svc Service) {
	handler := Handler{service: svc}

	router.POST("/user/general/refresh", handler.upsertUser)
	router.DELETE("/user/general/delete_user", handler.deleteUser)
	router.POST("/user/general/dn_change", handler.updateUserDisplayName)
}

func (handler *Handler) upsertUser(c *gin.Context) {
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
		slog.Info("Unauthorized request")
		return
	}

	if err := handler.service.UpsertUser(user, userDto); err != nil {
		c.Error(err)
		return
	}

	c.Status(200)
}

func (handler *Handler) deleteUser(c *gin.Context) {
	_, ok := c.Get("user")
	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	var deleteRequest DeleteRequest
	if err := c.ShouldBindJSON(&deleteRequest); err != nil {
		c.JSON(400, gin.H{"message": "invalid request body"})
		slog.Info("Invalid request body", "error", err)
		return
	}

	if err := handler.service.DeleteUser(deleteRequest.UID); err != nil {
		if errors.Is(err, ErrUserNotFound) {
			c.JSON(404, gin.H{"message": "user not found"})
			slog.Info("User not found", "uid", deleteRequest.UID)
			return
		}
		c.Error(err)
		return
	}

	c.JSON(200, gin.H{"message": "user deleted successfully"})
	slog.Info("User deleted successfully", "uid", deleteRequest.UID)
}

func (handler *Handler) updateUserDisplayName(c *gin.Context) {
	var displayNameRequest DisplayNameChangeRequest
	if err := c.ShouldBindJSON(&displayNameRequest); err != nil {
		c.JSON(404, gin.H{"message": "invalid request"})
		slog.Info("Invalid request body", "error", err)
		return
	}

	if err := handler.service.UpdateUserDisplayName(displayNameRequest.UID, displayNameRequest.DisplayName); err != nil {
		if errors.Is(err, ErrUserNotFound) {
			slog.Info("User not found", "uid", displayNameRequest.UID)
		}
		c.JSON(404, gin.H{"message": "failed to update display name"})
		slog.Info("Failed to update display name", "uid", displayNameRequest.UID, "error", err)
		return
	}

	c.Status(200)
	slog.Info("Display name updated successfully", "uid", displayNameRequest.UID, "display_name", displayNameRequest.DisplayName)
}
