package community

import (
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
)

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
	group.GET(
		"/community",
		gzip.Gzip(gzip.DefaultCompression),
		handler.browseAuthenticated,
	)
	group.PUT("/community/:checklist_id/vote", handler.putVote)
	group.DELETE("/community/:checklist_id/vote", handler.deleteVote)
}

func RegisterPublicRoutes(publicGroup *gin.RouterGroup, service Service) {
	handler := Handler{service: service}
	group := publicGroup.Group("/user-pmcs/community")
	group.GET(
		"",
		gzip.Gzip(gzip.DefaultCompression),
		handler.browse,
	)
	group.GET(
		"/:checklist_id",
		gzip.Gzip(gzip.DefaultCompression),
		handler.getCurrentRelease,
	)
}
