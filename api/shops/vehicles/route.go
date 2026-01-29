package vehicles

import (
	"miltechserver/api/controller"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(router *gin.RouterGroup, controller *controller.ShopsController) {
	router.POST("/shops/vehicles", controller.CreateShopVehicle)
	router.GET("/shops/:shop_id/vehicles", controller.GetShopVehicles)
	router.GET("/shops/vehicles/:vehicle_id", controller.GetShopVehicleByID)
	router.PUT("/shops/vehicles", controller.UpdateShopVehicle)
	router.DELETE("/shops/vehicles/:vehicle_id", controller.DeleteShopVehicle)
}
