package notifications

import (
	"miltechserver/api/controller"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(router *gin.RouterGroup, controller *controller.ShopsController) {
	router.POST("/shops/vehicles/notifications", controller.CreateVehicleNotification)
	router.GET("/shops/vehicles/:vehicle_id/notifications", controller.GetVehicleNotifications)
	router.GET("/shops/vehicles/:vehicle_id/notifications-with-items", controller.GetVehicleNotificationsWithItems)
	router.GET("/shops/:shop_id/notifications", controller.GetShopNotifications)
	router.GET("/shops/vehicles/notifications/:notification_id", controller.GetVehicleNotificationByID)
	router.PUT("/shops/vehicles/notifications", controller.UpdateVehicleNotification)
	router.DELETE("/shops/vehicles/notifications/:notification_id", controller.DeleteVehicleNotification)
}
