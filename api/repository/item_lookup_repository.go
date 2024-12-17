package repository

import (
	"github.com/gin-gonic/gin"
	"miltechserver/api/response"
	"miltechserver/prisma/db"
)

type ItemLookupRepository interface {
	SearchLINByPage(ctx *gin.Context, page int) (response.LINPageResponse, error)
	SearchLINByNIIN(ctx *gin.Context, niin string) ([]db.LookupLinNiinModel, error)
	SearchNIINByLIN(ctx *gin.Context, lin string) ([]db.LookupLinNiinModel, error)

	SearchUOCByPage(ctx *gin.Context, page int) (response.UOCPageResponse, error)
	SearchSpecificUOC(ctx *gin.Context, uoc string) ([]db.LookupUocModel, error)
	SearchUOCByModel(ctx *gin.Context, model string) ([]db.LookupUocModel, error)
}
