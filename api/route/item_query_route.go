package route

import (
	"database/sql"
	"github.com/gin-gonic/gin"
	"miltechserver/api/controller"
	"miltechserver/api/repository"
	"miltechserver/api/service"
)

func NewItemQueryRouter(db *sql.DB, group *gin.RouterGroup) {
	itemQueryRepo := repository.NewItemQueryRepositoryImpl(db)
	itemDetailedRepo := repository.NewItemDetailedRepositoryImpl(db)
	analyticsRepo := repository.NewAnalyticsRepositoryImpl(db)
	analyticsService := service.NewAnalyticsServiceImpl(analyticsRepo)

	pc := &controller.ItemQueryController{
		ItemQueryService: service.NewItemQueryServiceImpl(
			itemQueryRepo,
			analyticsService,
		),
		ItemDetailedService: service.NewItemDetailedServiceImpl(
			itemDetailedRepo),
	}

	group.GET("/queries/items/initial", pc.FindShort)

	group.GET("/queries/items/detailed", pc.FindDetailed)

}
