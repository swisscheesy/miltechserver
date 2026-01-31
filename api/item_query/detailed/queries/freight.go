package queries

import (
	"database/sql"

	"miltechserver/.gen/miltech_ng/public/table"
	"miltechserver/api/details"

	. "github.com/go-jet/jet/v2/postgres"
)

func GetFreight(db *sql.DB, niin string) (details.Freight, error) {
	freight := details.Freight{}

	freightStmt := SELECT(
		table.FlisFreight.AllColumns,
	).FROM(table.FlisFreight).
		WHERE(table.FlisFreight.Niin.EQ(String(niin)))

	err := freightStmt.Query(db, &freight.FlisFreight)
	if err != nil {
		return details.Freight{}, err
	}

	return freight, nil
}
