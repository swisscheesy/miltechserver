package messages

import (
	"miltechserver/api/controller"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(router *gin.RouterGroup, controller *controller.ShopsController) {
	router.POST("/shops/messages", controller.CreateShopMessage)
	router.GET("/shops/:shop_id/messages", controller.GetShopMessages)
	router.GET("/shops/:shop_id/messages/paginated", controller.GetShopMessagesPaginated)
	router.PUT("/shops/messages", controller.UpdateShopMessage)
	router.DELETE("/shops/messages/:message_id", controller.DeleteShopMessage)
	router.POST("/shops/messages/image/upload", controller.UploadMessageImage)
	router.DELETE("/shops/messages/image/:message_id", controller.DeleteMessageImage)
}
