package repository

import (
	"database/sql"
	. "github.com/go-jet/jet/v2/postgres"
	"log/slog"
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/.gen/miltech_ng/public/table"
)

type ItemDetailedRepositoryImpl struct {
	Db *sql.DB
}

func NewItemDetailedRepositoryImpl(db *sql.DB) *ItemDetailedRepositoryImpl {
	return &ItemDetailedRepositoryImpl{Db: db}
}

func (repo *ItemDetailedRepositoryImpl) GetDetailedItemData(niin string) (interface{}, error) {
	// Get data from each table
	// Call helper methods to get data from each table
	// Return a DetailedItem struct with all the data

	var dest []struct {
		Amdf struct {
			ArmyMasterDataFile model.ArmyMasterDataFile
			AmdfManagement     model.AmdfManagement
			AmdfCredit         model.AmdfCredit
			AmdfBilling        model.AmdfBilling
			AmdfMatcat         model.AmdfMatcat
			AmdfPhrases        []model.AmdfPhrase
			AmdfIAndS          []model.AmdfIAndS
			ArmyLineItemNumber model.ArmyLineItemNumber
		}
		Packaging struct {
			ArmyPackagingAndFreight      model.ArmyPackagingAndFreight
			ArmyPackaging1               model.ArmyPackaging1
			ArmyPackaging2               model.ArmyPackaging2
			ArmyPackSpecialInstruct      model.ArmyPackagingSpecialInstruct
			ArmyFreight                  model.ArmyFreight
			ArmyPackSupplementalInstruct []model.ArmyPackSupplementalInstruct
		}

		Sarsscat struct {
			Sarsscat    model.ArmySarsscat
			MoeRule     []model.MoeRule
			AmdfFreight model.AmdfFreight
		}

		Identification struct {
			FlisManagementId    model.FlisManagementID
			ColloquialNames     []model.ColloquialName
			FlisStandardization []model.FlisStandardization
			FlisCancelledNiins  []model.FlisCancelledNiin
		}

		Management struct {
			FlisManagement        []model.FlisManagement
			FlisPhrase            []model.FlisPhrase
			ComponentEndItem      []model.ComponentEndItem
			ArmyManagement        []model.ArmyManagement
			AirForceManagement    model.AirForceManagement
			MarineCorpsManagement []model.MarineCorpsManagement
			NavyManagement        model.NavyManagement
			FaaManagement         []model.FaaManagement
		}

		Reference struct {
			FlisReference          model.FlisIdentification
			ReferenceAndPartNumber []model.FlisReference
			CageAddresses          []model.CageAddress
			CageStatusAndType      []model.CageStatusAndType
		}
	}

	stmt := SELECT(
		table.ArmyMasterDataFile.AllColumns,
		table.AmdfManagement.AllColumns,
		table.AmdfCredit.AllColumns,
		table.AmdfBilling.AllColumns,
		table.AmdfMatcat.AllColumns,
		table.AmdfPhrase.AllColumns,
		table.AmdfIAndS.AllColumns,
		table.ArmyLineItemNumber.AllColumns,
		table.ArmyPackagingAndFreight.AllColumns,
		table.ArmyPackaging1.AllColumns,
		table.ArmyPackaging2.AllColumns,
		table.ArmyPackagingSpecialInstruct.AllColumns,
		table.ArmyFreight.AllColumns,
		table.ArmyPackSupplementalInstruct.AllColumns,
		table.ArmySarsscat.AllColumns,
		table.MoeRule.AllColumns,
		table.AmdfFreight.AllColumns,
		table.FlisManagementID.AllColumns,
		table.ColloquialName.AllColumns,
		table.FlisStandardization.AllColumns,
		table.FlisCancelledNiin.AllColumns,
		table.FlisManagement.AllColumns,
		table.FlisPhrase.AllColumns,
		table.ComponentEndItem.AllColumns,
		table.ArmyManagement.AllColumns,
		table.AirForceManagement.AllColumns,
		table.MarineCorpsManagement.AllColumns,
		table.NavyManagement.AllColumns,
		table.FaaManagement.AllColumns,
		table.FlisIdentification.AllColumns,
		table.FlisReference.AllColumns,
		table.CageAddress.AllColumns,
		table.CageStatusAndType.AllColumns,
	).FROM(
		table.ArmyMasterDataFile.
			LEFT_JOIN(
				table.AmdfCredit, table.AmdfCredit.Niin.EQ(table.ArmyMasterDataFile.Niin)).
			LEFT_JOIN(table.AmdfManagement, table.AmdfManagement.Niin.EQ(table.ArmyMasterDataFile.Niin)).
			LEFT_JOIN(
				table.AmdfBilling, table.AmdfBilling.Niin.EQ(table.ArmyMasterDataFile.Niin)).
			LEFT_JOIN(
				table.AmdfMatcat, table.AmdfMatcat.Niin.EQ(table.ArmyMasterDataFile.Niin)).
			LEFT_JOIN(
				table.AmdfPhrase, table.AmdfPhrase.Niin.EQ(table.ArmyMasterDataFile.Niin)).
			LEFT_JOIN(table.AmdfIAndS, table.AmdfIAndS.Niin.EQ(table.ArmyMasterDataFile.Niin)).
			LEFT_JOIN(table.ArmyLineItemNumber, table.ArmyLineItemNumber.Lin.EQ(table.ArmyLineItemNumber.Lin)).
			// Army Packaging and Freight JOINS
			LEFT_JOIN(
				table.ArmyPackagingAndFreight, table.ArmyPackagingAndFreight.Niin.EQ(table.ArmyMasterDataFile.Niin)).
			LEFT_JOIN(
				table.ArmyPackaging1, table.ArmyPackaging1.Niin.EQ(table.ArmyMasterDataFile.Niin)).
			LEFT_JOIN(
				table.ArmyPackaging2, table.ArmyPackaging2.Niin.EQ(table.ArmyMasterDataFile.Niin)).
			LEFT_JOIN(
				table.ArmyPackagingSpecialInstruct, table.ArmyPackagingSpecialInstruct.Niin.EQ(table.ArmyMasterDataFile.Niin)).
			LEFT_JOIN(
				table.ArmyFreight, table.ArmyFreight.Niin.EQ(table.ArmyMasterDataFile.Niin)).
			LEFT_JOIN(
				table.ArmyPackSupplementalInstruct, table.ArmyPackSupplementalInstruct.Niin.EQ(table.ArmyMasterDataFile.Niin)).
			// Sarsscat JOINS
			LEFT_JOIN(table.ArmySarsscat, table.ArmySarsscat.Niin.EQ(table.ArmyMasterDataFile.Niin)).
			LEFT_JOIN(table.MoeRule, table.MoeRule.Niin.EQ(table.ArmyMasterDataFile.Niin)).
			LEFT_JOIN(table.AmdfFreight, table.AmdfFreight.Niin.EQ(table.ArmyMasterDataFile.Niin)).
			// Identification JOINS
			LEFT_JOIN(table.FlisManagementID, table.FlisManagementID.Niin.EQ(table.ArmyMasterDataFile.Niin)).
			LEFT_JOIN(table.ColloquialName, table.ColloquialName.Inc.EQ(table.ColloquialName.Inc)).
			LEFT_JOIN(table.FlisStandardization, table.FlisStandardization.Niin.EQ(table.ArmyMasterDataFile.Niin)).
			LEFT_JOIN(table.FlisCancelledNiin, table.FlisCancelledNiin.Niin.EQ(table.ArmyMasterDataFile.Niin)).
			// Management JOINS
			LEFT_JOIN(table.FlisManagement, table.FlisManagement.Niin.EQ(table.ArmyMasterDataFile.Niin)).
			LEFT_JOIN(table.FlisPhrase, table.FlisPhrase.Niin.EQ(table.ArmyMasterDataFile.Niin)).
			LEFT_JOIN(table.ComponentEndItem, table.ComponentEndItem.Niin.EQ(table.ArmyMasterDataFile.Niin)).
			LEFT_JOIN(table.ArmyManagement, table.ArmyManagement.Niin.EQ(table.ArmyMasterDataFile.Niin)).
			LEFT_JOIN(table.AirForceManagement, table.AirForceManagement.Niin.EQ(table.ArmyMasterDataFile.Niin)).
			LEFT_JOIN(table.MarineCorpsManagement, table.MarineCorpsManagement.Niin.EQ(table.ArmyMasterDataFile.Niin)).
			LEFT_JOIN(table.NavyManagement, table.NavyManagement.Niin.EQ(table.ArmyMasterDataFile.Niin)).
			LEFT_JOIN(table.FaaManagement, table.FaaManagement.Niin.EQ(table.ArmyMasterDataFile.Niin)).
			//Reference JOINS
			LEFT_JOIN(table.FlisIdentification, table.FlisIdentification.Niin.EQ(table.ArmyMasterDataFile.Niin)).
			LEFT_JOIN(table.FlisReference, table.FlisReference.Niin.EQ(table.ArmyMasterDataFile.Niin)).
			LEFT_JOIN(table.CageAddress, table.CageAddress.CageCode.EQ(table.FlisReference.CageCode)).
			LEFT_JOIN(table.CageStatusAndType, table.CageStatusAndType.CageCode.EQ(table.FlisReference.CageCode)),
	).WHERE(
		table.ArmyMasterDataFile.Niin.EQ(String(niin)),
	)

	debugSql := stmt.DebugSql()
	slog.Info(debugSql)

	err := stmt.Query(repo.Db, &dest)
	if err != nil {
		return nil, err
	} else {
		return dest, nil
	}
}

//table.ArmyMasterDataFile.LEFT_JOIN(
//// Amdf JOINS
//table.AmdfManagement, table.ArmyMasterDataFile.Niin.EQ(table.AmdfManagement.Niin)).

// GetAmdfData retrieves the AMDF field data for a given NIIN
// || ArmyMasterDataFile, AmdfManagement, AmdfCredit, AmdfBilling, AmdfMatcat, AmdfPhrases, AmdfIAndS, ArmyLin
//func (repo *ItemDetailedRepositoryImpl) GetAmdfData(ctx *gin.Context, niin string) (details.Amdf, error) {
//	var linData *db.ArmyLineItemNumberModel
//
//	amdfData, _ := repo.Db.ArmyMasterDataFile.FindFirst(db.ArmyMasterDataFile.Niin.Equals(niin)).Exec(ctx)
//	managementData, _ := repo.Db.AmdfManagement.FindFirst(db.AmdfManagement.Niin.Equals(niin)).Exec(ctx)
//	creditData, _ := repo.Db.AmdfCredit.FindFirst(db.AmdfCredit.Niin.Equals(niin)).Exec(ctx)
//	billingData, _ := repo.Db.AmdfBilling.FindFirst(db.AmdfBilling.Niin.Equals(niin)).Exec(ctx)
//	matcatData, _ := repo.Db.AmdfMatcat.FindFirst(db.AmdfMatcat.Niin.Equals(niin)).Exec(ctx)
//	phrasesData, _ := repo.Db.AmdfPhrase.FindMany(db.AmdfPhrase.Niin.Equals(niin)).Exec(ctx)
//	iAndSData, _ := repo.Db.AmdfIAndS.FindMany(db.AmdfIAndS.Niin.Equals(niin)).Exec(ctx)
//
//	testLin, _ := managementData.Lin()
//
//	if testLin != "" {
//		linData, _ = repo.Db.ArmyLineItemNumber.FindFirst(db.ArmyLineItemNumber.Lin.Equals(testLin)).Exec(ctx)
//	}
//
//	data := details.Amdf{
//		// Map fields from data to details.Amdf
//		ArmyMasterDataFile: *amdfData,
//		AmdfManagement:     *managementData,
//		AmdfCredit:         *creditData,
//		AmdfBilling:        *billingData,
//		AmdfMatcat:         *matcatData,
//		AmdfPhrases:        phrasesData,
//		AmdfIandS:          iAndSData,
//		ArmyLin:            linData,
//	}
//	return data, nil
//}
//
//func (repo *ItemDetailedRepositoryImpl) GetArmyPackagingAndFreight(ctx *gin.Context, niin string) (details.ArmyPackagingAndFreight, error) {
//	armyPackingAndFreightData, _ := repo.Db.ArmyPackagingAndFreight.FindFirst(db.ArmyPackagingAndFreight.Niin.Equals(niin)).Exec(ctx)
//	armyPackaging1Data, _ := repo.Db.ArmyPackaging1.FindUnique(db.ArmyPackaging1.Niin.Equals(niin)).Exec(ctx)
//	armyPackaging2Data, _ := repo.Db.ArmyPackaging2.FindUnique(db.ArmyPackaging2.Niin.Equals(niin)).Exec(ctx)
//	armyPackSpecialInstructData, _ := repo.Db.ArmyPackagingSpecialInstruct.FindUnique(db.ArmyPackagingSpecialInstruct.Niin.Equals(niin)).Exec(ctx)
//	armyFreightData, _ := repo.Db.ArmyFreight.FindUnique(db.ArmyFreight.Niin.Equals(niin)).Exec(ctx)
//	armyPackSupplementalInstructData, _ := repo.Db.ArmyPackSupplementalInstruct.FindMany(db.ArmyPackSupplementalInstruct.Niin.Equals(niin)).Exec(ctx)
//
//	data := details.ArmyPackagingAndFreight{
//		ArmyPackagingAndFreight:      *armyPackingAndFreightData,
//		ArmyPackaging1:               *armyPackaging1Data,
//		ArmyPackaging2:               *armyPackaging2Data,
//		ArmyPackSpecialInstruct:      *armyPackSpecialInstructData,
//		ArmyFreight:                  *armyFreightData,
//		ArmyPackSupplementalInstruct: armyPackSupplementalInstructData,
//	}
//	return data, nil
//}
//
//func (repo *ItemDetailedRepositoryImpl) GetSarsscat(ctx *gin.Context, niin string) (details.Sarsscat, error) {
//	sarsscatData, _ := repo.Db.ArmySarsscat.FindUnique(db.ArmySarsscat.Niin.Equals(niin)).Exec(ctx)
//	moeRuleData, _ := repo.Db.MoeRule.FindMany(db.MoeRule.Niin.Equals(niin)).Exec(ctx)
//	amdfFreightData, _ := repo.Db.AmdfFreight.FindUnique(db.AmdfFreight.Niin.Equals(niin)).Exec(ctx)
//
//	data := details.Sarsscat{
//		ArmySarsscat: *sarsscatData,
//		MoeRule:      moeRuleData,
//		AmdfFreight:  *amdfFreightData,
//	}
//
//	return data, nil
//
//}
//
//func (repo *ItemDetailedRepositoryImpl) GetIdentification(ctx *gin.Context, niin string) (details.Identification, error) {
//	var colloqNames []db.ColloquialNameModel
//	flisManagementIdData, _ := repo.Db.FlisManagementID.FindFirst(db.FlisManagementID.Niin.Equals(niin)).Exec(ctx)
//	flisStandardizationData, _ := repo.Db.FlisStandardization.FindMany(db.FlisStandardization.Niin.Equals(niin)).Exec(ctx)
//	FlisCancelledNiinData, _ := repo.Db.FlisCancelledNiin.FindMany(db.FlisCancelledNiin.Niin.Equals(niin)).Exec(ctx)
//
//	if flisManagementIdData != nil {
//		incPlacerholder, _ := flisManagementIdData.Inc()
//		if len(incPlacerholder) == 5 {
//			colloqNameData, _ := repo.Db.ColloquialName.FindMany(db.ColloquialName.Inc.Equals(incPlacerholder)).Exec(ctx)
//
//			for _, name := range colloqNameData {
//				colloqNames = append(colloqNames, name)
//			}
//		}
//	}
//
//	if flisManagementIdData == nil {
//		flisManagementIdData = &db.FlisManagementIDModel{}
//	}
//
//	data := details.Identification{
//		FlisManagementId:    *flisManagementIdData,
//		ColloquialNames:     colloqNames,
//		FlisStandardization: flisStandardizationData,
//		FlisCancelledNiin:   FlisCancelledNiinData,
//	}
//
//	return data, nil
//
//}
//
//func (repo *ItemDetailedRepositoryImpl) GetManagement(ctx *gin.Context, niin string) (details.Management, error) {
//	flisManData, _ := repo.Db.FlisManagement.FindMany(db.FlisManagement.Niin.Equals(niin)).Exec(ctx)
//	flisPhraseData, _ := repo.Db.FlisPhrase.FindMany(db.FlisPhrase.Niin.Equals(niin)).Exec(ctx)
//	componentEndItemData, _ := repo.Db.ComponentEndItem.FindMany(db.ComponentEndItem.Niin.Equals(niin)).Exec(ctx)
//	armyManagementData, _ := repo.Db.ArmyManagement.FindMany(db.ArmyManagement.Niin.Equals(niin)).Exec(ctx)
//	airForceManagementData, _ := repo.Db.AirForceManagement.FindFirst(db.AirForceManagement.Niin.Equals(niin)).Exec(ctx)
//	marineManagementData, _ := repo.Db.MarineCorpsManagement.FindMany(db.MarineCorpsManagement.Niin.Equals(niin)).Exec(ctx)
//	navyManagementData, _ := repo.Db.NavyManagement.FindFirst(db.NavyManagement.Niin.Equals(niin)).Exec(ctx)
//	faaManagementData, _ := repo.Db.FaaManagement.FindMany(db.FaaManagement.Niin.Equals(niin)).Exec(ctx)
//
//	data := details.Management{
//		FLisManagement:        flisManData,
//		FlisPhrase:            flisPhraseData,
//		ComponentEndItem:      componentEndItemData,
//		ArmyManagement:        armyManagementData,
//		AirForceManagement:    airForceManagementData,
//		MarineCorpsManagement: marineManagementData,
//		NavyManagement:        navyManagementData,
//		FaaManagement:         faaManagementData,
//	}
//
//	return data, nil
//
//}
//
//func (repo *ItemDetailedRepositoryImpl) GetReference(ctx *gin.Context, niin string) (details.Reference, error) {
//	flisRefData, _ := repo.Db.FlisIdentification.FindUnique(db.FlisIdentification.Niin.Equals(niin)).Exec(ctx)
//	refAndPartData, _ := repo.Db.FlisReference.FindMany(db.FlisReference.Niin.Equals(niin)).Exec(ctx)
//
//	var cageAdressPlaceholder []db.CageAddressModel
//	var cageStatusAndTypePlaceholder []db.CageStatusAndTypeModel
//
//	if refAndPartData != nil {
//		for _, part := range refAndPartData {
//			cage, _ := part.CageCode()
//			cageData, _ := repo.Db.CageAddress.FindFirst(db.CageAddress.CageCode.Equals(cage)).Exec(ctx)
//			cageAdressPlaceholder = append(cageAdressPlaceholder, *cageData)
//
//			cageStatusData, _ := repo.Db.CageStatusAndType.FindFirst(db.CageStatusAndType.CageCode.Equals(cage)).Exec(ctx)
//			cageStatusAndTypePlaceholder = append(cageStatusAndTypePlaceholder, *cageStatusData)
//		}
//	}
//
//	data := details.Reference{
//		FlisReference:          flisRefData,
//		ReferenceAndPartNumber: refAndPartData,
//		CageAddresses:          cageAdressPlaceholder,
//		CageStatusAndType:      cageStatusAndTypePlaceholder,
//	}
//
//	return data, nil
//
//}
//
//func (repo *ItemDetailedRepositoryImpl) GetFreight(ctx *gin.Context, niin string) (details.Freight, error) {
//	flisFreightData, _ := repo.Db.FlisFreight.FindFirst(db.FlisFreight.Niin.Equals(niin)).Exec(ctx)
//
//	data := details.Freight{
//		FlisFreight: *flisFreightData,
//	}
//
//	return data, nil
//}
//
//func (repo *ItemDetailedRepositoryImpl) GetPackaging(ctx *gin.Context, niin string) (details.Packaging, error) {
//	var cageAddressPlaceholder []db.CageAddressModel
//
//	flisPackaging1Data, _ := repo.Db.FlisPackaging1.FindMany(db.FlisPackaging1.Niin.Equals(niin)).Exec(ctx)
//	flisPackaging2Data, _ := repo.Db.FlisPackaging2.FindMany(db.FlisPackaging2.Niin.Equals(niin)).Exec(ctx)
//	dssWeightAndCubeData, _ := repo.Db.DssWeightAndCube.FindFirst(db.DssWeightAndCube.Niin.Equals(niin)).Exec(ctx)
//
//	if flisPackaging1Data != nil {
//		for _, part := range flisPackaging1Data {
//			cage, _ := part.PkgDesignActy()
//			cageData, _ := repo.Db.CageAddress.FindFirst(db.CageAddress.CageCode.Equals(cage)).Exec(ctx)
//			cageAddressPlaceholder = append(cageAddressPlaceholder, *cageData)
//		}
//	}
//
//	data := details.Packaging{
//		FlisPackaging1:   flisPackaging1Data,
//		FlisPackaging2:   flisPackaging2Data,
//		CageAddress:      cageAddressPlaceholder,
//		DssWeightAndCube: *dssWeightAndCubeData,
//	}
//
//	return data, nil
//}
//
//func (repo *ItemDetailedRepositoryImpl) GetCharacteristics(ctx *gin.Context, niin string) (details.Characteristics, error) {
//	characteristicsData, _ := repo.Db.FlisItemCharacteristics.FindMany(db.FlisItemCharacteristics.Niin.Equals(niin)).Exec(ctx)
//
//	data := details.Characteristics{
//		Characteristics: characteristicsData,
//	}
//	return data, nil
//
//}
//
//func (repo *ItemDetailedRepositoryImpl) GetDisposition(ctx *gin.Context, niin string) (details.Disposition, error) {
//	dispositionData, _ := repo.Db.Disposition.FindFirst(db.Disposition.Niin.Equals(niin)).Exec(ctx)
//
//	data := details.Disposition{
//		Disposition: dispositionData,
//	}
//
//	return data, nil
//}
