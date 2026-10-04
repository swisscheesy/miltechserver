package capabilities

import (
	"context"
	"miltechserver/api/response"
	"miltechserver/api/shops/shared"
	"miltechserver/bootstrap"

	"github.com/gin-gonic/gin"
)

// Flags carries the server-side capability switches. MessageSyncReady is
// consulted per request so a disabled trigger turns the capability off without
// a restart; nil means unavailable.
type Flags struct {
	AtomicNotificationSave  bool
	AtomicNotificationReady func(context.Context) bool
	MessageSyncReady        func(context.Context) bool
}

func RegisterRoutes(router *gin.RouterGroup, flags Flags) {
	router.GET("/shops/capabilities", func(c *gin.Context) {
		get(c, flags)
	})
}

func get(c *gin.Context, flags Flags) {
	user, ok := c.Get("user")
	u, valid := user.(*bootstrap.User)
	if !ok || !valid || u == nil {
		response.Error(c, 401, "unauthorized")
		return
	}
	if !shared.RequireContract2(c) {
		return
	}
	// message_sync is true only when SHOPS_MESSAGE_SYNC_ENABLED is on and the counter schema/trigger exist; backfill and fleet readiness are runbook gates (docs/testing/shops-release-contracts.md).
	messageSync := flags.MessageSyncReady != nil && flags.MessageSyncReady(c.Request.Context())
	c.Header("Cache-Control", "no-store")
	response.OK(c, gin.H{"contract_version": 2, "typed_errors": false, "atomic_notification_save": flags.AtomicNotificationSave && flags.AtomicNotificationReady != nil && flags.AtomicNotificationReady(c.Request.Context()), "service_dates": false, "service_reads": false, "message_sync": messageSync})
}
