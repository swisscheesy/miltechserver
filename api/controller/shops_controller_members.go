package controller

import (
	"log/slog"
	"miltechserver/api/request"
	"miltechserver/api/response"
	"miltechserver/bootstrap"

	"github.com/gin-gonic/gin"
)

// Shop Member Operations

// JoinShopViaInviteCode allows a user to join a shop using an invite code
func (controller *ShopsController) JoinShopViaInviteCode(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	var req request.JoinShopRequest
	if err := c.BindJSON(&req); err != nil {
		slog.Info("invalid request", "error", err)
		c.JSON(400, gin.H{"message": "invalid request"})
		return
	}

	service := controller.serviceForRequest(c)
	err := service.JoinShopViaInviteCode(user, req.InviteCode)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(200, gin.H{"message": "Successfully joined shop"})
}

// LeaveShop allows a user to leave a shop
func (controller *ShopsController) LeaveShop(c *gin.Context) {
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

	service := controller.serviceForRequest(c)
	err := service.LeaveShop(user, shopID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(200, gin.H{"message": "Successfully left shop"})
}

// RemoveMemberFromShop allows admins to remove members from a shop
func (controller *ShopsController) RemoveMemberFromShop(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	var req request.RemoveMemberRequest
	if err := c.BindJSON(&req); err != nil {
		slog.Info("invalid request", "error", err)
		c.JSON(400, gin.H{"message": "invalid request"})
		return
	}

	service := controller.serviceForRequest(c)
	err := service.RemoveMemberFromShop(user, req.ShopID, req.TargetUserID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(200, gin.H{"message": "Member removed successfully"})
}

// PromoteMemberToAdmin allows admins to promote members to admin role
func (controller *ShopsController) PromoteMemberToAdmin(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	var req request.PromoteMemberRequest
	if err := c.BindJSON(&req); err != nil {
		slog.Info("invalid request", "error", err)
		c.JSON(400, gin.H{"message": "invalid request"})
		return
	}

	service := controller.serviceForRequest(c)
	err := service.PromoteMemberToAdmin(user, req.ShopID, req.TargetUserID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(200, gin.H{"message": "Member promoted to admin successfully"})
}

// GetShopMembers returns all members of a shop
func (controller *ShopsController) GetShopMembers(c *gin.Context) {
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

	service := controller.serviceForRequest(c)
	members, err := service.GetShopMembers(user, shopID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(200, response.StandardResponse{
		Status:  200,
		Message: "",
		Data:    members,
	})
}
