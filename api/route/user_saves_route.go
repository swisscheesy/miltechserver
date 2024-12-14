package route

import (
	"github.com/gin-gonic/gin"
	"miltechserver/api/controller"
	"miltechserver/api/repository"
	"miltechserver/api/service"
	"miltechserver/prisma/db"
)

func NewUserSavesRouter(db *db.PrismaClient, group *gin.RouterGroup) {
	userSavesRepository := repository.NewUserSavesRepositoryImpl(db)

	pc := &controller.UserSavesController{
		UserSavesService: service.NewUserSavesServiceImpl(
			userSavesRepository),
	}

	group.GET("/user/saves/items/quick/:id", pc.GetQuickSaveItemsByUser)

}
