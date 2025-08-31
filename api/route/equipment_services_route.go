package route

import (
	"database/sql"
	"miltechserver/api/controller"
	"miltechserver/api/repository"
	"miltechserver/api/service"
	"miltechserver/bootstrap"

	"github.com/gin-gonic/gin"
)

func NewEquipmentServicesRouter(db *sql.DB, env *bootstrap.Env, group *gin.RouterGroup) {
	equipmentServicesRepository := repository.NewEquipmentServicesRepositoryImpl(db)
	shopsRepository := repository.NewShopsRepositoryImpl(db)
	
	equipmentServicesController := &controller.EquipmentServicesController{
		EquipmentServicesService: service.NewEquipmentServicesServiceImpl(equipmentServicesRepository, shopsRepository),
	}
	
	// Equipment Service CRUD Operations
	group.POST("/shops/:shop_id/equipment-services", equipmentServicesController.CreateEquipmentService)
	group.GET("/shops/:shop_id/equipment-services", equipmentServicesController.GetEquipmentServices) 
	group.GET("/shops/:shop_id/equipment-services/:service_id", equipmentServicesController.GetEquipmentServiceByID)
	group.PUT("/shops/:shop_id/equipment-services/:service_id", equipmentServicesController.UpdateEquipmentService)
	group.DELETE("/shops/:shop_id/equipment-services/:service_id", equipmentServicesController.DeleteEquipmentService)
	
	// Service Completion
	group.POST("/shops/:shop_id/equipment-services/:service_id/complete", equipmentServicesController.CompleteEquipmentService)
	
	// Equipment-specific Services
	group.GET("/shops/:shop_id/equipment/:equipment_id/services", equipmentServicesController.GetServicesByEquipment)
	
	// Calendar and Status Operations
	group.GET("/shops/:shop_id/equipment-services/calendar", equipmentServicesController.GetServicesInDateRange)
	group.GET("/shops/:shop_id/equipment-services/overdue", equipmentServicesController.GetOverdueServices)
	group.GET("/shops/:shop_id/equipment-services/due-soon", equipmentServicesController.GetServicesDueSoon)
}