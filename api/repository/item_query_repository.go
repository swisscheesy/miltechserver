package repository

import (
	"context"
	"miltechserver/model"
)

type ItemQueryRepository interface {
	ShortItemSearchNiin(ctx context.Context, niin string) (model.ShortItem, error)
	ShortItemSearchPart(ctx context.Context, part string) ([]model.ShortItem, error)

	DetailedItemSearchNiin(ctx context.Context, niin string) (model.DetailedItem, error)

	//GetAmdfData(ctx context.Context, niin string) (db.ArmyMasterDataFileModel, error)

	DoesAmdfExist(ctx context.Context, niin string) (bool, error)
	DoesFlisExist(ctx context.Context, niin string) (bool, error)
}
