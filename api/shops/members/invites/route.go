package invites

import (
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(router *gin.RouterGroup, service Service) {
	handler := Handler{service: service}
	router.POST("/shops/invite-codes", handler.GenerateInviteCode)
	router.GET("/shops/:shop_id/invite-codes", handler.GetInviteCodesByShop)
	router.DELETE("/shops/invite-codes/:code_id", handler.DeactivateInviteCode)
	router.DELETE("/shops/invite-codes/:code_id/delete", handler.DeleteInviteCode)
}
