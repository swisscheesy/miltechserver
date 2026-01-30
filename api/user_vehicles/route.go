package user_vehicles

import (
	"database/sql"
	"miltechserver/api/user_vehicles/notification_items"
	"miltechserver/api/user_vehicles/notifications"
	"miltechserver/api/user_vehicles/vehicles"

	"github.com/gin-gonic/gin"
)

type Dependencies struct {
	DB *sql.DB
}

func RegisterRoutes(deps Dependencies, group *gin.RouterGroup) {
	vehiclesRepository := vehicles.NewRepository(deps.DB)
	notificationsRepository := notifications.NewRepository(deps.DB)
	notificationItemsRepository := notification_items.NewRepository(deps.DB)

	vehiclesService := vehicles.NewService(vehiclesRepository)
	notificationsService := notifications.NewService(notificationsRepository)
	notificationItemsService := notification_items.NewService(notificationItemsRepository)

	vehicles.RegisterRoutes(group, vehiclesService)
	notifications.RegisterRoutes(group, notificationsService)
	notification_items.RegisterRoutes(group, notificationItemsService)
}
