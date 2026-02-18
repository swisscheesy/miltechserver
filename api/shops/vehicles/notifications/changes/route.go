package changes

import (
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(router *gin.RouterGroup, service Service) {
	handler := Handler{service: service}
	router.GET("/shops/notifications/:notification_id/changes", handler.GetNotificationChangeHistory)
	router.GET("/shops/:shop_id/notifications/changes", handler.GetShopNotificationChanges)
	router.GET("/shops/vehicles/:vehicle_id/notifications/changes", handler.GetVehicleNotificationChanges)
}
