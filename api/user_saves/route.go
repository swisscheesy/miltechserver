package user_saves

import (
	"database/sql"
	"miltechserver/api/user_saves/categories"
	categoryitems "miltechserver/api/user_saves/categories/items"
	"miltechserver/api/user_saves/images"
	"miltechserver/api/user_saves/quick"
	"miltechserver/api/user_saves/serialized"
	"miltechserver/bootstrap"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/gin-gonic/gin"
)

type Dependencies struct {
	DB         *sql.DB
	BlobClient *azblob.Client
	Env        *bootstrap.Env
}

func RegisterRoutes(deps Dependencies, group *gin.RouterGroup) {
	imagesRepository := images.NewRepository(deps.DB, deps.BlobClient, deps.Env)
	imagesService := images.NewService(imagesRepository)
	quickRepository := quick.NewRepository(deps.DB)
	quickService := quick.NewService(quickRepository, imagesRepository)
	serializedRepository := serialized.NewRepository(deps.DB)
	serializedService := serialized.NewService(serializedRepository, imagesRepository)
	categoriesRepository := categories.NewRepository(deps.DB)
	categoryItemsRepository := categoryitems.NewRepository(deps.DB)
	categoriesService := categories.NewService(categoriesRepository, categoryItemsRepository, imagesRepository)
	categoryItemsService := categoryitems.NewService(categoryItemsRepository, imagesRepository)

	quick.RegisterRoutes(group, quickService)
	serialized.RegisterRoutes(group, serializedService)
	categories.RegisterRoutes(group, categoriesService)
	categoryitems.RegisterRoutes(group, categoryItemsService)
	images.RegisterRoutes(group, imagesService)
}
