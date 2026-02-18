package shops

import (
	"database/sql"
	"miltechserver/api/shops/core"
	"miltechserver/api/shops/lists"
	listitems "miltechserver/api/shops/lists/items"
	"miltechserver/api/shops/members"
	"miltechserver/api/shops/members/invites"
	"miltechserver/api/shops/messages"
	"miltechserver/api/shops/settings"
	"miltechserver/api/shops/shared"
	"miltechserver/api/shops/vehicles"
	"miltechserver/api/shops/vehicles/notifications"
	notificationchanges "miltechserver/api/shops/vehicles/notifications/changes"
	notificationitems "miltechserver/api/shops/vehicles/notifications/items"
	"miltechserver/bootstrap"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/gin-gonic/gin"
)

type Dependencies struct {
	DB         *sql.DB
	BlobClient *azblob.Client
	Env        *bootstrap.Env
}

func RegisterRoutes(deps Dependencies, router *gin.RouterGroup) {
	authorization := shared.NewShopAuthorization(deps.DB)

	coreRepository := core.NewRepository(deps.DB, deps.BlobClient, deps.Env)
	settingsRepository := settings.NewRepository(deps.DB)
	membersRepository := members.NewRepository(deps.DB, deps.BlobClient, deps.Env)
	inviteRepository := invites.NewRepository(deps.DB)
	listRepository := lists.NewRepository(deps.DB)
	listItemsRepository := listitems.NewRepository(deps.DB)
	messagesRepository := messages.NewRepository(deps.DB, deps.BlobClient, deps.Env)
	vehiclesRepository := vehicles.NewRepository(deps.DB)
	notificationsRepository := notifications.NewRepository(deps.DB)
	notificationItemsRepository := notificationitems.NewRepository(deps.DB)
	notificationChangesRepository := notificationchanges.NewRepository(deps.DB)

	coreService := core.NewService(coreRepository, authorization)
	settingsService := settings.NewService(settingsRepository, authorization)
	membersService := members.NewService(membersRepository, inviteRepository, authorization)
	inviteService := invites.NewService(inviteRepository, authorization)
	listsService := lists.NewService(listRepository, settingsRepository, authorization)
	listItemsService := listitems.NewService(listItemsRepository, listRepository, settingsRepository, authorization)
	messagesService := messages.NewService(messagesRepository, authorization)
	vehiclesService := vehicles.NewService(vehiclesRepository, authorization)
	notificationsService := notifications.NewService(notificationsRepository, authorization)
	notificationItemsService := notificationitems.NewService(notificationItemsRepository)
	notificationChangesService := notificationchanges.NewService(notificationChangesRepository)

	core.RegisterRoutes(router, coreService)
	settings.RegisterRoutes(router, settingsService)
	members.RegisterRoutes(router, membersService)
	invites.RegisterRoutes(router, inviteService)
	messages.RegisterRoutes(router, messagesService)
	vehicles.RegisterRoutes(router, vehiclesService)
	notifications.RegisterRoutes(router, notificationsService)
	notificationitems.RegisterRoutes(router, notificationItemsService)
	notificationchanges.RegisterRoutes(router, notificationChangesService)
	lists.RegisterRoutes(router, listsService)
	listitems.RegisterRoutes(router, listItemsService)
}
