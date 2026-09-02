package members

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

// Shop Member Operations

// JoinShopViaInviteCode allows a user to join a shop using an invite code
func (handler *Handler) JoinShopViaInviteCode(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		response.Error(c, 401, "unauthorized")
		slog.Info("Unauthorized request")
		return
	}

	var req request.JoinShopRequest
	if err := c.BindJSON(&req); err != nil {
		slog.Info("invalid request", "error", err)
		response.Error(c, 400, "invalid request")
		return
	}

	service := handler.service
	err := service.JoinShopViaInviteCode(c.Request.Context(), user, req.InviteCode)
	if err != nil {
		c.Error(err)
		return
	}

	response.OK(c, gin.H{"message": "Successfully joined shop"})
}

// LeaveShop allows a user to leave a shop
func (handler *Handler) LeaveShop(c *gin.Context) {
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
	err := service.LeaveShop(c.Request.Context(), user, shopID)
	if err != nil {
		c.Error(err)
		return
	}

	response.OK(c, gin.H{"message": "Successfully left shop"})
}

// RemoveMemberFromShop allows admins to remove members from a shop
func (handler *Handler) RemoveMemberFromShop(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		response.Error(c, 401, "unauthorized")
		slog.Info("Unauthorized request")
		return
	}

	var req request.RemoveMemberRequest
	if err := c.BindJSON(&req); err != nil {
		slog.Info("invalid request", "error", err)
		response.Error(c, 400, "invalid request")
		return
	}

	service := handler.service
	err := service.RemoveMemberFromShop(c.Request.Context(), user, req.ShopID, req.TargetUserID)
	if err != nil {
		c.Error(err)
		return
	}

	response.OK(c, gin.H{"message": "Member removed successfully"})
}

// PromoteMemberToAdmin allows admins to promote members to admin role
func (handler *Handler) PromoteMemberToAdmin(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		response.Error(c, 401, "unauthorized")
		slog.Info("Unauthorized request")
		return
	}

	var req request.PromoteMemberRequest
	if err := c.BindJSON(&req); err != nil {
		slog.Info("invalid request", "error", err)
		response.Error(c, 400, "invalid request")
		return
	}

	service := handler.service
	err := service.PromoteMemberToAdmin(c.Request.Context(), user, req.ShopID, req.TargetUserID)
	if err != nil {
		c.Error(err)
		return
	}

	response.OK(c, gin.H{"message": "Member promoted to admin successfully"})
}

// GetShopMembers returns all members of a shop
func (handler *Handler) GetShopMembers(c *gin.Context) {
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
	members, err := service.GetShopMembers(c.Request.Context(), user, shopID)
	if err != nil {
		c.Error(err)
		return
	}

	response.OK(c, members)
}
