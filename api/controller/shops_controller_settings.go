package controller

import (
	"log/slog"
	"miltechserver/api/request"
	"miltechserver/api/response"
	"miltechserver/bootstrap"

	"github.com/gin-gonic/gin"
)

// Shop Settings Operations

// GetShopAdminOnlyListsSetting returns the admin_only_lists setting for a shop
func (controller *ShopsController) GetShopAdminOnlyListsSetting(c *gin.Context) {
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

	adminOnlyLists, err := controller.ShopsService.GetShopAdminOnlyListsSetting(user, shopID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(200, response.StandardResponse{
		Status:  200,
		Message: "Shop admin_only_lists setting retrieved successfully",
		Data: gin.H{
			"shop_id":          shopID,
			"admin_only_lists": adminOnlyLists,
		},
	})
}

// UpdateShopAdminOnlyListsSetting updates the admin_only_lists setting for a shop (admin only)
func (controller *ShopsController) UpdateShopAdminOnlyListsSetting(c *gin.Context) {
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

	var req request.UpdateAdminOnlyListsRequest
	if err := c.BindJSON(&req); err != nil {
		slog.Info("invalid request", "error", err)
		c.JSON(400, gin.H{"message": "invalid request"})
		return
	}

	err := controller.ShopsService.UpdateShopAdminOnlyListsSetting(user, shopID, req.AdminOnlyLists)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(200, response.StandardResponse{
		Status:  200,
		Message: "Shop admin_only_lists setting updated successfully",
		Data: gin.H{
			"shop_id":          shopID,
			"admin_only_lists": req.AdminOnlyLists,
		},
	})
}

// CheckUserIsShopAdmin checks if the current user is an admin for the specified shop
func (controller *ShopsController) CheckUserIsShopAdmin(c *gin.Context) {
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

	isAdmin, err := controller.ShopsService.IsUserShopAdmin(user, shopID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(200, response.StandardResponse{
		Status:  200,
		Message: "Admin status checked successfully",
		Data: gin.H{
			"shop_id":  shopID,
			"user_id":  user.UserID,
			"is_admin": isAdmin,
		},
	})
}

// Unified Shop Settings Operations

// GetShopSettings returns all settings for a shop
func (controller *ShopsController) GetShopSettings(c *gin.Context) {
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

	settings, err := controller.ShopsService.GetShopSettings(user, shopID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(200, response.StandardResponse{
		Status:  200,
		Message: "Shop settings retrieved successfully",
		Data:    settings,
	})
}

// UpdateShopSettings updates one or more shop settings (admin only)
func (controller *ShopsController) UpdateShopSettings(c *gin.Context) {
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

	var req request.UpdateShopSettingsRequest
	if err := c.BindJSON(&req); err != nil {
		slog.Info("invalid request", "error", err)
		c.JSON(400, gin.H{"message": "invalid request"})
		return
	}

	// Validate that at least one setting is being updated
	if req.AdminOnlyLists == nil {
		// Future settings will be checked here with OR conditions
		c.JSON(400, gin.H{"message": "at least one setting must be provided"})
		return
	}

	updatedSettings, err := controller.ShopsService.UpdateShopSettings(user, shopID, req)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(200, response.StandardResponse{
		Status:  200,
		Message: "Shop settings updated successfully",
		Data:    updatedSettings,
	})
}
