package lists

import (
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(router *gin.RouterGroup, service Service) {
	handler := Handler{service: service}
	router.POST("/shops/lists", handler.CreateShopList)
	router.GET("/shops/:shop_id/lists", handler.GetShopLists)
	router.GET("/shops/lists/:list_id", handler.GetShopListByID)
	router.PUT("/shops/lists", handler.UpdateShopList)
	router.DELETE("/shops/lists", handler.DeleteShopList)
}
