package repository

import (
	"database/sql"
	. "github.com/go-jet/jet/v2/postgres"
	"miltechserver/.gen/miltech_ng/public/table"
	"miltechserver/api/details"
	"miltechserver/api/response"
)

type ItemDetailedRepositoryImpl struct {
	Db *sql.DB
}

func NewItemDetailedRepositoryImpl(db *sql.DB) *ItemDetailedRepositoryImpl {
	return &ItemDetailedRepositoryImpl{Db: db}
}

func (repo *ItemDetailedRepositoryImpl) GetDetailedItemData(niin string) (response.DetailedResponse, error) {
	// Get data from each table
	// Call helper methods to get data from each table
	// Return a DetailedItem struct with all the data

	amdfData, _ := repo.getAmdfData(niin)

	armyPackData, _ := repo.getArmyPackagingAndFreight(niin)
	sarsscatData, _ := repo.getSarsscat(niin)

	identificationData, _ := repo.getIdentification(niin)

	managementData, _ := repo.getManagement(niin)

	referenceData, _ := repo.getReference(niin)

	freightData, _ := repo.getFreight(niin)

	packagingData, _ := repo.getPackaging(niin)

	characteristicsData, _ := repo.getCharacteristics(niin)

	dispositionData, _ := repo.getDisposition(niin)

	fullDetailedItem := response.DetailedResponse{
		Amdf:                    amdfData,
		ArmyPackagingAndFreight: armyPackData,
		Sarsscat:                sarsscatData,
		Identification:          identificationData,
		Management:              managementData,
		Reference:               referenceData,
		Freight:                 freightData,
		Packaging:               packagingData,
		Characteristics:         characteristicsData,
		Disposition:             dispositionData,
	}

	return fullDetailedItem, nil
}

func (repo *ItemDetailedRepositoryImpl) getAmdfData(niin string) (details.Amdf, error) {
	amdf := details.Amdf{}

	amdfStmt :=
		SELECT(
			table.ArmyMasterDataFile.AllColumns,
		).FROM(table.ArmyMasterDataFile).
			WHERE(table.ArmyMasterDataFile.Niin.EQ(String(niin)))

	err := amdfStmt.Query(repo.Db, &amdf.ArmyMasterDataFile)

	amdfManagementStmt := SELECT(
		table.AmdfManagement.AllColumns).FROM(table.AmdfManagement).
		WHERE(table.AmdfManagement.Niin.EQ(String(niin)))

	err = amdfManagementStmt.Query(repo.Db, &amdf.AmdfManagement)

	amdfCreditStmt := SELECT(
		table.AmdfCredit.AllColumns).FROM(table.AmdfCredit).
		WHERE(table.AmdfCredit.Niin.EQ(String(niin)))

	err = amdfCreditStmt.Query(repo.Db, &amdf.AmdfCredit)

	amdfBillingStmt := SELECT(
		table.AmdfBilling.AllColumns).FROM(table.AmdfBilling).
		WHERE(table.AmdfBilling.Niin.EQ(String(niin)))

	err = amdfBillingStmt.Query(repo.Db, &amdf.AmdfBilling)

	amdfMatcatStmt := SELECT(
		table.AmdfMatcat.AllColumns).FROM(table.AmdfMatcat).
		WHERE(table.AmdfMatcat.Niin.EQ(String(niin)))

	err = amdfMatcatStmt.Query(repo.Db, &amdf.AmdfMatcat)

	amdfPhrasesStmt := SELECT(
		table.AmdfPhrase.AllColumns).FROM(table.AmdfPhrase).
		WHERE(table.AmdfPhrase.Niin.EQ(String(niin)))

	err = amdfPhrasesStmt.Query(repo.Db, &amdf.AmdfPhrases)

	amdfIandSStmt := SELECT(
		table.AmdfIAndS.AllColumns).FROM(table.AmdfIAndS).
		WHERE(table.AmdfIAndS.Niin.EQ(String(niin)))

	err = amdfIandSStmt.Query(repo.Db, &amdf.AmdfIandS)

	// Ensure the item has LIN data, otherwise skip
	if amdf.AmdfManagement.Lin != nil {
		armyLinStmt := SELECT(
			table.ArmyLineItemNumber.AllColumns).FROM(table.ArmyLineItemNumber).
			WHERE(table.ArmyLineItemNumber.Lin.EQ(String(*amdf.AmdfManagement.Lin)))

		err = armyLinStmt.Query(repo.Db, &amdf.ArmyLin)
	}

	if err != nil {
		return details.Amdf{}, err
	} else {
		return amdf, nil
	}

}

func (repo *ItemDetailedRepositoryImpl) getArmyPackagingAndFreight(niin string) (details.ArmyPackagingAndFreight, error) {
	armyPackagingAndFreight := details.ArmyPackagingAndFreight{}

	armyPackagingAndFreightStmt := SELECT(
		table.ArmyPackagingAndFreight.AllColumns).FROM(table.ArmyPackagingAndFreight).
		WHERE(table.ArmyPackagingAndFreight.Niin.EQ(String(niin)))

	err := armyPackagingAndFreightStmt.Query(repo.Db, &armyPackagingAndFreight.ArmyPackagingAndFreight)

	armyPackaging1Stmt := SELECT(
		table.ArmyPackaging1.AllColumns).FROM(table.ArmyPackaging1).
		WHERE(table.ArmyPackaging1.Niin.EQ(String(niin)))

	err = armyPackaging1Stmt.Query(repo.Db, &armyPackagingAndFreight.ArmyPackaging1)

	armyPackaging2Stmt := SELECT(
		table.ArmyPackaging2.AllColumns).FROM(table.ArmyPackaging2).
		WHERE(table.ArmyPackaging2.Niin.EQ(String(niin)))

	err = armyPackaging2Stmt.Query(repo.Db, &armyPackagingAndFreight.ArmyPackaging2)

	armyPackSpecialInstructStmt := SELECT(
		table.ArmyPackagingSpecialInstruct.AllColumns).FROM(table.ArmyPackagingSpecialInstruct).
		WHERE(table.ArmyPackagingSpecialInstruct.Niin.EQ(String(niin)))

	err = armyPackSpecialInstructStmt.Query(repo.Db, &armyPackagingAndFreight.ArmyPackSpecialInstruct)

	armyFreightStmt := SELECT(
		table.ArmyFreight.AllColumns).FROM(table.ArmyFreight).
		WHERE(table.ArmyFreight.Niin.EQ(String(niin)))

	err = armyFreightStmt.Query(repo.Db, &armyPackagingAndFreight.ArmyFreight)

	armyPackSupplementalInstructStmt := SELECT(
		table.ArmyPackSupplementalInstruct.AllColumns).FROM(table.ArmyPackSupplementalInstruct).
		WHERE(table.ArmyPackSupplementalInstruct.Niin.EQ(String(niin)))

	err = armyPackSupplementalInstructStmt.Query(repo.Db, &armyPackagingAndFreight.ArmyPackSupplementalInstruct)

	if err != nil {
		return details.ArmyPackagingAndFreight{}, err
	} else {
		return armyPackagingAndFreight, nil
	}
}

func (repo *ItemDetailedRepositoryImpl) getSarsscat(niin string) (details.Sarsscat, error) {
	sarsscat := details.Sarsscat{}

	sarsscatStmt := SELECT(
		table.ArmySarsscat.AllColumns).FROM(table.ArmySarsscat).
		WHERE(table.ArmySarsscat.Niin.EQ(String(niin)))

	err := sarsscatStmt.Query(repo.Db, &sarsscat.ArmySarsscat)

	moeRuleStmt := SELECT(
		table.MoeRule.AllColumns).FROM(table.MoeRule).
		WHERE(table.MoeRule.Niin.EQ(String(niin)))

	err = moeRuleStmt.Query(repo.Db, &sarsscat.MoeRule)

	amdfFreightStmt := SELECT(
		table.AmdfFreight.AllColumns).FROM(table.AmdfFreight).
		WHERE(table.AmdfFreight.Niin.EQ(String(niin)))

	err = amdfFreightStmt.Query(repo.Db, &sarsscat.AmdfFreight)

	if err != nil {
		return details.Sarsscat{}, err
	} else {
		return sarsscat, nil
	}
}

func (repo *ItemDetailedRepositoryImpl) getIdentification(niin string) (details.Identification, error) {
	identification := details.Identification{}

	flisMgmtStmt := SELECT(
		table.FlisManagementID.AllColumns).FROM(table.FlisManagementID).
		WHERE(table.FlisManagementID.Niin.EQ(String(niin)))

	err := flisMgmtStmt.Query(repo.Db, &identification.FlisManagementId)

	// Only run if the FlisManagementId.Inc is not nil
	if identification.FlisManagementId.Inc == nil {
		colloquialNamesStmt := SELECT(
			table.ColloquialName.AllColumns).FROM(table.ColloquialName).
			WHERE(table.ColloquialName.Inc.EQ(String(*identification.FlisManagementId.Inc)))

		err = colloquialNamesStmt.Query(repo.Db, &identification.ColloquialName)
	}

	flisStandardizationStmt := SELECT(
		table.FlisStandardization.AllColumns).FROM(table.FlisStandardization).
		WHERE(table.FlisStandardization.Niin.EQ(String(niin)))

	err = flisStandardizationStmt.Query(repo.Db, &identification.FlisStandardization)

	flisCancelledNiinStmt := SELECT(
		table.FlisCancelledNiin.AllColumns).FROM(table.FlisCancelledNiin).
		WHERE(table.FlisCancelledNiin.Niin.EQ(String(niin)))

	err = flisCancelledNiinStmt.Query(repo.Db, &identification.FlisCancelledNiin)

	if err != nil {
		return details.Identification{}, err
	} else {
		return identification, nil
	}
}

func (repo *ItemDetailedRepositoryImpl) getManagement(niin string) (details.Management, error) {
	management := details.Management{}

	flisManagementStmt := SELECT(
		table.FlisManagement.AllColumns).FROM(table.FlisManagement).
		WHERE(table.FlisManagement.Niin.EQ(String(niin)))

	err := flisManagementStmt.Query(repo.Db, &management.FLisManagement)

	flisPhraseStmt := SELECT(
		table.FlisPhrase.AllColumns).FROM(table.FlisPhrase).
		WHERE(table.FlisPhrase.Niin.EQ(String(niin)))

	err = flisPhraseStmt.Query(repo.Db, &management.FlisPhrase)

	componentEndItemStmt := SELECT(
		table.ComponentEndItem.AllColumns).FROM(table.ComponentEndItem).
		WHERE(table.ComponentEndItem.Niin.EQ(String(niin)))

	err = componentEndItemStmt.Query(repo.Db, &management.ComponentEndItem)

	armyManagementStmt := SELECT(
		table.ArmyManagement.AllColumns).FROM(table.ArmyManagement).
		WHERE(table.ArmyManagement.Niin.EQ(String(niin)))

	err = armyManagementStmt.Query(repo.Db, &management.ArmyManagement)

	airForceManagementStmt := SELECT(
		table.AirForceManagement.AllColumns).FROM(table.AirForceManagement).
		WHERE(table.AirForceManagement.Niin.EQ(String(niin)))

	err = airForceManagementStmt.Query(repo.Db, &management.AirForceManagement)

	marineCorpsManagementStmt := SELECT(
		table.MarineCorpsManagement.AllColumns).FROM(table.MarineCorpsManagement).
		WHERE(table.MarineCorpsManagement.Niin.EQ(String(niin)))

	err = marineCorpsManagementStmt.Query(repo.Db, &management.MarineCorpsManagement)

	navyManagementStmt := SELECT(
		table.NavyManagement.AllColumns).FROM(table.NavyManagement).
		WHERE(table.NavyManagement.Niin.EQ(String(niin)))

	err = navyManagementStmt.Query(repo.Db, &management.NavyManagement)

	faaManagementStmt := SELECT(
		table.FaaManagement.AllColumns).FROM(table.FaaManagement).
		WHERE(table.FaaManagement.Niin.EQ(String(niin)))

	err = faaManagementStmt.Query(repo.Db, &management.FaaManagement)

	if err != nil {
		return details.Management{}, err
	} else {
		return management, nil
	}
}

func (repo *ItemDetailedRepositoryImpl) getReference(niin string) (details.Reference, error) {
	reference := details.Reference{}

	// FlisIdentification
	flisReferenceStmt := SELECT(
		table.FlisIdentification.AllColumns).FROM(table.FlisIdentification).
		WHERE(table.FlisIdentification.Niin.EQ(String(niin)))

	err := flisReferenceStmt.Query(repo.Db, &reference.FlisReference)

	// This will be multiple returns
	referenceAndPartNumberStmt := SELECT(
		table.FlisReference.AllColumns).FROM(table.FlisReference).
		WHERE(table.FlisReference.Niin.EQ(String(niin)))

	err = referenceAndPartNumberStmt.Query(repo.Db, &reference.ReferenceAndPartNumber)

	// Loop through all the referenceAndPartNumber results and get the CageCode
	// Ensure none are nil or empty
	var cageCodes string
	for _, ref := range reference.ReferenceAndPartNumber {
		if ref.CageCode != nil {
			if cageCodes != "" {
				cageCodes += ","
			}
			cageCodes += *ref.CageCode
		}
	}
	// The two following statements will have to use the multiple returns from the previous statement
	cageAddressesStmt := SELECT(
		table.CageAddress.AllColumns).FROM(table.CageAddress).
		WHERE(table.CageAddress.CageCode.IN(String(cageCodes)))

	err = cageAddressesStmt.Query(repo.Db, &reference.CageAddresses)

	cageStatusAndTypeStmt := SELECT(
		table.CageStatusAndType.AllColumns).FROM(table.CageStatusAndType).
		WHERE(table.CageStatusAndType.CageCode.IN(String(cageCodes)))

	err = cageStatusAndTypeStmt.Query(repo.Db, &reference.CageStatusAndType)

	if err != nil {
		return details.Reference{}, err
	} else {
		return reference, nil
	}
}

func (repo *ItemDetailedRepositoryImpl) getFreight(niin string) (details.Freight, error) {
	freight := details.Freight{}

	freightStmt := SELECT(
		table.FlisFreight.AllColumns).FROM(table.FlisFreight).
		WHERE(table.FlisFreight.Niin.EQ(String(niin)))

	err := freightStmt.Query(repo.Db, &freight.FlisFreight)

	if err != nil {
		return details.Freight{}, err
	} else {
		return freight, nil
	}
}

func (repo *ItemDetailedRepositoryImpl) getPackaging(niin string) (details.Packaging, error) {
	packaging := details.Packaging{}

	pack1Stmt := SELECT(
		table.FlisPackaging1.AllColumns).FROM(table.FlisPackaging1).
		WHERE(table.FlisPackaging1.Niin.EQ(String(niin)))

	err := pack1Stmt.Query(repo.Db, &packaging.FlisPackaging1)

	// Loop through all the referenceAndPartNumber results and get the CageCode

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
		table.FlisPackaging2.AllColumns).FROM(table.FlisPackaging2).
		WHERE(table.FlisPackaging2.Niin.EQ(String(niin)))

	err = pack2Stmt.Query(repo.Db, &packaging.FlisPackaging2)

	cageAddressStmt := SELECT(
		table.CageAddress.AllColumns).FROM(table.CageAddress).
		WHERE(table.CageAddress.CageCode.IN(String(cageCodes)))

	err = cageAddressStmt.Query(repo.Db, &packaging.CageAddress)

	dssWeightStmt := SELECT(
		table.DssWeightAndCube.AllColumns).FROM(table.DssWeightAndCube).
		WHERE(table.DssWeightAndCube.Niin.EQ(String(niin)))

	err = dssWeightStmt.Query(repo.Db, &packaging.DssWeightAndCube)

	if err != nil {
		return details.Packaging{}, err
	} else {
		return packaging, nil
	}

}

func (repo *ItemDetailedRepositoryImpl) getCharacteristics(niin string) (details.Characteristics, error) {
	characteristics := details.Characteristics{}

	characteristicsStmt := SELECT(
		table.FlisItemCharacteristics.AllColumns).FROM(table.FlisItemCharacteristics).
		WHERE(table.FlisItemCharacteristics.Niin.EQ(String(niin)))

	err := characteristicsStmt.Query(repo.Db, &characteristics.Characteristics)

	if err != nil {
		return details.Characteristics{}, err
	} else {
		return characteristics, nil
	}

}

func (repo *ItemDetailedRepositoryImpl) getDisposition(niin string) (details.Disposition, error) {
	disposition := details.Disposition{}

	dispositionStmt := SELECT(
		table.Disposition.AllColumns).FROM(table.Disposition).
		WHERE(table.Disposition.Niin.EQ(String(niin)))

	err := dispositionStmt.Query(repo.Db, &disposition.Disposition)

	if err != nil {
		return details.Disposition{}, err
	} else {
		return disposition, nil
	}
}
