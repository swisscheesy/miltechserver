package queries

import (
	"database/sql"

	"miltechserver/.gen/miltech_ng/public/table"
	"miltechserver/api/details"

	. "github.com/go-jet/jet/v2/postgres"
)

func GetAmdfData(db *sql.DB, niin string) (details.Amdf, error) {
	amdf := details.Amdf{}

	amdfStmt := SELECT(
		table.ArmyMasterDataFile.AllColumns,
	).FROM(table.ArmyMasterDataFile).
		WHERE(table.ArmyMasterDataFile.Niin.EQ(String(niin)))

	err := amdfStmt.Query(db, &amdf.ArmyMasterDataFile)
	if err != nil {
		return details.Amdf{}, err
	}

	amdfManagementStmt := SELECT(
		table.AmdfManagement.AllColumns,
	).FROM(table.AmdfManagement).
		WHERE(table.AmdfManagement.Niin.EQ(String(niin)))

	err = amdfManagementStmt.Query(db, &amdf.AmdfManagement)
	if err != nil {
		return details.Amdf{}, err
	}

	amdfCreditStmt := SELECT(
		table.AmdfCredit.AllColumns,
	).FROM(table.AmdfCredit).
		WHERE(table.AmdfCredit.Niin.EQ(String(niin)))

	err = amdfCreditStmt.Query(db, &amdf.AmdfCredit)
	if err != nil {
		return details.Amdf{}, err
	}

	amdfBillingStmt := SELECT(
		table.AmdfBilling.AllColumns,
	).FROM(table.AmdfBilling).
		WHERE(table.AmdfBilling.Niin.EQ(String(niin)))

	err = amdfBillingStmt.Query(db, &amdf.AmdfBilling)
	if err != nil {
		return details.Amdf{}, err
	}

	amdfMatcatStmt := SELECT(
		table.AmdfMatcat.AllColumns,
	).FROM(table.AmdfMatcat).
		WHERE(table.AmdfMatcat.Niin.EQ(String(niin)))

	err = amdfMatcatStmt.Query(db, &amdf.AmdfMatcat)
	if err != nil {
		return details.Amdf{}, err
	}

	amdfPhrasesStmt := SELECT(
		table.AmdfPhrase.AllColumns,
	).FROM(table.AmdfPhrase).
		WHERE(table.AmdfPhrase.Niin.EQ(String(niin)))

	err = amdfPhrasesStmt.Query(db, &amdf.AmdfPhrases)
	if err != nil {
		return details.Amdf{}, err
	}

	amdfIandSStmt := SELECT(
		table.AmdfIAndS.AllColumns,
	).FROM(table.AmdfIAndS).
		WHERE(table.AmdfIAndS.Niin.EQ(String(niin)))

	err = amdfIandSStmt.Query(db, &amdf.AmdfIandS)
	if err != nil {
		return details.Amdf{}, err
	}

	if amdf.AmdfManagement.Lin != nil {
		armyLinStmt := SELECT(
			table.ArmyLineItemNumber.AllColumns,
		).FROM(table.ArmyLineItemNumber).
			WHERE(table.ArmyLineItemNumber.Lin.EQ(String(*amdf.AmdfManagement.Lin)))

		err = armyLinStmt.Query(db, &amdf.ArmyLin)
		if err != nil {
			return details.Amdf{}, err
		}
	}

	return amdf, nil
}
