package repository

import (
	"github.com/gin-gonic/gin"
	"miltechserver/miltech_ng/public/model"
)

type ItemQueryRepository interface {
	ShortItemSearchNiin(ctx *gin.Context, niin string) (model.NiinLookup, error)
	ShortItemSearchPart(ctx *gin.Context, part string) ([]model.NiinLookup, error)

	//DetailedItemSearchNiin(ctx *gin.Context, niin string) (model.DetailedItem, error)
	//
	////GetAmdfData(ctx context.Context, niin string) (db.ArmyMasterDataFileModel, error)
	//
	//DoesAmdfExist(ctx *gin.Context, niin string) (bool, error)
	//DoesFlisExist(ctx *gin.Context, niin string) (bool, error)
}
