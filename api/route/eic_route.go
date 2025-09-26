package route

import (
	"database/sql"
	"miltechserver/api/controller"
	"miltechserver/api/repository"
	"miltechserver/api/service"

	"github.com/gin-gonic/gin"
)

func NewEICRouter(db *sql.DB, group *gin.RouterGroup) {
	eicRepo := repository.NewEICRepositoryImpl(db)
	ec := &controller.EICController{
		EICService: service.NewEICServiceImpl(eicRepo),
	}

	group.GET("/eic/niin/:niin", ec.LookupByNIIN)
	group.GET("/eic/lin/:lin", ec.LookupByLIN)
	group.GET("/eic/fsc/:fsc", ec.LookupByFSCPaginated)
	group.GET("/eic/items", ec.LookupAllPaginated)
}
