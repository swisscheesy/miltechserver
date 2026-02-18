package lists

import (
	"log/slog"
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/request"
	"miltechserver/api/response"
	"miltechserver/bootstrap"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service Service
}

// Shop List Operations

// CreateShopList creates a new list for a shop
func (handler *Handler) CreateShopList(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	var req request.CreateShopListRequest
	if err := c.BindJSON(&req); err != nil {
		slog.Info("invalid request", "error", err)
		c.JSON(400, gin.H{"message": "invalid request"})
		return
	}

	list := model.ShopLists{
		ShopID:      req.ShopID,
		Description: req.Description,
	}

	service := handler.service
	createdList, err := service.CreateShopList(user, list)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(201, response.StandardResponse{
		Status:  201,
		Message: "List created successfully",
		Data:    *createdList,
	})
}

// GetShopLists returns all lists for a shop with creator usernames
func (handler *Handler) GetShopLists(c *gin.Context) {
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
	lists, err := service.GetShopLists(user, shopID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(200, response.StandardResponse{
		Status:  200,
		Message: "",
		Data:    lists,
	})
}

// GetShopListByID returns a specific list by ID
func (handler *Handler) GetShopListByID(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	listID := c.Param("list_id")
	if listID == "" {
		c.JSON(400, gin.H{"message": "list_id is required"})
		return
	}

	service := handler.service
	list, err := service.GetShopListByID(user, listID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(200, response.StandardResponse{
		Status:  200,
		Message: "",
		Data:    *list,
	})
}

// UpdateShopList updates an existing shop list
func (handler *Handler) UpdateShopList(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	var req request.UpdateShopListRequest
	if err := c.BindJSON(&req); err != nil {
		slog.Info("invalid request", "error", err)
		c.JSON(400, gin.H{"message": "invalid request"})
		return
	}

	list := model.ShopLists{
		ID:          req.ListID,
		Description: req.Description,
	}

	service := handler.service
	err := service.UpdateShopList(user, list)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(200, gin.H{"message": "List updated successfully"})
}

// DeleteShopList deletes a shop list
func (handler *Handler) DeleteShopList(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	var req request.DeleteShopListRequest
	if err := c.BindJSON(&req); err != nil {
		slog.Info("invalid request", "error", err)
		c.JSON(400, gin.H{"message": "invalid request"})
		return
	}

	service := handler.service
	err := service.DeleteShopList(user, req.ListID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(200, gin.H{"message": "List deleted successfully"})
}
