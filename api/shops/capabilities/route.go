package capabilities

import (
	"github.com/gin-gonic/gin"
	"miltechserver/api/response"
	"miltechserver/api/shops/shared"
	"miltechserver/bootstrap"
)

func RegisterRoutes(router *gin.RouterGroup) { router.GET("/shops/capabilities", Get) }
func Get(c *gin.Context) {
	user, ok := c.Get("user")
	u, valid := user.(*bootstrap.User)
	if !ok || !valid || u == nil {
		response.Error(c, 401, "unauthorized")
		return
	}
	if !shared.RequireContract2(c) {
		return
	}
	// Activation requires released-client and serving-instance evidence.
	response.OK(c, gin.H{"contract_version": 2, "typed_errors": false, "atomic_notification_save": false, "service_dates": false, "service_reads": false, "message_sync": false})
}
