package changes

import (
	"fmt"
	"log/slog"
	"miltechserver/api/response"
	"miltechserver/bootstrap"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service Service
}

// Notification Change Tracking (Audit Trail) Operations

// GetNotificationChangeHistory returns the complete change history for a notification
func (handler *Handler) GetNotificationChangeHistory(c *gin.Context) {
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
	changes, err := service.GetNotificationChangeHistory(user, notificationID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(200, response.StandardResponse{
		Status:  200,
		Message: "",
		Data:    changes,
	})
}

// GetShopNotificationChanges returns recent notification changes for all notifications in a shop
func (handler *Handler) GetShopNotificationChanges(c *gin.Context) {
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

	// Get optional limit parameter (default 100, max 500)
	limit := 100
	if limitParam := c.Query("limit"); limitParam != "" {
		var parsedLimit int
		if _, err := fmt.Sscanf(limitParam, "%d", &parsedLimit); err == nil {
			limit = parsedLimit
		}
	}

	service := handler.service
	changes, err := service.GetShopNotificationChanges(user, shopID, limit)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(200, response.StandardResponse{
		Status:  200,
		Message: "",
		Data:    changes,
	})
}

// GetVehicleNotificationChanges returns all notification changes for a specific vehicle
func (handler *Handler) GetVehicleNotificationChanges(c *gin.Context) {
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

	service := handler.service
	changes, err := service.GetVehicleNotificationChanges(user, vehicleID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(200, response.StandardResponse{
		Status:  200,
		Message: "",
		Data:    changes,
	})
}
