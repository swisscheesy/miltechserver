package route

import (
	"database/sql"
	"miltechserver/api/controller"
	"miltechserver/api/repository"
	"miltechserver/api/service"

	"github.com/gin-gonic/gin"
)

func NewUserVehicleRouter(db *sql.DB, group *gin.RouterGroup) {
	userVehicleRepository := repository.NewUserVehicleRepositoryImpl(db)

	pc := &controller.UserVehicleController{
		UserVehicleService: service.NewUserVehicleServiceImpl(
			userVehicleRepository),
	}

	// User Vehicle Routes
	group.GET("/user/vehicles", pc.GetUserVehiclesByUser)
	group.GET("/user/vehicles/:vehicleId", pc.GetUserVehicleById)
	group.PUT("/user/vehicles", pc.UpsertUserVehicle)
	group.DELETE("/user/vehicles/:vehicleId", pc.DeleteUserVehicle)
	group.DELETE("/user/vehicles", pc.DeleteAllUserVehicles)

	// User Vehicle Notifications Routes
	group.GET("/user/vehicle-notifications", pc.GetVehicleNotificationsByUser)
	group.GET("/user/vehicle-notifications/vehicle/:vehicleId", pc.GetVehicleNotificationsByVehicle)
	group.GET("/user/vehicle-notifications/:notificationId", pc.GetVehicleNotificationById)
	group.PUT("/user/vehicle-notifications", pc.UpsertVehicleNotification)
	group.DELETE("/user/vehicle-notifications/:notificationId", pc.DeleteVehicleNotification)
	group.DELETE("/user/vehicle-notifications/vehicle/:vehicleId", pc.DeleteAllVehicleNotificationsByVehicle)

	// User Vehicle Comments Routes -- Not Used
	group.GET("/user/vehicle-comments", pc.GetVehicleCommentsByUser)
	group.GET("/user/vehicle-comments/vehicle/:vehicleId", pc.GetVehicleCommentsByVehicle)
	group.GET("/user/vehicle-comments/notification/:notificationId", pc.GetVehicleCommentsByNotification)
	group.GET("/user/vehicle-comments/:commentId", pc.GetVehicleCommentById)
	group.PUT("/user/vehicle-comments", pc.UpsertVehicleComment)
	group.DELETE("/user/vehicle-comments/:commentId", pc.DeleteVehicleComment)
	group.DELETE("/user/vehicle-comments/vehicle/:vehicleId", pc.DeleteAllVehicleCommentsByVehicle)

	// User Notification Items Routes
	group.GET("/user/notification-items", pc.GetNotificationItemsByUser)
	group.GET("/user/notification-items/notification/:notificationId", pc.GetNotificationItemsByNotification)
	group.GET("/user/notification-items/:itemId", pc.GetNotificationItemById)
	group.PUT("/user/notification-items", pc.UpsertNotificationItem)
	group.PUT("/user/notification-items/list", pc.UpsertNotificationItemList)
	group.DELETE("/user/notification-items/:itemId", pc.DeleteNotificationItem)
	group.DELETE("/user/notification-items/notification/:notificationId", pc.DeleteAllNotificationItemsByNotification)
}
