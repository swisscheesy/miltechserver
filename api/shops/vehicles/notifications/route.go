package notifications

import (
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(router *gin.RouterGroup, service Service) {
	handler := Handler{service: service}
	router.POST("/shops/vehicles/notifications", handler.CreateVehicleNotification)
	router.GET("/shops/vehicles/:vehicle_id/notifications", handler.GetVehicleNotifications)
	router.GET("/shops/vehicles/:vehicle_id/notifications-with-items", handler.GetVehicleNotificationsWithItems)
	router.GET("/shops/:shop_id/notifications", handler.GetShopNotifications)
	router.GET("/shops/vehicles/notifications/:notification_id", handler.GetVehicleNotificationByID)
	router.PUT("/shops/vehicles/notifications", handler.UpdateVehicleNotification)
	router.DELETE("/shops/vehicles/notifications/:notification_id", handler.DeleteVehicleNotification)
}
