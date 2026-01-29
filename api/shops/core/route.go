package core

import (
	"miltechserver/api/controller"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(router *gin.RouterGroup, controller *controller.ShopsController) {
	router.POST("/shops", controller.CreateShop)
	router.GET("/shops", controller.GetUserShops)
	router.GET("/shops/user-data", controller.GetUserDataWithShops)
	router.GET("/shops/:shop_id", controller.GetShopByID)
	router.PUT("/shops/:shop_id", controller.UpdateShop)
	router.DELETE("/shops/:shop_id", controller.DeleteShop)
	router.GET("/shops/:shop_id/is-admin", controller.CheckUserIsShopAdmin)
}
