package route

import (
	"database/sql"
	"miltechserver/api/shops"
	"miltechserver/bootstrap"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/gin-gonic/gin"
)

func NewShopsRouter(db *sql.DB, blobClient *azblob.Client, env *bootstrap.Env, group *gin.RouterGroup) {
	shops.RegisterRoutes(shops.Dependencies{
		DB:         db,
		BlobClient: blobClient,
		Env:        env,
	}, group)
}
