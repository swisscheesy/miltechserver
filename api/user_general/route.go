package user_general

import (
	"database/sql"
	"errors"
	"log/slog"
	"strings"

	"github.com/gin-gonic/gin"

	"miltechserver/api/auth"
	"miltechserver/api/response"
	"miltechserver/api/user_pmcs/persistence"
	"miltechserver/bootstrap"
)

type Dependencies struct {
	DB *sql.DB
}

type Handler struct {
	service Service
}

func RegisterRoutes(deps Dependencies, router *gin.RouterGroup) {
	repo := NewRepository(deps.DB, persistence.NewAccountCleaner())
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
		response.Error(c, 400, "invalid request body")
		slog.Info("Invalid request body", "error", err)
		return
	}

	if !ok {
		response.Error(c, 401, "unauthorized")
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
	value, exists := c.Get("user")
	currentUser, ok := value.(*bootstrap.User)
	if !exists || !ok || currentUser == nil ||
		strings.TrimSpace(currentUser.UserID) == "" {
		response.Error(c, 401, "unauthorized")
		slog.Info("Unauthorized request")
		return
	}

	if err := handler.service.DeleteUser(
		c.Request.Context(),
		currentUser.UserID,
	); err != nil {
		if errors.Is(err, ErrUserNotFound) {
			response.Error(c, 404, "user not found")
			slog.Info("User not found", "uid", currentUser.UserID)
			return
		}
		c.Error(err)
		return
	}

	response.OK(c, gin.H{"message": "user deleted successfully"})
	slog.Info("User deleted successfully", "uid", currentUser.UserID)
}

func (handler *Handler) updateUserDisplayName(c *gin.Context) {
	var displayNameRequest DisplayNameChangeRequest
	if err := c.ShouldBindJSON(&displayNameRequest); err != nil {
		response.Error(c, 404, "invalid request")
		slog.Info("Invalid request body", "error", err)
		return
	}

	if err := handler.service.UpdateUserDisplayName(displayNameRequest.UID, displayNameRequest.DisplayName); err != nil {
		if errors.Is(err, ErrUserNotFound) {
			slog.Info("User not found", "uid", displayNameRequest.UID)
		}
		response.Error(c, 404, "failed to update display name")
		slog.Info("Failed to update display name", "uid", displayNameRequest.UID, "error", err)
		return
	}

	c.Status(200)
	slog.Info("Display name updated successfully", "uid", displayNameRequest.UID, "display_name", displayNameRequest.DisplayName)
}
