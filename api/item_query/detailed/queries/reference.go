package queries

import (
	"database/sql"

	"miltechserver/.gen/miltech_ng/public/table"
	"miltechserver/api/details"

	. "github.com/go-jet/jet/v2/postgres"
)

func GetReference(db *sql.DB, niin string) (details.Reference, error) {
	reference := details.Reference{}

	flisReferenceStmt := SELECT(
		table.FlisIdentification.AllColumns,
	).FROM(table.FlisIdentification).
		WHERE(table.FlisIdentification.Niin.EQ(String(niin)))

	err := flisReferenceStmt.Query(db, &reference.FlisReference)
	if err != nil {
		return details.Reference{}, err
	}

	referenceAndPartNumberStmt := SELECT(
		table.FlisReference.AllColumns,
	).FROM(table.FlisReference).
		WHERE(table.FlisReference.Niin.EQ(String(niin)))

	err = referenceAndPartNumberStmt.Query(db, &reference.ReferenceAndPartNumber)
	if err != nil {
		return details.Reference{}, err
	}

	var cageCodes string
	for _, ref := range reference.ReferenceAndPartNumber {
		if cageCodes != "" {
			cageCodes += ","
		}
		cageCodes += ref.CageCode
	}

	if cageCodes != "" {
		cageAddressesStmt := SELECT(
			table.CageAddress.AllColumns,
		).FROM(table.CageAddress).
			WHERE(table.CageAddress.CageCode.IN(String(cageCodes)))

		err = cageAddressesStmt.Query(db, &reference.CageAddresses)
		if err != nil {
			return details.Reference{}, err
		}

		cageStatusAndTypeStmt := SELECT(
			table.CageStatusAndType.AllColumns,
		).FROM(table.CageStatusAndType).
			WHERE(table.CageStatusAndType.CageCode.IN(String(cageCodes)))

		err = cageStatusAndTypeStmt.Query(db, &reference.CageStatusAndType)
		if err != nil {
			return details.Reference{}, err
		}
	}

	return reference, nil
}
