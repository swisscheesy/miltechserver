package items

import (
	"miltechserver/api/controller"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(router *gin.RouterGroup, controller *controller.ShopsController) {
	router.POST("/shops/notifications/items", controller.AddNotificationItem)
	router.GET("/shops/notifications/:notification_id/items", controller.GetNotificationItems)
	router.GET("/shops/:shop_id/notification-items", controller.GetShopNotificationItems)
	router.POST("/shops/notifications/items/bulk", controller.AddNotificationItemList)
	router.DELETE("/shops/notifications/items/:item_id", controller.RemoveNotificationItem)
	router.DELETE("/shops/notifications/items/bulk", controller.RemoveNotificationItemList)
}
