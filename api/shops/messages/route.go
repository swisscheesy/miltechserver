package messages

import (
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(router *gin.RouterGroup, service Service) {
	handler := Handler{service: service}
	router.POST("/shops/messages", handler.CreateShopMessage)
	router.GET("/shops/:shop_id/messages", handler.GetShopMessages)
	router.GET("/shops/:shop_id/messages/paginated", handler.GetShopMessagesPaginated)
	router.PUT("/shops/messages", handler.UpdateShopMessage)
	router.DELETE("/shops/messages/:message_id", handler.DeleteShopMessage)
	router.POST("/shops/messages/image/upload", handler.UploadMessageImage)
	router.DELETE("/shops/messages/image/:message_id", handler.DeleteMessageImage)
}
