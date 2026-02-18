package settings

import (
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(router *gin.RouterGroup, service Service) {
	handler := Handler{service: service}
	router.GET("/shops/:shop_id/settings", handler.GetShopSettings)
	router.PUT("/shops/:shop_id/settings", handler.UpdateShopSettings)
	router.GET("/shops/:shop_id/settings/admin-only-lists", handler.GetShopAdminOnlyListsSetting)
	router.PUT("/shops/:shop_id/settings/admin-only-lists", handler.UpdateShopAdminOnlyListsSetting)
	router.GET("/shops/:shop_id/is-admin", handler.CheckUserIsShopAdmin)
}
