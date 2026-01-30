package notification_items

import (
	"log/slog"
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/response"
	"miltechserver/api/user_vehicles/shared"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service Service
}

func RegisterRoutes(router *gin.RouterGroup, service Service) {
	handler := Handler{service: service}

	router.GET("/user/notification-items", handler.getByUser)
	router.GET("/user/notification-items/notification/:notificationId", handler.getByNotification)
	router.GET("/user/notification-items/:itemId", handler.getByID)
	router.PUT("/user/notification-items", handler.upsert)
	router.PUT("/user/notification-items/list", handler.upsertBatch)
	router.DELETE("/user/notification-items/:itemId", handler.delete)
	router.DELETE("/user/notification-items/notification/:notificationId", handler.deleteAllByNotification)
}

func (handler *Handler) getByUser(c *gin.Context) {
	user, err := shared.GetUserFromContext(c)
	if err != nil {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	result, err := handler.service.GetByUser(user)
	if err != nil {
		c.JSON(404, response.EmptyResponseMessage())
		return
	}

	c.JSON(200, response.StandardResponse{
		Status:  200,
		Message: "",
		Data:    result,
	})
}

func (handler *Handler) getByNotification(c *gin.Context) {
	user, err := shared.GetUserFromContext(c)
	if err != nil {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	notificationID := c.Param("notificationId")
	if notificationID == "" {
		c.JSON(400, gin.H{"message": "notification ID is required"})
		return
	}

	result, err := handler.service.GetByNotification(user, notificationID)
	if err != nil {
		c.JSON(404, response.EmptyResponseMessage())
		return
	}

	c.JSON(200, response.StandardResponse{
		Status:  200,
		Message: "",
		Data:    result,
	})
}

func (handler *Handler) getByID(c *gin.Context) {
	user, err := shared.GetUserFromContext(c)
	if err != nil {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	itemID := c.Param("itemId")
	if itemID == "" {
		c.JSON(400, gin.H{"message": "item ID is required"})
		return
	}

	result, err := handler.service.GetByID(user, itemID)
	if err != nil {
		c.JSON(404, response.EmptyResponseMessage())
		return
	}

	c.JSON(200, response.StandardResponse{
		Status:  200,
		Message: "",
		Data:    result,
	})
}

func (handler *Handler) upsert(c *gin.Context) {
	user, err := shared.GetUserFromContext(c)
	if err != nil {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	var item model.UserNotificationItems
	if err := c.BindJSON(&item); err != nil {
		slog.Info("invalid request", "error", err)
		c.JSON(400, gin.H{"message": "invalid request"})
		return
	}

	err = handler.service.Upsert(user, item)
	if err != nil {
		c.Error(err)
		return
	}

	c.Status(200)
}

func (handler *Handler) upsertBatch(c *gin.Context) {
	user, err := shared.GetUserFromContext(c)
	if err != nil {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	var items []model.UserNotificationItems
	if err := c.BindJSON(&items); err != nil {
		slog.Info("invalid request", "error", err)
		c.JSON(400, gin.H{"message": "invalid request"})
		return
	}

	err = handler.service.UpsertBatch(user, items)
	if err != nil {
		c.Error(err)
		return
	}

	c.Status(200)
}

func (handler *Handler) delete(c *gin.Context) {
	user, err := shared.GetUserFromContext(c)
	if err != nil {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	itemID := c.Param("itemId")
	if itemID == "" {
		c.JSON(400, gin.H{"message": "item ID is required"})
		return
	}

	err = handler.service.Delete(user, itemID)
	if err != nil {
		c.Error(err)
		return
	}

	c.Status(200)
}

func (handler *Handler) deleteAllByNotification(c *gin.Context) {
	user, err := shared.GetUserFromContext(c)
	if err != nil {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	notificationID := c.Param("notificationId")
	if notificationID == "" {
		c.JSON(400, gin.H{"message": "notification ID is required"})
		return
	}

	err = handler.service.DeleteAllByNotification(user, notificationID)
	if err != nil {
		c.Error(err)
		return
	}

	c.Status(200)
}
