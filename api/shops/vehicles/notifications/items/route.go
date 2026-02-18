package items

import (
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(router *gin.RouterGroup, service Service) {
	handler := Handler{service: service}
	router.POST("/shops/notifications/items", handler.AddNotificationItem)
	router.GET("/shops/notifications/:notification_id/items", handler.GetNotificationItems)
	router.GET("/shops/:shop_id/notification-items", handler.GetShopNotificationItems)
	router.POST("/shops/notifications/items/bulk", handler.AddNotificationItemList)
	router.DELETE("/shops/notifications/items/:item_id", handler.RemoveNotificationItem)
	router.DELETE("/shops/notifications/items/bulk", handler.RemoveNotificationItemList)
}
