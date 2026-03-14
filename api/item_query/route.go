package item_query

import (
	"database/sql"

	"github.com/gin-gonic/gin"

	"miltechserver/api/analytics"
	"miltechserver/api/item_query/detailed"
	"miltechserver/api/item_query/help"
	"miltechserver/api/item_query/short"
)

type Dependencies struct {
	DB *sql.DB
}

func RegisterRoutes(deps Dependencies, router *gin.RouterGroup) {
	analyticsService := analytics.New(deps.DB)

	shortRepo := short.NewRepository(deps.DB)
	shortService := short.NewService(shortRepo, analyticsService)
	short.RegisterRoutes(router, shortService)

	detailedRepo := detailed.NewRepository(deps.DB)
	detailedService := detailed.NewService(detailedRepo)
	detailed.RegisterRoutes(router, detailedService)

	helpRepo := help.NewRepository(deps.DB)
	helpService := help.NewService(helpRepo)
	help.RegisterRoutes(router, helpService)
}
