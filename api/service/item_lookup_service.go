package service

import (
	"github.com/gin-gonic/gin"
	"miltechserver/model"
	"miltechserver/prisma/db"
)

type ItemLookupService interface {
	LookupLINByPage(ctx *gin.Context, page int) (model.LINPageResponse, error)
	LookupLINByNIIN(ctx *gin.Context, niin string) ([]db.LookupLinNiinModel, error)
	LookupNIINByLIN(ctx *gin.Context, niin string) ([]db.LookupLinNiinModel, error)

	LookupUOCByPage(ctx *gin.Context, page int) (model.UOCPageResponse, error)
	LookupSpecificUOC(ctx *gin.Context, uoc string) ([]db.LookupUocModel, error)
	LookupUOCByModel(ctx *gin.Context, model string) ([]db.LookupUocModel, error)
}
