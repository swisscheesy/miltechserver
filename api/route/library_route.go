package route

import (
	"database/sql"

	"firebase.google.com/go/v4/auth"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/gin-gonic/gin"

	"miltechserver/api/controller"
	"miltechserver/api/repository"
	"miltechserver/api/service"
	"miltechserver/bootstrap"
)

func NewLibraryRouter(
	db *sql.DB,
	blobClient *azblob.Client,
	blobCredential *azblob.SharedKeyCredential,
	env *bootstrap.Env,
	authClient *auth.Client,
	group *gin.RouterGroup,
	authGroup *gin.RouterGroup,
) {
	// Initialize repository (currently unused but follows pattern)
	repo := repository.NewLibraryRepositoryImpl(db)
	_ = repo // Silence unused variable warning

	analyticsRepo := repository.NewAnalyticsRepositoryImpl(db)
	analyticsService := service.NewAnalyticsServiceImpl(analyticsRepo)

	// Initialize service with blob client and credential (needed for SAS generation)
	svc := service.NewLibraryServiceImpl(blobClient, blobCredential, env, analyticsService)

	// Initialize controller
	ctrl := controller.NewLibraryController(svc)

	// Public routes (no authentication required)
	group.GET("/library/pmcs/vehicles", ctrl.GetPMCSVehicles)
	group.GET("/library/pmcs/:vehicle/documents", ctrl.GetPMCSDocuments)
	group.GET("/library/download", ctrl.GenerateDownloadURL)

	// Future public routes:
	// group.GET("/library/bii/categories", ctrl.GetBIICategories)
	// group.GET("/library/bii/:category/documents", ctrl.GetBIIDocuments)

	// Future authenticated routes (downloads, favorites, etc.):
	// authGroup.POST("/library/favorites", ctrl.AddFavorite)
	// authGroup.DELETE("/library/favorites/:document_path", ctrl.RemoveFavorite)
	// authGroup.GET("/library/favorites", ctrl.GetUserFavorites)
	// authGroup.GET("/library/download/:path", ctrl.GenerateDownloadURL)
}
