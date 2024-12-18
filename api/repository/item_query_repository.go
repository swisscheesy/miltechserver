package repository

import (
	"miltechserver/.gen/miltech_ng/public/model"
)

type ItemQueryRepository interface {
	ShortItemSearchNiin(niin string) (model.NiinLookup, error)
	ShortItemSearchPart(part string) ([]model.NiinLookup, error)

	//DetailedItemSearchNiin(ctx *gin.Context, niin string) (model.DetailedItem, error)
	//
	////GetAmdfData(ctx context.Context, niin string) (db.ArmyMasterDataFileModel, error)
	//
	//DoesAmdfExist(ctx *gin.Context, niin string) (bool, error)
	//DoesFlisExist(ctx *gin.Context, niin string) (bool, error)
}
