package route

import (
	"database/sql"

	"github.com/gin-gonic/gin"

	"miltechserver/api/controller"
	"miltechserver/api/repository"
	"miltechserver/api/service"
)

func NewItemCommentsRouter(db *sql.DB, group *gin.RouterGroup, authGroup *gin.RouterGroup) {
	repo := repository.NewItemCommentsRepositoryImpl(db)
	svc := service.NewItemCommentsServiceImpl(repo)
	ctrl := controller.NewItemCommentsController(svc)

	// Public route
	group.GET("/items/:niin/comments", ctrl.GetCommentsByNiin)

	// Authenticated routes
	authGroup.POST("/items/:niin/comments", ctrl.CreateComment)
	authGroup.PUT("/items/:niin/comments/:comment_id", ctrl.UpdateComment)
	authGroup.DELETE("/items/:niin/comments/:comment_id", ctrl.DeleteComment)
	authGroup.POST("/items/:niin/comments/:comment_id/flags", ctrl.FlagComment)
}
