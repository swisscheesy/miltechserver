package sync

import (
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"

	"miltechserver/api/user_pmcs/shared"
)

func RegisterRoutes(
	authGroup *gin.RouterGroup,
	service Service,
	config shared.Config,
) {
	handler := Handler{service: service, config: config}
	authGroup.GET(
		"/user-pmcs/sync",
		gzip.Gzip(gzip.DefaultCompression),
		handler.getDelta,
	)
}
