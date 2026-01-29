package controller

import (
	"log/slog"
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/request"
	"miltechserver/api/response"
	"miltechserver/bootstrap"

	"github.com/gin-gonic/gin"
)

// Shop Vehicle Notification Operations

// CreateVehicleNotification creates a new notification for a vehicle
func (controller *ShopsController) CreateVehicleNotification(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	var req request.CreateVehicleNotificationRequest
	if err := c.BindJSON(&req); err != nil {
		slog.Info("invalid request", "error", err)
		c.JSON(400, gin.H{"message": "invalid request"})
		return
	}

	notification := model.ShopVehicleNotifications{
		VehicleID:   req.VehicleID,
		ShopID:      req.ShopID,
		Title:       req.Title,
		Description: req.Description,
		Type:        req.Type,
		Completed:   false,
	}

	createdNotification, err := controller.ShopsService.CreateVehicleNotification(user, notification)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(201, response.StandardResponse{
		Status:  201,
		Message: "Notification created successfully",
		Data:    *createdNotification,
	})
}

// GetVehicleNotifications returns all notifications for a vehicle
func (controller *ShopsController) GetVehicleNotifications(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	vehicleID := c.Param("vehicle_id")
	if vehicleID == "" {
		c.JSON(400, gin.H{"message": "vehicle_id is required"})
		return
	}

	notifications, err := controller.ShopsService.GetVehicleNotifications(user, vehicleID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(200, response.StandardResponse{
		Status:  200,
		Message: "",
		Data:    notifications,
	})
}

// GetVehicleNotificationsWithItems returns all notifications for a vehicle with their items
func (controller *ShopsController) GetVehicleNotificationsWithItems(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	vehicleID := c.Param("vehicle_id")
	if vehicleID == "" {
		c.JSON(400, gin.H{"message": "vehicle_id is required"})
		return
	}

	notificationsWithItems, err := controller.ShopsService.GetVehicleNotificationsWithItems(user, vehicleID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(200, response.StandardResponse{
		Status:  200,
		Message: "",
		Data:    notificationsWithItems,
	})
}

// GetShopNotifications returns all notifications for a shop
func (controller *ShopsController) GetShopNotifications(c *gin.Context) {
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

	notifications, err := controller.ShopsService.GetShopNotifications(user, shopID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(200, response.StandardResponse{
		Status:  200,
		Message: "",
		Data:    notifications,
	})
}

// GetVehicleNotificationByID returns a specific notification by ID
func (controller *ShopsController) GetVehicleNotificationByID(c *gin.Context) {
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

	notification, err := controller.ShopsService.GetVehicleNotificationByID(user, notificationID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(200, response.StandardResponse{
		Status:  200,
		Message: "",
		Data:    *notification,
	})
}

// UpdateVehicleNotification updates an existing vehicle notification
func (controller *ShopsController) UpdateVehicleNotification(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	var req request.UpdateVehicleNotificationRequest
	if err := c.BindJSON(&req); err != nil {
		slog.Info("invalid request", "error", err)
		c.JSON(400, gin.H{"message": "invalid request"})
		return
	}

	notification := model.ShopVehicleNotifications{
		ID:          req.NotificationID,
		Title:       req.Title,
		Description: req.Description,
		Type:        req.Type,
		Completed:   req.Completed,
	}

	err := controller.ShopsService.UpdateVehicleNotification(user, notification)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(200, gin.H{"message": "Notification updated successfully"})
}

// DeleteVehicleNotification deletes a vehicle notification
func (controller *ShopsController) DeleteVehicleNotification(c *gin.Context) {
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

	err := controller.ShopsService.DeleteVehicleNotification(user, notificationID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(200, gin.H{"message": "Notification deleted successfully"})
}
