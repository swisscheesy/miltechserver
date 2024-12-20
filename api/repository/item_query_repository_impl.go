package repository

import (
	"database/sql"
	"errors"
	. "github.com/go-jet/jet/v2/postgres"
	"miltechserver/.gen/miltech_ng/public/model"
	. "miltechserver/.gen/miltech_ng/public/table"
	"miltechserver/.gen/miltech_ng/public/view"
)

type ItemQueryRepositoryImpl struct {
	Db *sql.DB
}

func NewItemQueryRepositoryImpl(db *sql.DB) *ItemQueryRepositoryImpl {
	return &ItemQueryRepositoryImpl{Db: db}
}

func (repo *ItemQueryRepositoryImpl) ShortItemSearchNiin(niin string) (model.NiinLookup, error) {
	item := model.NiinLookup{}

	stmt := SELECT(
		view.NiinLookup.AllColumns).FROM(view.NiinLookup).WHERE(view.NiinLookup.Niin.EQ(String(niin)))

	err := stmt.Query(repo.Db, &item)

	if err != nil || *item.Niin == "" {
		return model.NiinLookup{}, errors.New("no items found")
	} else {
		return item, nil
	}

}

func (repo *ItemQueryRepositoryImpl) ShortItemSearchPart(part string) ([]model.NiinLookup, error) {
	var items []model.NiinLookup

	stmt := SELECT(
		view.NiinLookup.AllColumns,
	).FROM(
		view.NiinLookup.
			INNER_JOIN(PartNumber, view.NiinLookup.Niin.EQ(PartNumber.Niin))).
		WHERE(PartNumber.PartNumber.EQ(String(part)))

	err := stmt.Query(repo.Db, &items)

	if err != nil || len(items) == 0 {
		return []model.NiinLookup{}, errors.New("no items found")
	} else {
		return items, nil
	}

}
