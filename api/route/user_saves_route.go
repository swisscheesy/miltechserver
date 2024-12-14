package route

import (
	"github.com/gin-gonic/gin"
	"miltechserver/prisma/db"
)

func NewUserSavesRouter(db *db.PrismaClient, group *gin.RouterGroup) {
	//userSavesRepository := repository.NewUserSavesRepositoryImpl(db)
	//
	//pc := &controller.UserSavesController{
	//	UserSavesService: service.NewUserSavesServiceImpl(
	//		userSavesRepository),
	//}

	//group.GET("/user/saves/items/quick/:id", func(c *gin.Context) {
	//	user, ok := c.Get("user")
	//	if !ok {
	//		c.JSON(401, gin.H{"message": "unauthorized"})
	//		return
	//	}
	//})

}
