package queries

import (
	"database/sql"

	"miltechserver/.gen/miltech_ng/public/table"
	"miltechserver/api/details"

	. "github.com/go-jet/jet/v2/postgres"
)

func GetManagement(db *sql.DB, niin string) (details.Management, error) {
	management := details.Management{}

	flisManagementStmt := SELECT(
		table.FlisManagement.AllColumns,
	).FROM(table.FlisManagement).
		WHERE(table.FlisManagement.Niin.EQ(String(niin)))

	err := flisManagementStmt.Query(db, &management.FLisManagement)
	if err != nil {
		return details.Management{}, err
	}

	flisPhraseStmt := SELECT(
		table.FlisPhrase.AllColumns,
	).FROM(table.FlisPhrase).
		WHERE(table.FlisPhrase.Niin.EQ(String(niin)))

	err = flisPhraseStmt.Query(db, &management.FlisPhrase)
	if err != nil {
		return details.Management{}, err
	}

	componentEndItemStmt := SELECT(
		table.ComponentEndItem.AllColumns,
	).FROM(table.ComponentEndItem).
		WHERE(table.ComponentEndItem.Niin.EQ(String(niin)))

	err = componentEndItemStmt.Query(db, &management.ComponentEndItem)
	if err != nil {
		return details.Management{}, err
	}

	armyManagementStmt := SELECT(
		table.ArmyManagement.AllColumns,
	).FROM(table.ArmyManagement).
		WHERE(table.ArmyManagement.Niin.EQ(String(niin)))

	err = armyManagementStmt.Query(db, &management.ArmyManagement)
	if err != nil {
		return details.Management{}, err
	}

	airForceManagementStmt := SELECT(
		table.AirForceManagement.AllColumns,
	).FROM(table.AirForceManagement).
		WHERE(table.AirForceManagement.Niin.EQ(String(niin)))

	err = airForceManagementStmt.Query(db, &management.AirForceManagement)
	if err != nil {
		return details.Management{}, err
	}

	marineCorpsManagementStmt := SELECT(
		table.MarineCorpsManagement.AllColumns,
	).FROM(table.MarineCorpsManagement).
		WHERE(table.MarineCorpsManagement.Niin.EQ(String(niin)))

	err = marineCorpsManagementStmt.Query(db, &management.MarineCorpsManagement)
	if err != nil {
		return details.Management{}, err
	}

	navyManagementStmt := SELECT(
		table.NavyManagement.AllColumns,
	).FROM(table.NavyManagement).
		WHERE(table.NavyManagement.Niin.EQ(String(niin)))

	err = navyManagementStmt.Query(db, &management.NavyManagement)
	if err != nil {
		return details.Management{}, err
	}

	faaManagementStmt := SELECT(
		table.FaaManagement.AllColumns,
	).FROM(table.FaaManagement).
		WHERE(table.FaaManagement.Niin.EQ(String(niin)))

	err = faaManagementStmt.Query(db, &management.FaaManagement)
	if err != nil {
		return details.Management{}, err
	}

	return management, nil
}
