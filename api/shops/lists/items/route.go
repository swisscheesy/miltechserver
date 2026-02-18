package items

import (
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(router *gin.RouterGroup, service Service) {
	handler := Handler{service: service}
	router.POST("/shops/lists/items", handler.AddListItem)
	router.GET("/shops/lists/:list_id/items", handler.GetListItems)
	router.PUT("/shops/lists/items", handler.UpdateListItem)
	router.DELETE("/shops/lists/items", handler.RemoveListItem)
	router.POST("/shops/lists/items/bulk", handler.AddListItemBatch)
	router.DELETE("/shops/lists/items/bulk", handler.RemoveListItemBatch)
}
