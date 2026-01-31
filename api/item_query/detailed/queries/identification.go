package queries

import (
	"database/sql"

	"miltechserver/.gen/miltech_ng/public/table"
	"miltechserver/api/details"

	. "github.com/go-jet/jet/v2/postgres"
)

func GetIdentification(db *sql.DB, niin string) (details.Identification, error) {
	identification := details.Identification{}

	flisMgmtStmt := SELECT(
		table.FlisManagementID.AllColumns,
	).FROM(table.FlisManagementID).
		WHERE(table.FlisManagementID.Niin.EQ(String(niin)))

	err := flisMgmtStmt.Query(db, &identification.FlisManagementId)
	if err != nil {
		return details.Identification{}, err
	}

	if identification.FlisManagementId.Inc != nil {
		colloquialNamesStmt := SELECT(
			table.ColloquialName.AllColumns,
		).FROM(table.ColloquialName).
			WHERE(table.ColloquialName.Inc.EQ(String(*identification.FlisManagementId.Inc)))

		err = colloquialNamesStmt.Query(db, &identification.ColloquialName)
		if err != nil {
			return details.Identification{}, err
		}
	}

	flisStandardizationStmt := SELECT(
		table.FlisStandardization.AllColumns,
	).FROM(table.FlisStandardization).
		WHERE(table.FlisStandardization.Niin.EQ(String(niin)))

	err = flisStandardizationStmt.Query(db, &identification.FlisStandardization)
	if err != nil {
		return details.Identification{}, err
	}

	flisCancelledNiinStmt := SELECT(
		table.FlisCancelledNiin.AllColumns,
	).FROM(table.FlisCancelledNiin).
		WHERE(table.FlisCancelledNiin.Niin.EQ(String(niin)))

	err = flisCancelledNiinStmt.Query(db, &identification.FlisCancelledNiin)
	if err != nil {
		return details.Identification{}, err
	}

	return identification, nil
}
