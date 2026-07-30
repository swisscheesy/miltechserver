package owned

import (
	"github.com/gin-gonic/gin"

	"miltechserver/api/user_pmcs/shared"
)

func RegisterRoutes(
	authGroup *gin.RouterGroup,
	service Service,
	config shared.Config,
) {
	handler := Handler{service: service, config: config}
	group := authGroup.Group("/user-pmcs")
	group.GET("/checklists/:checklist_id", handler.get)
	group.PUT("/checklists/:checklist_id", handler.create)
	group.PUT(
		"/checklists/:checklist_id/drafts/:revision_id",
		handler.putDraft,
	)
	group.DELETE(
		"/checklists/:checklist_id/drafts/:revision_id",
		handler.deleteDraft,
	)
	group.PUT(
		"/checklists/:checklist_id/publications/:revision_id",
		handler.publish,
	)
	group.GET(
		"/checklists/:checklist_id/revisions/:revision_id",
		handler.getRevision,
	)
}
