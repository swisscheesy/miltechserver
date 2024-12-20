package route

import (
	"database/sql"
	"github.com/gin-gonic/gin"
	"miltechserver/api/controller"
	"miltechserver/api/repository"
	"miltechserver/api/service"
)

func NewItemLookupRouter(db *sql.DB, group *gin.RouterGroup) {
	itemLookupRepo := repository.NewItemLookupRepositoryImpl(db)

	pc := &controller.ItemLookupController{
		ItemLookupService: service.NewItemLookupServiceImpl(
			itemLookupRepo),
	}
	group.GET("/lookup/lin", pc.LookupLINByPage)
	group.GET("/lookup/lin/:niin", pc.LookupLINByNIIN)
	group.GET("/lookup/niin/:lin", pc.LookupNIINByLIN)

	group.GET("/lookup/uoc", pc.LookupUOCByPage)
	group.GET("/lookup/uoc/:uoc", pc.LookupSpecificUOC)
	group.GET("/lookup/uoc/model/:model", pc.LookupUOCByModel)

}
