package queries

import (
	"database/sql"

	"miltechserver/.gen/miltech_ng/public/table"
	"miltechserver/api/details"

	. "github.com/go-jet/jet/v2/postgres"
)

func GetArmyPackagingAndFreight(db *sql.DB, niin string) (details.ArmyPackagingAndFreight, error) {
	armyPackagingAndFreight := details.ArmyPackagingAndFreight{}

	armyPackagingAndFreightStmt := SELECT(
		table.ArmyPackagingAndFreight.AllColumns,
	).FROM(table.ArmyPackagingAndFreight).
		WHERE(table.ArmyPackagingAndFreight.Niin.EQ(String(niin)))

	err := armyPackagingAndFreightStmt.Query(db, &armyPackagingAndFreight.ArmyPackagingAndFreight)
	if err != nil {
		return details.ArmyPackagingAndFreight{}, err
	}

	armyPackaging1Stmt := SELECT(
		table.ArmyPackaging1.AllColumns,
	).FROM(table.ArmyPackaging1).
		WHERE(table.ArmyPackaging1.Niin.EQ(String(niin)))

	err = armyPackaging1Stmt.Query(db, &armyPackagingAndFreight.ArmyPackaging1)
	if err != nil {
		return details.ArmyPackagingAndFreight{}, err
	}

	armyPackaging2Stmt := SELECT(
		table.ArmyPackaging2.AllColumns,
	).FROM(table.ArmyPackaging2).
		WHERE(table.ArmyPackaging2.Niin.EQ(String(niin)))

	err = armyPackaging2Stmt.Query(db, &armyPackagingAndFreight.ArmyPackaging2)
	if err != nil {
		return details.ArmyPackagingAndFreight{}, err
	}

	armyPackSpecialInstructStmt := SELECT(
		table.ArmyPackagingSpecialInstruct.AllColumns,
	).FROM(table.ArmyPackagingSpecialInstruct).
		WHERE(table.ArmyPackagingSpecialInstruct.Niin.EQ(String(niin)))

	err = armyPackSpecialInstructStmt.Query(db, &armyPackagingAndFreight.ArmyPackSpecialInstruct)
	if err != nil {
		return details.ArmyPackagingAndFreight{}, err
	}

	armyFreightStmt := SELECT(
		table.ArmyFreight.AllColumns,
	).FROM(table.ArmyFreight).
		WHERE(table.ArmyFreight.Niin.EQ(String(niin)))

	err = armyFreightStmt.Query(db, &armyPackagingAndFreight.ArmyFreight)
	if err != nil {
		return details.ArmyPackagingAndFreight{}, err
	}

	armyPackSupplementalInstructStmt := SELECT(
		table.ArmyPackSupplementalInstruct.AllColumns,
	).FROM(table.ArmyPackSupplementalInstruct).
		WHERE(table.ArmyPackSupplementalInstruct.Niin.EQ(String(niin)))

	err = armyPackSupplementalInstructStmt.Query(db, &armyPackagingAndFreight.ArmyPackSupplementalInstruct)
	if err != nil {
		return details.ArmyPackagingAndFreight{}, err
	}

	return armyPackagingAndFreight, nil
}

func GetPackaging(db *sql.DB, niin string) (details.Packaging, error) {
	packaging := details.Packaging{}

	pack1Stmt := SELECT(
		table.FlisPackaging1.AllColumns,
	).FROM(table.FlisPackaging1).
		WHERE(table.FlisPackaging1.Niin.EQ(String(niin)))

	err := pack1Stmt.Query(db, &packaging.FlisPackaging1)
	if err != nil {
		return details.Packaging{}, err
	}

	var cageCodes string
	for _, ref := range packaging.FlisPackaging1 {
		if ref.PkgDesignActy != nil {
			if cageCodes != "" {
				cageCodes += ","
			}
			cageCodes += *ref.PkgDesignActy
		}
	}

	pack2Stmt := SELECT(
		table.FlisPackaging2.AllColumns,
	).FROM(table.FlisPackaging2).
		WHERE(table.FlisPackaging2.Niin.EQ(String(niin)))

	err = pack2Stmt.Query(db, &packaging.FlisPackaging2)
	if err != nil {
		return details.Packaging{}, err
	}

	if cageCodes != "" {
		cageAddressStmt := SELECT(
			table.CageAddress.AllColumns,
		).FROM(table.CageAddress).
			WHERE(table.CageAddress.CageCode.IN(String(cageCodes)))

		err = cageAddressStmt.Query(db, &packaging.CageAddress)
		if err != nil {
			return details.Packaging{}, err
		}
	}

	dssWeightStmt := SELECT(
		table.DssWeightAndCube.AllColumns,
	).FROM(table.DssWeightAndCube).
		WHERE(table.DssWeightAndCube.Niin.EQ(String(niin)))

	err = dssWeightStmt.Query(db, &packaging.DssWeightAndCube)
	if err != nil {
		return details.Packaging{}, err
	}

	return packaging, nil
}
