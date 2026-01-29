package changes

import (
	"miltechserver/api/controller"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(router *gin.RouterGroup, controller *controller.ShopsController) {
	router.GET("/shops/notifications/:notification_id/changes", controller.GetNotificationChangeHistory)
	router.GET("/shops/:shop_id/notifications/changes", controller.GetShopNotificationChanges)
	router.GET("/shops/vehicles/:vehicle_id/notifications/changes", controller.GetVehicleNotificationChanges)
}
