package route

import (
	"database/sql"
	"miltechserver/api/controller"
	"miltechserver/api/repository"
	"miltechserver/api/service"
	"miltechserver/bootstrap"

	"github.com/gin-gonic/gin"
)

func NewShopsRouter(db *sql.DB, env *bootstrap.Env, group *gin.RouterGroup) {
	shopsRepository := repository.NewShopsRepositoryImpl(db)

	pc := &controller.ShopsController{
		ShopsService: service.NewShopsServiceImpl(shopsRepository),
	}

	// Shop Operations
	group.POST("/shops", pc.CreateShop)
	group.GET("/shops", pc.GetUserShops)
	group.GET("/shops/user-data", pc.GetUserDataWithShops)
	group.GET("/shops/:shop_id", pc.GetShopByID)
	group.PUT("/shops/:shop_id", pc.UpdateShop)
	group.DELETE("/shops/:shop_id", pc.DeleteShop)

	// Shop Member Operations
	group.POST("/shops/join", pc.JoinShopViaInviteCode)
	group.DELETE("/shops/:shop_id/leave", pc.LeaveShop)
	group.DELETE("/shops/members/remove", pc.RemoveMemberFromShop)
	group.PUT("/shops/members/promote", pc.PromoteMemberToAdmin)
	group.GET("/shops/:shop_id/members", pc.GetShopMembers)

	// Shop Invite Code Operations
	group.POST("/shops/invite-codes", pc.GenerateInviteCode)
	group.GET("/shops/:shop_id/invite-codes", pc.GetInviteCodesByShop)
	group.DELETE("/shops/invite-codes/:code_id", pc.DeactivateInviteCode)
	group.DELETE("/shops/invite-codes/:code_id/delete", pc.DeleteInviteCode)

	// Shop Message Operations
	group.POST("/shops/messages", pc.CreateShopMessage)
	group.GET("/shops/:shop_id/messages", pc.GetShopMessages)
	group.GET("/shops/:shop_id/messages/paginated", pc.GetShopMessagesPaginated)
	group.PUT("/shops/messages", pc.UpdateShopMessage)
	group.DELETE("/shops/messages/:message_id", pc.DeleteShopMessage)

	// Shop Vehicle Operations
	group.POST("/shops/vehicles", pc.CreateShopVehicle)
	group.GET("/shops/:shop_id/vehicles", pc.GetShopVehicles)
	group.GET("/shops/vehicles/:vehicle_id", pc.GetShopVehicleByID)
	group.PUT("/shops/vehicles", pc.UpdateShopVehicle)
	group.DELETE("/shops/vehicles/:vehicle_id", pc.DeleteShopVehicle)

	// Shop Vehicle Notification Operations
	group.POST("/shops/vehicles/notifications", pc.CreateVehicleNotification)
	group.GET("/shops/vehicles/:vehicle_id/notifications", pc.GetVehicleNotifications)
	group.GET("/shops/vehicles/:vehicle_id/notifications-with-items", pc.GetVehicleNotificationsWithItems)
	group.GET("/shops/:shop_id/notifications", pc.GetShopNotifications)
	group.GET("/shops/vehicles/notifications/:notification_id", pc.GetVehicleNotificationByID)
	group.PUT("/shops/vehicles/notifications", pc.UpdateVehicleNotification)
	group.DELETE("/shops/vehicles/notifications/:notification_id", pc.DeleteVehicleNotification)

	// Shop Notification Item Operations
	group.POST("/shops/notifications/items", pc.AddNotificationItem)
	group.GET("/shops/notifications/:notification_id/items", pc.GetNotificationItems)
	group.GET("/shops/:shop_id/notification-items", pc.GetShopNotificationItems)
	group.POST("/shops/notifications/items/bulk", pc.AddNotificationItemList)
	group.DELETE("/shops/notifications/items/:item_id", pc.RemoveNotificationItem)
	group.DELETE("/shops/notifications/items/bulk", pc.RemoveNotificationItemList)

	// Shop List Operations
	group.POST("/shops/lists", pc.CreateShopList)
	group.GET("/shops/:shop_id/lists", pc.GetShopLists)
	group.GET("/shops/lists/:list_id", pc.GetShopListByID)
	group.PUT("/shops/lists", pc.UpdateShopList)
	group.DELETE("/shops/lists", pc.DeleteShopList)

	// Shop List Item Operations
	group.POST("/shops/lists/items", pc.AddListItem)
	group.GET("/shops/lists/:list_id/items", pc.GetListItems)
	group.PUT("/shops/lists/items", pc.UpdateListItem)
	group.DELETE("/shops/lists/items", pc.RemoveListItem)
	group.POST("/shops/lists/items/bulk", pc.AddListItemBatch)
	group.DELETE("/shops/lists/items/bulk", pc.RemoveListItemBatch)

	// Notification Change Tracking (Audit Trail) Operations
	group.GET("/shops/notifications/:notification_id/changes", pc.GetNotificationChangeHistory)
	group.GET("/shops/:shop_id/notifications/changes", pc.GetShopNotificationChanges)
	group.GET("/shops/vehicles/:vehicle_id/notifications/changes", pc.GetVehicleNotificationChanges)
}
