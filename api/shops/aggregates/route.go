package aggregates

import (
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(router *gin.RouterGroup, service Service) {
	handler := Handler{service: service}
	router.GET("/shops/:shop_id/lists-with-items", gzip.Gzip(gzip.DefaultCompression), handler.getListsWithItems)
}
