package material_images

import (
	"database/sql"

	"firebase.google.com/go/v4/auth"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/gin-gonic/gin"

	"miltechserver/api/material_images/flags"
	"miltechserver/api/material_images/images"
	"miltechserver/api/material_images/ratelimit"
	"miltechserver/api/material_images/votes"
	"miltechserver/bootstrap"
)

type Dependencies struct {
	DB         *sql.DB
	BlobClient *azblob.Client
	Env        *bootstrap.Env
	AuthClient *auth.Client
}

func RegisterRoutes(deps Dependencies, publicRouter *gin.RouterGroup, authRouter *gin.RouterGroup) {
	rateLimitRepo := ratelimit.NewRepository(deps.DB)
	imagesRepo := images.NewRepository(deps.DB)
	votesRepo := votes.NewRepository(deps.DB)
	flagsRepo := flags.NewRepository(deps.DB)

	imagesService := images.NewService(imagesRepo, rateLimitRepo, votesRepo, deps.BlobClient, deps.Env)
	votesService := votes.NewService(votesRepo, imagesRepo)
	flagsService := flags.NewService(flagsRepo, imagesRepo)

	images.RegisterRoutes(publicRouter, authRouter, imagesService, deps.AuthClient)
	votes.RegisterRoutes(authRouter, votesService, imagesService)
	flags.RegisterRoutes(authRouter, flagsService, imagesService)
}
