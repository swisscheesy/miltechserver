package notifications

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

	router.GET("/user/vehicle-notifications", handler.getByUser)
	router.GET("/user/vehicle-notifications/vehicle/:vehicleId", handler.getByVehicle)
	router.GET("/user/vehicle-notifications/:notificationId", handler.getByID)
	router.PUT("/user/vehicle-notifications", handler.upsert)
	router.DELETE("/user/vehicle-notifications/:notificationId", handler.delete)
	router.DELETE("/user/vehicle-notifications/vehicle/:vehicleId", handler.deleteAllByVehicle)
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

func (handler *Handler) getByVehicle(c *gin.Context) {
	user, err := shared.GetUserFromContext(c)
	if err != nil {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	vehicleID := c.Param("vehicleId")
	if vehicleID == "" {
		c.JSON(400, gin.H{"message": "vehicle ID is required"})
		return
	}

	result, err := handler.service.GetByVehicle(user, vehicleID)
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

	notificationID := c.Param("notificationId")
	if notificationID == "" {
		c.JSON(400, gin.H{"message": "notification ID is required"})
		return
	}

	result, err := handler.service.GetByID(user, notificationID)
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

	var notification model.UserVehicleNotifications
	if err := c.BindJSON(&notification); err != nil {
		slog.Info("invalid request", "error", err)
		c.JSON(400, gin.H{"message": "invalid request"})
		return
	}

	err = handler.service.Upsert(user, notification)
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

	notificationID := c.Param("notificationId")
	if notificationID == "" {
		c.JSON(400, gin.H{"message": "notification ID is required"})
		return
	}

	err = handler.service.Delete(user, notificationID)
	if err != nil {
		c.Error(err)
		return
	}

	c.Status(200)
}

func (handler *Handler) deleteAllByVehicle(c *gin.Context) {
	user, err := shared.GetUserFromContext(c)
	if err != nil {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	vehicleID := c.Param("vehicleId")
	if vehicleID == "" {
		c.JSON(400, gin.H{"message": "vehicle ID is required"})
		return
	}

	err = handler.service.DeleteAllByVehicle(user, vehicleID)
	if err != nil {
		c.Error(err)
		return
	}

	c.Status(200)
}
