package route

import (
	"database/sql"
	"miltechserver/api/controller"
	"miltechserver/api/repository"
	"miltechserver/api/service"
	"miltechserver/bootstrap"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/gin-gonic/gin"
)

func NewUserSavesRouter(db *sql.DB, blobClient *azblob.Client, env *bootstrap.Env, group *gin.RouterGroup) {
	userSavesRepository := repository.NewUserSavesRepositoryImpl(db, blobClient, env)

	pc := &controller.UserSavesController{
		UserSavesService: service.NewUserSavesServiceImpl(
			userSavesRepository, blobClient),
	}

	group.GET("/user/saves/quick_items", pc.GetQuickSaveItemsByUser)
	group.PUT("/user/saves/quick_items/add", pc.UpsertQuickSaveItemByUser)
	group.PUT("/user/saves/quick_items/addlist", pc.UpsertQuickSaveItemListByUser)
	group.DELETE("/user/saves/quick_items", pc.DeleteQuickSaveItemByUser)
	group.DELETE("/user/saves/quick_items/all", pc.DeleteAllQuickSaveItemsByUser)

	group.GET("/user/saves/serialized_items", pc.GetSerializedItemsByUser)
	group.PUT("/user/saves/serialized_items/add", pc.UpsertSerializedSaveItemByUser)
	group.PUT("/user/saves/serialized_items/addlist", pc.UpsertSerializedSaveItemListByUser)
	group.DELETE("/user/saves/serialized_items", pc.DeleteSerializedSaveItemByUser)
	group.DELETE("/user/saves/serialized_items/all", pc.DeleteAllSerializedItemsByUser)

	group.GET("/user/saves/item_category", pc.GetItemCategoriesByUser)
	group.PUT("/user/saves/item_category", pc.UpsertItemCategoryByUser)
	group.DELETE("/user/saves/item_category", pc.DeleteItemCategory)

	group.GET("/user/saves/categorized_items/category", pc.GetCategorizedItemsByCategory)
	group.GET("/user/saves/categorized_items", pc.GetCategorizedItemsByUser)
	group.PUT("/user/saves/categorized_items/add", pc.UpsertCategorizedItemByUser)
	group.PUT("/user/saves/categorized_items/addlist", pc.UpsertCategorizedItemListByUser)
	group.DELETE("/user/saves/categorized_items", pc.DeleteCategorizedItemByCategoryId)

	// Image management routes
	group.POST("/user/saves/items/image/upload/:table_type", pc.UploadItemImage)
	group.DELETE("/user/saves/items/image/:table_type", pc.DeleteItemImage)
	group.GET("/user/saves/items/image/:table_type", pc.GetItemImage)
}
