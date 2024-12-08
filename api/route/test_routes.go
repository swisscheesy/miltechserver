package route

import (
	"github.com/gin-gonic/gin"
	"miltechserver/prisma/db"
	"net/http"
)

func NewTestRouter(db *db.PrismaClient, group *gin.RouterGroup) {
	//itemQueryRepo := repository.NewItemQueryRepositoryImpl(db)
	//itemDetailedRepo := repository.NewItemDetailedRepositoryImpl(db)
	/**pc := &controller.ItemQueryController{
		ItemQueryService: service.NewItemQueryServiceImpl(
			itemQueryRepo),
		ItemDetailedService: service.NewItemDetailedServiceImpl(
			itemDetailedRepo),
	} **/

	group.GET("/", func(c *gin.Context) {
		user, ok := c.Get("user")
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "You have access to this route", "user": user})
	})

	//router.GET("/item_query", func(c *gin.Context) {
	//	c.JSON(200, gin.H{
	//		"message": "Hello World",
	//	})
	//})
}
