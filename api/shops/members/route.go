package members

import (
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(router *gin.RouterGroup, service Service) {
	handler := Handler{service: service}
	router.POST("/shops/join", handler.JoinShopViaInviteCode)
	router.DELETE("/shops/:shop_id/leave", handler.LeaveShop)
	router.DELETE("/shops/members/remove", handler.RemoveMemberFromShop)
	router.PUT("/shops/members/promote", handler.PromoteMemberToAdmin)
	router.GET("/shops/:shop_id/members", handler.GetShopMembers)
}
