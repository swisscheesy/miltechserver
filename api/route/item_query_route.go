package route

import (
	"github.com/gin-gonic/gin"
	"miltechserver/api/controller"
	"miltechserver/api/repository"
	"miltechserver/prisma/db"
	"miltechserver/service"
)

func NewItemQueryRouter(db *db.PrismaClient, group *gin.RouterGroup) {
	ur := repository.NewItemQueryRepositoryImpl(db)
	pc := &controller.ItemQueryController{
		ItemQueryService: service.NewItemQueryServiceImpl(
			ur),
	}
	group.GET("/item_query/niin", pc.FindShortByNiin)

	//router.GET("/item_query", func(c *gin.Context) {
	//	c.JSON(200, gin.H{
	//		"message": "Hello World",
	//	})
	//})
}
