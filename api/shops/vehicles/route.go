package vehicles

import (
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(router *gin.RouterGroup, service Service) {
	handler := Handler{service: service}
	router.POST("/shops/vehicles", handler.CreateShopVehicle)
	router.GET("/shops/:shop_id/vehicles", handler.GetShopVehicles)
	router.GET("/shops/vehicles/:vehicle_id", handler.GetShopVehicleByID)
	router.PUT("/shops/vehicles", handler.UpdateShopVehicle)
	router.DELETE("/shops/vehicles/:vehicle_id", handler.DeleteShopVehicle)
}
