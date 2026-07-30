package community

import "github.com/gin-gonic/gin"

func RegisterRoutes(authGroup *gin.RouterGroup, service Service) {
	handler := Handler{service: service}
	group := authGroup.Group("/user-pmcs")
	group.PUT(
		"/checklists/:checklist_id/community-releases/:revision_id",
		handler.release,
	)
	group.DELETE(
		"/checklists/:checklist_id/community-source",
		handler.retire,
	)
}
