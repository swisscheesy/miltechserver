package items

import (
	"miltechserver/api/controller"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(router *gin.RouterGroup, controller *controller.ShopsController) {
	router.POST("/shops/lists/items", controller.AddListItem)
	router.GET("/shops/lists/:list_id/items", controller.GetListItems)
	router.PUT("/shops/lists/items", controller.UpdateListItem)
	router.DELETE("/shops/lists/items", controller.RemoveListItem)
	router.POST("/shops/lists/items/bulk", controller.AddListItemBatch)
	router.DELETE("/shops/lists/items/bulk", controller.RemoveListItemBatch)
}
