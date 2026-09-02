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
		response.Error(c, 401, "unauthorized")
		slog.Info("Unauthorized request")
		return
	}

	var req request.GenerateInviteCodeRequest
	if err := c.BindJSON(&req); err != nil {
		slog.Info("invalid request", "error", err)
		response.Error(c, 400, "invalid request")
		return
	}

	service := handler.service
	code, err := service.GenerateInviteCode(c.Request.Context(), user, req.ShopID)
	if err != nil {
		c.Error(err)
		return
	}

	// Kept as a raw StandardResponse literal: response.OK hardcodes an empty
	// Message and has no parameter to carry this success text.
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
		response.Error(c, 401, "unauthorized")
		slog.Info("Unauthorized request")
		return
	}

	shopID := c.Param("shop_id")
	if shopID == "" {
		response.Error(c, 400, "shop_id is required")
		return
	}

	service := handler.service
	codes, err := service.GetInviteCodesByShop(c.Request.Context(), user, shopID)
	if err != nil {
		c.Error(err)
		return
	}

	response.OK(c, codes)
}

// DeactivateInviteCode deactivates an invite code
func (handler *Handler) DeactivateInviteCode(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		response.Error(c, 401, "unauthorized")
		slog.Info("Unauthorized request")
		return
	}

	codeID := c.Param("code_id")
	if codeID == "" {
		response.Error(c, 400, "code_id is required")
		return
	}

	service := handler.service
	err := service.DeactivateInviteCode(c.Request.Context(), user, codeID)
	if err != nil {
		c.Error(err)
		return
	}

	response.OK(c, gin.H{"message": "Invite code deactivated successfully"})
}

// DeleteInviteCode permanently deletes an invite code
func (handler *Handler) DeleteInviteCode(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		response.Error(c, 401, "unauthorized")
		slog.Info("Unauthorized request")
		return
	}

	codeID := c.Param("code_id")
	if codeID == "" {
		response.Error(c, 400, "code_id is required")
		return
	}

	service := handler.service
	err := service.DeleteInviteCode(c.Request.Context(), user, codeID)
	if err != nil {
		c.Error(err)
		return
	}

	response.OK(c, gin.H{"message": "Invite code deleted successfully"})
}
