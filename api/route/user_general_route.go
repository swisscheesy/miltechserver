package route

import (
	"database/sql"

	"miltechserver/api/controller"
	"miltechserver/api/repository"
	"miltechserver/api/service"

	"github.com/gin-gonic/gin"
)

func NewUserGeneralRouter(db *sql.DB, group *gin.RouterGroup) {
	userGeneralRepository := repository.NewUserGeneralRepositoryImpl(db)

	pc := &controller.UserGeneralController{
		UserGeneralService: service.NewUserGeneralServiceImpl(
			userGeneralRepository),
	}

	group.POST("/user/general/refresh", pc.UpsertUser)
}
