package controller

import (
	"log/slog"
	"miltechserver/api/request"
	"miltechserver/api/response"
	"miltechserver/bootstrap"

	"github.com/gin-gonic/gin"
)

// Shop Invite Code Operations

// GenerateInviteCode creates a new invite code for a shop
func (controller *ShopsController) GenerateInviteCode(c *gin.Context) {
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

	code, err := controller.ShopsService.GenerateInviteCode(user, req.ShopID)
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
func (controller *ShopsController) GetInviteCodesByShop(c *gin.Context) {
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

	codes, err := controller.ShopsService.GetInviteCodesByShop(user, shopID)
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
func (controller *ShopsController) DeactivateInviteCode(c *gin.Context) {
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

	err := controller.ShopsService.DeactivateInviteCode(user, codeID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(200, gin.H{"message": "Invite code deactivated successfully"})
}

// DeleteInviteCode permanently deletes an invite code
func (controller *ShopsController) DeleteInviteCode(c *gin.Context) {
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

	err := controller.ShopsService.DeleteInviteCode(user, codeID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(200, gin.H{"message": "Invite code deleted successfully"})
}
