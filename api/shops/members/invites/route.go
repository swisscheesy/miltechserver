package invites

import (
	"miltechserver/api/controller"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(router *gin.RouterGroup, controller *controller.ShopsController) {
	router.POST("/shops/invite-codes", controller.GenerateInviteCode)
	router.GET("/shops/:shop_id/invite-codes", controller.GetInviteCodesByShop)
	router.DELETE("/shops/invite-codes/:code_id", controller.DeactivateInviteCode)
	router.DELETE("/shops/invite-codes/:code_id/delete", controller.DeleteInviteCode)
}
