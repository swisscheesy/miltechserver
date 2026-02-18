package core

import (
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(router *gin.RouterGroup, service ShopService) {
	handler := Handler{service: service}
	router.POST("/shops", handler.CreateShop)
	router.GET("/shops", handler.GetUserShops)
	router.GET("/shops/user-data", handler.GetUserDataWithShops)
	router.GET("/shops/:shop_id", handler.GetShopByID)
	router.PUT("/shops/:shop_id", handler.UpdateShop)
	router.DELETE("/shops/:shop_id", handler.DeleteShop)
}
