package lists

import (
	"miltechserver/api/controller"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(router *gin.RouterGroup, controller *controller.ShopsController) {
	router.POST("/shops/lists", controller.CreateShopList)
	router.GET("/shops/:shop_id/lists", controller.GetShopLists)
	router.GET("/shops/lists/:list_id", controller.GetShopListByID)
	router.PUT("/shops/lists", controller.UpdateShopList)
	router.DELETE("/shops/lists", controller.DeleteShopList)
}
