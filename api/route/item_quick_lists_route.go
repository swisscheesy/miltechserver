package route

import (
	"database/sql"
	"miltechserver/api/controller"
	"miltechserver/api/repository"
	"miltechserver/api/service"

	"github.com/gin-gonic/gin"
)

func NewItemQuickListsRouter(db *sql.DB, group *gin.RouterGroup) {
	quickListsRepo := repository.NewItemQuickListsRepositoryImpl(db)

	pc := &controller.QuickListsController{
		QuickListsService: &service.ItemQuickListsServiceImpl{
			ItemQuickListsRepository: quickListsRepo,
		},
	}

	group.GET("/quick-lists/clothing", pc.QueryQuickListClothing)
	group.GET("/quick-lists/wheels", pc.QueryQuickListWheels)
	group.GET("/quick-lists/batteries", pc.QueryQuickListBatteries)
}
