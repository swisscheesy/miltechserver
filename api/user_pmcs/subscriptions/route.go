package subscriptions

import "github.com/gin-gonic/gin"

func RegisterRoutes(authGroup *gin.RouterGroup, service Service) {
	handler := Handler{service: service}
	group := authGroup.Group("/user-pmcs/subscriptions")
	group.PUT("/:checklist_id", handler.install)
	group.DELETE("/:checklist_id", handler.unsubscribe)
	group.GET("/:checklist_id/installed-releases/:revision_id", handler.getInstalledRelease)
}
