package queries

import (
	"database/sql"

	"miltechserver/.gen/miltech_ng/public/table"
	"miltechserver/api/details"

	. "github.com/go-jet/jet/v2/postgres"
)

func GetCharacteristics(db *sql.DB, niin string) (details.Characteristics, error) {
	characteristics := details.Characteristics{}

	characteristicsStmt := SELECT(
		table.FlisItemCharacteristics.AllColumns,
	).FROM(table.FlisItemCharacteristics).
		WHERE(table.FlisItemCharacteristics.Niin.EQ(String(niin)))

	err := characteristicsStmt.Query(db, &characteristics.Characteristics)
	if err != nil {
		return details.Characteristics{}, err
	}

	return characteristics, nil
}
