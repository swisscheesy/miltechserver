package repository

import (
	"context"
	"miltechserver/model"
	"miltechserver/prisma/db"
)

type ItemQueryRepositoryImpl struct {
	Db *db.PrismaClient
}

func NewItemQueryRepositoryImpl(db *db.PrismaClient) *ItemQueryRepositoryImpl {
	return &ItemQueryRepositoryImpl{Db: db}
}

func (repo *ItemQueryRepositoryImpl) ShortItemSearchNiin(ctx context.Context, niin string) (model.ShortItem, error) {
	item, err := repo.Db.NiinLookup.FindFirst(db.NiinLookup.Niin.Equals(niin)).Exec(ctx)

	if err != nil {
		return model.ShortItem{}, err
	} else {
		name, _ := item.ItemName()
		itemNiin := item.Niin
		fsc, _ := item.Fsc()
		hasAmdfData, _ := item.HasAmdf()
		hasFlisData, _ := item.HasFlis()

		itemData := model.ShortItem{
			ItemName:    name,
			Niin:        itemNiin,
			Fsc:         fsc,
			HasAmdfData: hasAmdfData,
			HasFlisData: hasFlisData,
		}
		return itemData, nil
	}

}

func (repo *ItemQueryRepositoryImpl) ShortItemSearchPart(ctx context.Context, part string) ([]model.ShortItem, error) {
	parts, err := repo.Db.PartNumber.FindMany(db.PartNumber.PartNumber.Equals(part)).Exec(ctx)

	if err != nil {
		return []model.ShortItem{}, err
	} else {
		var items []model.ShortItem

		for _, item := range parts {
			curItemNiin, _ := item.Niin()
			curItemFsc, _ := item.Fsc()

			name, _ := item.ItemName()
			itemNiin := curItemNiin
			fsc := curItemFsc

			amdfData, _ := repo.DoesAmdfExist(ctx, curItemNiin)
			flisData, _ := repo.DoesFlisExist(ctx, curItemNiin)

			hasAmdfData := amdfData
			hasFlisData := flisData

			itemData := model.ShortItem{
				ItemName:    name,
				Niin:        itemNiin,
				Fsc:         fsc,
				HasAmdfData: hasAmdfData,
				HasFlisData: hasFlisData,
			}
			items = append(items, itemData)
		}
		return items, nil
	}
}

// DoesAmdfExist Helper function to query the database for the existence of AMDF data for a given NIIN
func (repo *ItemQueryRepositoryImpl) DoesAmdfExist(ctx context.Context, niin string) (bool, error) {
	amdf, err := repo.Db.ArmyMasterDataFile.FindFirst(db.ArmyMasterDataFile.Niin.Equals(niin)).Exec(ctx)

	if err != nil {
		return false, err
	} else {
		return amdf != nil, nil
	}
}

// DoesFlisExist Helper function to query the database for the existence of FLIS data for a given NIIN
func (repo *ItemQueryRepositoryImpl) DoesFlisExist(ctx context.Context, niin string) (bool, error) {
	flis, err := repo.Db.FlisManagementID.FindFirst(db.FlisManagementID.Niin.Equals(niin)).Exec(ctx)

	if err != nil {
		return false, err
	} else {
		return flis != nil, nil
	}
}

//func (repo *ItemQueryRepositoryImpl) DetailedItemSearchNiin(ctx context.Context, niin string) (response.StandardResponse, error) {
//	// Implementation here
//	return response.StandardResponse{}, nil
//}
