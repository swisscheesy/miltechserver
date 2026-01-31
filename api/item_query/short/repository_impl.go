package short

import (
	"database/sql"
	"strings"

	"miltechserver/.gen/miltech_ng/public/model"
	. "miltechserver/.gen/miltech_ng/public/table"
	"miltechserver/.gen/miltech_ng/public/view"
	"miltechserver/api/item_query/shared"

	. "github.com/go-jet/jet/v2/postgres"
)

type RepositoryImpl struct {
	Db *sql.DB
}

func NewRepository(db *sql.DB) *RepositoryImpl {
	return &RepositoryImpl{Db: db}
}

func (repo *RepositoryImpl) ShortItemSearchNiin(niin string) (model.NiinLookup, error) {
	item := model.NiinLookup{}

	stmt := SELECT(
		view.NiinLookup.AllColumns,
	).FROM(view.NiinLookup).
		WHERE(view.NiinLookup.Niin.EQ(String(niin)))

	err := stmt.Query(repo.Db, &item)
	if err != nil || item.Niin == nil || *item.Niin == "" {
		return model.NiinLookup{}, shared.ErrNoItemsFound
	}

	return item, nil
}

func (repo *RepositoryImpl) ShortItemSearchPart(part string) ([]model.NiinLookup, error) {
	var items []model.NiinLookup

	stmt := SELECT(
		view.NiinLookup.AllColumns,
	).FROM(
		view.NiinLookup.
			INNER_JOIN(PartNumber, view.NiinLookup.Niin.EQ(PartNumber.Niin)),
	).WHERE(PartNumber.PartNumber.EQ(String(strings.ToUpper(part))))

	err := stmt.Query(repo.Db, &items)
	if err != nil || len(items) == 0 {
		return []model.NiinLookup{}, shared.ErrNoItemsFound
	}

	return items, nil
}
