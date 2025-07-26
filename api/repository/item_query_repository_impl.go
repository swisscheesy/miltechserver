package repository

import (
	"database/sql"
	"errors"
	"miltechserver/.gen/miltech_ng/public/model"
	. "miltechserver/.gen/miltech_ng/public/table"
	"miltechserver/.gen/miltech_ng/public/view"
	"strings"

	. "github.com/go-jet/jet/v2/postgres"
)

type ItemQueryRepositoryImpl struct {
	Db *sql.DB
}

func NewItemQueryRepositoryImpl(db *sql.DB) *ItemQueryRepositoryImpl {
	return &ItemQueryRepositoryImpl{Db: db}
}

// ShortItemSearchNiin searches for a short item by NIIN (National Item Identification Number).
// \param niin - the NIIN to search for.
// \return a NiinLookup containing the item data.
// \return an error if the operation fails.
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

// ShortItemSearchPart searches for short items by part number.
// \param part - the part number to search for.
// \return a slice of NiinLookup containing the item data.
// \return an error if the operation fails.
func (repo *ItemQueryRepositoryImpl) ShortItemSearchPart(part string) ([]model.NiinLookup, error) {
	var items []model.NiinLookup

	stmt := SELECT(
		view.NiinLookup.AllColumns,
	).FROM(
		view.NiinLookup.
										INNER_JOIN(PartNumber, view.NiinLookup.Niin.EQ(PartNumber.Niin))).
		WHERE(PartNumber.PartNumber.EQ(String(strings.ToUpper(part)))) // Ensure part number is uppercase

	err := stmt.Query(repo.Db, &items)

	if err != nil || len(items) == 0 {
		return []model.NiinLookup{}, errors.New("no items found")
	} else {
		return items, nil
	}

}
