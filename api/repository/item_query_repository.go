package repository

import (
	"github.com/gin-gonic/gin"
	"miltechserver/model"
)

type ItemQueryRepository interface {
	ShortItemSearchNiin(ctx *gin.Context, niin string) (model.ShortItem, error)
	ShortItemSearchPart(ctx *gin.Context, part string) ([]model.ShortItem, error)

	DetailedItemSearchNiin(ctx *gin.Context, niin string) (model.DetailedItem, error)

	//GetAmdfData(ctx context.Context, niin string) (db.ArmyMasterDataFileModel, error)

	DoesAmdfExist(ctx *gin.Context, niin string) (bool, error)
	DoesFlisExist(ctx *gin.Context, niin string) (bool, error)
}
