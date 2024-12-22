package route

import (
	"database/sql"
	"github.com/gin-gonic/gin"
	"miltechserver/api/controller"
	"miltechserver/api/repository"
	"miltechserver/api/service"
)

func NewUserSavesRouter(db *sql.DB, group *gin.RouterGroup) {
	userSavesRepository := repository.NewUserSavesRepositoryImpl(db)

	pc := &controller.UserSavesController{
		UserSavesService: service.NewUserSavesServiceImpl(
			userSavesRepository),
	}

	group.GET("/user/saves/quick_items", pc.GetQuickSaveItemsByUser)
	group.PUT("/user/saves/quick_items/add", pc.UpsertQuickSaveItemByUser)
	group.PUT("/user/saves/quick_items/addlist", pc.UpsertQuickSaveItemListByUser)
	group.DELETE("/user/saves/quick_items", pc.DeleteQuickSaveItemByUser)
	group.DELETE("/user/saves/quick_items/all", pc.DeleteAllQuickSaveItemsByUser)

	group.GET("/user/saves/serialized_items", pc.GetSerializedItemsByUser)

}
