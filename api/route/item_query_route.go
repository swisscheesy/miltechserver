package route

import (
	"github.com/gin-gonic/gin"
	"miltechserver/api/controller"
	"miltechserver/api/repository"
	"miltechserver/api/service"
	"miltechserver/prisma/db"
)

func NewItemQueryRouter(db *db.PrismaClient, group *gin.RouterGroup) {
	ur := repository.NewItemQueryRepositoryImpl(db)
	pc := &controller.ItemQueryController{
		ItemQueryService: service.NewItemQueryServiceImpl(
			ur),
	}
	group.GET("/queries/items/initial", pc.FindShort)
	group.GET("/queries/items/detailed", pc.FindDetailed)

	//router.GET("/item_query", func(c *gin.Context) {
	//	c.JSON(200, gin.H{
	//		"message": "Hello World",
	//	})
	//})
}
