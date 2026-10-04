package core

import (
	"log/slog"
	"net/http"
	"time"

	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/request"
	"miltechserver/api/response"
	"miltechserver/api/shops/shared"
	"miltechserver/bootstrap"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service ShopService
}

func (handler *Handler) GetShopEquipmentOverview(c *gin.Context) {
	startedAt := time.Now()
	ctxUser, ok := c.Get("user")
	user, userOK := ctxUser.(*bootstrap.User)
	if !ok || !userOK || user == nil {
		response.Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	overview, err := handler.service.GetShopEquipmentOverview(c.Request.Context(), user)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, ErrShopEquipmentOverviewUnavailable.Error())
		return
	}

	equipmentCount := 0
	for i := range overview.Shops {
		equipmentCount += overview.Shops[i].EquipmentCount
	}

	// Kept as a raw StandardResponse literal: response.OK hardcodes an empty
	// Message and has no parameter to carry this success text.
	c.JSON(http.StatusOK, response.StandardResponse{
		Status:  http.StatusOK,
		Message: "Shop equipment overview retrieved successfully",
		Data:    overview,
	})
	slog.Info(
		"Shop equipment overview request completed",
		"user_id", user.UserID,
		"shop_count", len(overview.Shops),
		"equipment_count", equipmentCount,
		"duration_ms", time.Since(startedAt).Milliseconds(),
	)
}

// Shop Operations

// CreateShop handles shop creation
func (handler *Handler) CreateShop(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		response.Error(c, 401, "unauthorized")
		slog.Info("Unauthorized request")
		return
	}

	var req request.CreateShopRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		slog.Info("invalid request", "error", err)
		response.Error(c, 400, "invalid request")
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
	createdShop, err := service.CreateShop(c.Request.Context(), user, shop)
	if err != nil {
		c.Error(err)
		return
	}

	// Kept as a raw StandardResponse literal: response.OK hardcodes an empty
	// Message and has no parameter to carry this success text.
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
	err := service.DeleteShop(c.Request.Context(), user, shopID)
	if err != nil {
		c.Error(err)
		return
	}

	response.OK(c, gin.H{"message": "Shop deleted successfully"})
}

// GetUserShops returns all shops for the authenticated user
func (handler *Handler) GetUserShops(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		response.Error(c, 401, "unauthorized")
		slog.Info("Unauthorized request")
		return
	}

	service := handler.service
	shops, err := service.GetShopsByUser(c.Request.Context(), user)
	if err != nil {
		if shared.UsesContract2(c) {
			c.Error(err)
			return
		}
		c.JSON(404, response.EmptyResponseMessage())
		return
	}

	response.OK(c, shops)
}

// GetUserDataWithShops returns user data along with all shops they are a part of
func (handler *Handler) GetUserDataWithShops(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		response.Error(c, 401, "unauthorized")
		slog.Info("Unauthorized request")
		return
	}

	service := handler.service
	userShopsData, err := service.GetUserDataWithShops(c.Request.Context(), user)
	if err != nil {
		c.Error(err)
		return
	}

	// Kept as a raw StandardResponse literal: response.OK hardcodes an empty
	// Message and has no parameter to carry this success text.
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
	shop, err := service.GetShopByID(c.Request.Context(), user, shopID)
	if err != nil {
		c.Error(err)
		return
	}

	response.OK(c, *shop)
}

// UpdateShop handles shop updates
func (handler *Handler) UpdateShop(c *gin.Context) {
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

	var req request.UpdateShopRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		slog.Info("invalid request", "error", err)
		response.Error(c, 400, "invalid request")
		return
	}

	// Nil details preserve stored metadata; only a supplied string changes it.
	shop := model.Shops{
		ID:      shopID,
		Name:    req.Name,
		Details: req.Details,
	}

	service := handler.service
	updatedShop, err := service.UpdateShop(c.Request.Context(), user, shop)
	if err != nil {
		c.Error(err)
		return
	}

	// Kept as a raw StandardResponse literal: response.OK hardcodes an empty
	// Message and has no parameter to carry this success text.
	c.JSON(200, response.StandardResponse{
		Status:  200,
		Message: "Shop updated successfully",
		Data:    *updatedShop,
	})
}
