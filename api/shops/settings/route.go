package settings

import (
	"miltechserver/api/controller"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(router *gin.RouterGroup, controller *controller.ShopsController) {
	router.GET("/shops/:shop_id/settings", controller.GetShopSettings)
	router.PUT("/shops/:shop_id/settings", controller.UpdateShopSettings)
	router.GET("/shops/:shop_id/settings/admin-only-lists", controller.GetShopAdminOnlyListsSetting)
	router.PUT("/shops/:shop_id/settings/admin-only-lists", controller.UpdateShopAdminOnlyListsSetting)
}
