package repository

import (
	"database/sql"
	"errors"
	"github.com/gin-gonic/gin"
	. "github.com/go-jet/jet/v2/postgres"
	"miltechserver/miltech_ng/public/model"
	"miltechserver/miltech_ng/public/view"
	"miltechserver/prisma/db"
)

type ItemQueryRepositoryImpl struct {
	Db *sql.DB
}

func NewItemQueryRepositoryImpl(db *sql.DB) *ItemQueryRepositoryImpl {
	return &ItemQueryRepositoryImpl{Db: db}
}

func (repo *ItemQueryRepositoryImpl) ShortItemSearchNiin(ctx *gin.Context, niin string) (model.NiinLookup, error) {
	item := model.NiinLookup{}

	stmt := SELECT(
		view.NiinLookup.AllColumns).FROM(view.NiinLookup).WHERE(view.NiinLookup.Niin.EQ(String(niin)))

	err := stmt.Query(repo.Db, &item)

	if err != nil {
		return model.NiinLookup{}, err
	} else {
		return item, nil
	}

}

func (repo *ItemQueryRepositoryImpl) ShortItemSearchPart(ctx *gin.Context, part string) ([]model.NiinLookup, error) {

	var items []model.NiinLookup

	parts, err := repo.Db.PartNumber.FindMany(db.PartNumber.PartNumber.Equals(part)).Exec(ctx)

	if err != nil {
		return []model.ShortItem{}, err
	} else if len(parts) == 0 {
		return []model.ShortItem{}, errors.New(db.ErrNotFound.Error())
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

func (repo *ItemQueryRepositoryImpl) DetailedItemSearchNiin(ctx *gin.Context, niin string) (model.DetailedItem, error) {
	_, err := repo.GetAmdfData(ctx, niin)
	data := model.DetailedItem{
		//Amdf: amdf,
	}
	return data, err
}
