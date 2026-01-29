package controller

import (
	"log/slog"
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/request"
	"miltechserver/api/response"
	"miltechserver/bootstrap"

	"github.com/gin-gonic/gin"
)

// Shop Notification Item Operations

// AddNotificationItem adds an item to a vehicle notification
func (controller *ShopsController) AddNotificationItem(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	var req request.AddNotificationItemRequest
	if err := c.BindJSON(&req); err != nil {
		slog.Info("invalid request", "error", err)
		c.JSON(400, gin.H{"message": "invalid request"})
		return
	}

	item := model.ShopNotificationItems{
		NotificationID: req.NotificationID,
		Niin:           req.Niin,
		Nomenclature:   req.Nomenclature,
		Quantity:       req.Quantity,
	}

	createdItem, err := controller.ShopsService.AddNotificationItem(user, item)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(201, response.StandardResponse{
		Status:  201,
		Message: "Item added successfully",
		Data:    *createdItem,
	})
}

// GetNotificationItems returns all items for a notification
func (controller *ShopsController) GetNotificationItems(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	notificationID := c.Param("notification_id")
	if notificationID == "" {
		c.JSON(400, gin.H{"message": "notification_id is required"})
		return
	}

	items, err := controller.ShopsService.GetNotificationItems(user, notificationID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(200, response.StandardResponse{
		Status:  200,
		Message: "",
		Data:    items,
	})
}

// GetShopNotificationItems returns all notification items for a shop
func (controller *ShopsController) GetShopNotificationItems(c *gin.Context) {
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

	items, err := controller.ShopsService.GetShopNotificationItems(user, shopID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(200, response.StandardResponse{
		Status:  200,
		Message: "",
		Data:    items,
	})
}

// AddNotificationItemList adds multiple items to a vehicle notification
func (controller *ShopsController) AddNotificationItemList(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	var req request.AddNotificationItemListRequest
	if err := c.BindJSON(&req); err != nil {
		slog.Info("invalid request", "error", err)
		c.JSON(400, gin.H{"message": "invalid request"})
		return
	}

	var items []model.ShopNotificationItems
	for _, reqItem := range req.Items {
		item := model.ShopNotificationItems{
			NotificationID: reqItem.NotificationID,
			Niin:           reqItem.Niin,
			Nomenclature:   reqItem.Nomenclature,
			Quantity:       reqItem.Quantity,
		}
		items = append(items, item)
	}

	createdItems, err := controller.ShopsService.AddNotificationItemList(user, items)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(201, response.StandardResponse{
		Status:  201,
		Message: "Items added successfully",
		Data:    createdItems,
	})
}

// RemoveNotificationItem removes an item from a vehicle notification
func (controller *ShopsController) RemoveNotificationItem(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	itemID := c.Param("item_id")
	if itemID == "" {
		c.JSON(400, gin.H{"message": "item_id is required"})
		return
	}

	err := controller.ShopsService.RemoveNotificationItem(user, itemID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(200, gin.H{"message": "Item removed successfully"})
}

// RemoveNotificationItemList removes multiple items from vehicle notifications
func (controller *ShopsController) RemoveNotificationItemList(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	var req request.RemoveNotificationItemListRequest
	if err := c.BindJSON(&req); err != nil {
		slog.Info("invalid request", "error", err)
		c.JSON(400, gin.H{"message": "invalid request"})
		return
	}

	err := controller.ShopsService.RemoveNotificationItemList(user, req.ItemIDs)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(200, gin.H{"message": "Items removed successfully", "count": len(req.ItemIDs)})
}
