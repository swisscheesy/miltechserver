package aggregates

import (
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(router *gin.RouterGroup, service Service) {
	handler := Handler{service: service}
	router.GET("/shops/bootstrap", gzip.Gzip(gzip.DefaultCompression), handler.getBootstrap)
	router.GET("/shops/:shop_id/snapshot", gzip.Gzip(gzip.DefaultCompression), handler.getShopSnapshot)
	router.GET("/shops/:shop_id/lists-with-items", gzip.Gzip(gzip.DefaultCompression), handler.getListsWithItems)
	router.GET("/shops/vehicles/:vehicle_id/maintenance-snapshot", gzip.Gzip(gzip.DefaultCompression), handler.getVehicleMaintenanceSnapshot)
	router.GET("/shops/equipment-pmcs-history", gzip.Gzip(gzip.DefaultCompression), handler.getEquipmentPmcsHistory)
}
