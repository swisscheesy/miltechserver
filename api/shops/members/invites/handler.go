package invites

import (
	"log/slog"
	"miltechserver/api/request"
	"miltechserver/api/response"
	"miltechserver/bootstrap"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service Service
}

// Shop Invite Code Operations

// GenerateInviteCode creates a new invite code for a shop
func (handler *Handler) GenerateInviteCode(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	var req request.GenerateInviteCodeRequest
	if err := c.BindJSON(&req); err != nil {
		slog.Info("invalid request", "error", err)
		c.JSON(400, gin.H{"message": "invalid request"})
		return
	}

	service := handler.service
	code, err := service.GenerateInviteCode(user, req.ShopID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(201, response.StandardResponse{
		Status:  201,
		Message: "Invite code generated successfully",
		Data:    *code,
	})
}

// GetInviteCodesByShop returns all invite codes for a shop
func (handler *Handler) GetInviteCodesByShop(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	shopID := c.Param("shop_id")
	if shopID == "" {
		c.JSON(400, gin.H{"message": "shop_id is required"})
		return
	}

	service := handler.service
	codes, err := service.GetInviteCodesByShop(user, shopID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(200, response.StandardResponse{
		Status:  200,
		Message: "",
		Data:    codes,
	})
}

// DeactivateInviteCode deactivates an invite code
func (handler *Handler) DeactivateInviteCode(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	codeID := c.Param("code_id")
	if codeID == "" {
		c.JSON(400, gin.H{"message": "code_id is required"})
		return
	}

	service := handler.service
	err := service.DeactivateInviteCode(user, codeID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(200, gin.H{"message": "Invite code deactivated successfully"})
}

// DeleteInviteCode permanently deletes an invite code
func (handler *Handler) DeleteInviteCode(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	codeID := c.Param("code_id")
	if codeID == "" {
		c.JSON(400, gin.H{"message": "code_id is required"})
		return
	}

	service := handler.service
	err := service.DeleteInviteCode(user, codeID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(200, gin.H{"message": "Invite code deleted successfully"})
}
