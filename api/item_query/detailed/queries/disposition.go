package queries

import (
	"database/sql"

	"miltechserver/.gen/miltech_ng/public/table"
	"miltechserver/api/details"

	. "github.com/go-jet/jet/v2/postgres"
)

func GetDisposition(db *sql.DB, niin string) (details.Disposition, error) {
	disposition := details.Disposition{}

	dispositionStmt := SELECT(
		table.Disposition.AllColumns,
	).FROM(table.Disposition).
		WHERE(table.Disposition.Niin.EQ(String(niin)))

	err := dispositionStmt.Query(db, &disposition.Disposition)
	if err != nil {
		return details.Disposition{}, err
	}

	return disposition, nil
}
