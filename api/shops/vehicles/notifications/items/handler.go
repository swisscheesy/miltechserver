package items

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

// Shop Notification Item Operations

// AddNotificationItem adds an item to a vehicle notification
func (handler *Handler) AddNotificationItem(c *gin.Context) {
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

	service := handler.service
	createdItem, err := service.AddNotificationItem(user, item)
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
func (handler *Handler) GetNotificationItems(c *gin.Context) {
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

	service := handler.service
	items, err := service.GetNotificationItems(user, notificationID)
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
func (handler *Handler) GetShopNotificationItems(c *gin.Context) {
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
	items, err := service.GetShopNotificationItems(user, shopID)
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
func (handler *Handler) AddNotificationItemList(c *gin.Context) {
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

	service := handler.service
	createdItems, err := service.AddNotificationItemList(user, items)
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
func (handler *Handler) RemoveNotificationItem(c *gin.Context) {
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

	service := handler.service
	err := service.RemoveNotificationItem(user, itemID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(200, gin.H{"message": "Item removed successfully"})
}

// RemoveNotificationItemList removes multiple items from vehicle notifications
func (handler *Handler) RemoveNotificationItemList(c *gin.Context) {
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

	service := handler.service
	err := service.RemoveNotificationItemList(user, req.ItemIDs)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(200, gin.H{"message": "Items removed successfully", "count": len(req.ItemIDs)})
}
