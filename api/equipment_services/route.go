package equipment_services

import (
	"database/sql"

	"github.com/gin-gonic/gin"

	"miltechserver/api/equipment_services/calendar"
	"miltechserver/api/equipment_services/completion"
	"miltechserver/api/equipment_services/core"
	"miltechserver/api/equipment_services/queries"
	"miltechserver/api/equipment_services/shared"
	"miltechserver/api/equipment_services/status"
	shopsShared "miltechserver/api/shops/shared"
)

type Dependencies struct {
	DB *sql.DB
}

func RegisterRoutes(deps Dependencies, router *gin.RouterGroup) {
	shopAuth := shopsShared.NewShopAuthorization(deps.DB)
	authorization := shared.NewAuthorization(deps.DB, shopAuth)
	usernameResolver := shared.NewUsernameRepository(deps.DB)

	coreRepo := core.NewRepository(deps.DB)
	queriesRepo := queries.NewRepository(deps.DB)
	calendarRepo := calendar.NewRepository(deps.DB)
	statusRepo := status.NewRepository(deps.DB)
	completionRepo := completion.NewRepository(deps.DB)

	coreService := core.NewService(coreRepo, authorization, usernameResolver)
	queriesService := queries.NewService(queriesRepo, authorization, usernameResolver)
	calendarService := calendar.NewService(calendarRepo, authorization, usernameResolver)
	statusService := status.NewService(statusRepo, authorization, usernameResolver)
	completionService := completion.NewService(completionRepo, authorization, usernameResolver)

	core.RegisterRoutes(router, coreService)
	queries.RegisterRoutes(router, queriesService)
	calendar.RegisterRoutes(router, calendarService)
	status.RegisterRoutes(router, statusService)
	completion.RegisterRoutes(router, completionService)
}
