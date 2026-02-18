package core

import (
	"log/slog"
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/request"
	"miltechserver/api/response"
	"miltechserver/bootstrap"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service ShopService
}

// Shop Operations

// CreateShop handles shop creation
func (handler *Handler) CreateShop(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	var req request.CreateShopRequest
	if err := c.BindJSON(&req); err != nil {
		slog.Info("invalid request", "error", err)
		c.JSON(400, gin.H{"message": "invalid request"})
		return
	}

	shop := model.Shops{
		Name:    req.Name,
		Details: req.Details,
	}

	// Set admin_only_lists if provided, otherwise defaults to false in database
	if req.AdminOnlyLists != nil {
		shop.AdminOnlyLists = *req.AdminOnlyLists
	}

	service := handler.service
	createdShop, err := service.CreateShop(user, shop)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(201, response.StandardResponse{
		Status:  201,
		Message: "Shop created successfully",
		Data:    *createdShop,
	})
}

// DeleteShop handles shop deletion
func (handler *Handler) DeleteShop(c *gin.Context) {
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
	err := service.DeleteShop(user, shopID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(200, gin.H{"message": "Shop deleted successfully"})
}

// GetUserShops returns all shops for the authenticated user
func (handler *Handler) GetUserShops(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	service := handler.service
	shops, err := service.GetShopsByUser(user)
	if err != nil {
		c.JSON(404, response.EmptyResponseMessage())
		return
	}

	c.JSON(200, response.StandardResponse{
		Status:  200,
		Message: "",
		Data:    shops,
	})
}

// GetUserDataWithShops returns user data along with all shops they are a part of
func (handler *Handler) GetUserDataWithShops(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	service := handler.service
	userShopsData, err := service.GetUserDataWithShops(user)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(200, response.StandardResponse{
		Status:  200,
		Message: "User data and shops retrieved successfully",
		Data:    *userShopsData,
	})
}

// GetShopByID returns a specific shop by ID
func (handler *Handler) GetShopByID(c *gin.Context) {
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
	shop, err := service.GetShopByID(user, shopID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(200, response.StandardResponse{
		Status:  200,
		Message: "",
		Data:    *shop,
	})
}

// UpdateShop handles shop updates
func (handler *Handler) UpdateShop(c *gin.Context) {
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

	var req request.UpdateShopRequest
	if err := c.BindJSON(&req); err != nil {
		slog.Info("invalid request", "error", err)
		c.JSON(400, gin.H{"message": "invalid request"})
		return
	}

	shop := model.Shops{
		ID:      shopID,
		Name:    req.Name,
		Details: req.Details,
	}

	service := handler.service
	updatedShop, err := service.UpdateShop(user, shop)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(200, response.StandardResponse{
		Status:  200,
		Message: "Shop updated successfully",
		Data:    *updatedShop,
	})
}
