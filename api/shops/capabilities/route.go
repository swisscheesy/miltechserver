package capabilities

import (
	"github.com/gin-gonic/gin"
	"miltechserver/api/response"
	"miltechserver/api/shops/shared"
	"miltechserver/bootstrap"
)

func RegisterRoutes(router *gin.RouterGroup, atomicNotificationSaveEnabled bool) {
	router.GET("/shops/capabilities", func(c *gin.Context) {
		get(c, atomicNotificationSaveEnabled)
	})
}

func get(c *gin.Context, atomicNotificationSaveEnabled bool) {
	user, ok := c.Get("user")
	u, valid := user.(*bootstrap.User)
	if !ok || !valid || u == nil {
		response.Error(c, 401, "unauthorized")
		return
	}
	if !shared.RequireContract2(c) {
		return
	}
	// message_sync also requires verified counter backfill and all-writer allocation;
	// the additive message sync service remains closed while F1 is blocked.
	c.Header("Cache-Control", "no-store")
	response.OK(c, gin.H{"contract_version": 2, "typed_errors": false, "atomic_notification_save": atomicNotificationSaveEnabled, "service_dates": false, "service_reads": false, "message_sync": false})
}
