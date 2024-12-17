package route

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"miltechserver/api/controller"
	"miltechserver/api/repository"
	"miltechserver/api/service"
)

func NewItemQueryRouter(db *gorm.DB, group *gin.RouterGroup) {
	itemQueryRepo := repository.NewItemQueryRepositoryImpl(db)
	itemDetailedRepo := repository.NewItemDetailedRepositoryImpl(db)
	pc := &controller.ItemQueryController{
		ItemQueryService: service.NewItemQueryServiceImpl(
			itemQueryRepo),
		ItemDetailedService: service.NewItemDetailedServiceImpl(
			itemDetailedRepo),
	}
	group.GET("/queries/items/initial", pc.FindShort)
	group.GET("/queries/items/detailed", pc.FindDetailed)

}
