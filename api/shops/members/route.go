package members

import (
	"miltechserver/api/controller"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(router *gin.RouterGroup, controller *controller.ShopsController) {
	router.POST("/shops/join", controller.JoinShopViaInviteCode)
	router.DELETE("/shops/:shop_id/leave", controller.LeaveShop)
	router.DELETE("/shops/members/remove", controller.RemoveMemberFromShop)
	router.PUT("/shops/members/promote", controller.PromoteMemberToAdmin)
	router.GET("/shops/:shop_id/members", controller.GetShopMembers)
}
