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

func NewMaterialImagesRouter(db *sql.DB, blobClient *azblob.Client, env *bootstrap.Env, authClient *auth.Client, group *gin.RouterGroup, authGroup *gin.RouterGroup) {
	// Initialize repository
	repo := repository.NewMaterialImagesRepositoryImpl(db)

	// Initialize service
	svc := service.NewMaterialImagesServiceImpl(repo, blobClient, env)

	// Initialize controller
	ctrl := controller.NewMaterialImagesController(svc)

	// Public routes (no auth required)
	group.GET("/material-images/niin/:niin", ctrl.GetImagesByNIIN)
	group.GET("/material-images/:image_id", ctrl.GetImageByID)

	// Protected routes (auth required)

	// Upload and management
	authGroup.POST("/material-images/upload", ctrl.UploadImage)
	authGroup.DELETE("/material-images/:image_id", ctrl.DeleteImage)
	authGroup.GET("/material-images/user/:user_id", ctrl.GetImagesByUser)

	// Voting
	authGroup.POST("/material-images/:image_id/vote", ctrl.VoteOnImage)
	authGroup.DELETE("/material-images/:image_id/vote", ctrl.RemoveVote)

	// Flagging
	authGroup.POST("/material-images/:image_id/flag", ctrl.FlagImage)

	// Admin routes (TODO: Add admin middleware)
	authGroup.GET("/material-images/:image_id/flags", ctrl.GetImageFlags)

}
