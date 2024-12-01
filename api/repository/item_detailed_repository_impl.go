package repository

import (
	"context"
	"miltechserver/model/details"
	"miltechserver/prisma/db"
)

type ItemDetailedRepositoryImpl struct {
	Db *db.PrismaClient
}

func NewItemDetailedRepositoryImpl(db *db.PrismaClient) *ItemDetailedRepositoryImpl {
	return &ItemDetailedRepositoryImpl{Db: db}
}

// GetAmdfData retrieves the AMDF field data for a given NIIN
// || ArmyMasterDataFile, AmdfManagement, AmdfCredit, AmdfBilling, AmdfMatcat, AmdfPhrases, AmdfIAndS, ArmyLin
func (repo *ItemDetailedRepositoryImpl) GetAmdfData(ctx context.Context, niin string) (details.Amdf, error) {
	var linData *db.ArmyLineItemNumberModel

	amdfData, _ := repo.Db.ArmyMasterDataFile.FindFirst(db.ArmyMasterDataFile.Niin.Equals(niin)).Exec(ctx)
	managementData, _ := repo.Db.AmdfManagement.FindFirst(db.AmdfManagement.Niin.Equals(niin)).Exec(ctx)
	creditData, _ := repo.Db.AmdfCredit.FindFirst(db.AmdfCredit.Niin.Equals(niin)).Exec(ctx)
	billingData, _ := repo.Db.AmdfBilling.FindFirst(db.AmdfBilling.Niin.Equals(niin)).Exec(ctx)
	matcatData, _ := repo.Db.AmdfMatcat.FindFirst(db.AmdfMatcat.Niin.Equals(niin)).Exec(ctx)
	phrasesData, _ := repo.Db.AmdfPhrase.FindMany(db.AmdfPhrase.Niin.Equals(niin)).Exec(ctx)
	iAndSData, _ := repo.Db.AmdfIAndS.FindMany(db.AmdfIAndS.Niin.Equals(niin)).Exec(ctx)

	testLin, _ := managementData.Lin()

	if testLin != "" {
		linData, _ = repo.Db.ArmyLineItemNumber.FindFirst(db.ArmyLineItemNumber.Lin.Equals(testLin)).Exec(ctx)
	}

	data := details.Amdf{
		// Map fields from data to details.Amdf
		ArmyMasterDataFile: *amdfData,
		AmdfManagement:     *managementData,
		AmdfCredit:         *creditData,
		AmdfBilling:        *billingData,
		AmdfMatcat:         *matcatData,
		AmdfPhrases:        phrasesData,
		AmdfIandS:          iAndSData,
		ArmyLin:            linData,
	}
	return data, nil
}

func (repo *ItemDetailedRepositoryImpl) GetArmyPackagingAndFreight(ctx context.Context, niin string) (details.ArmyPackagingAndFreight, error) {
	armyPackingAndFreightData, _ := repo.Db.ArmyPackagingAndFreight.FindFirst(db.ArmyPackagingAndFreight.Niin.Equals(niin)).Exec(ctx)
	armyPackaging1Data, _ := repo.Db.ArmyPackaging1.FindUnique(db.ArmyPackaging1.Niin.Equals(niin)).Exec(ctx)
	armyPackaging2Data, _ := repo.Db.ArmyPackaging2.FindUnique(db.ArmyPackaging2.Niin.Equals(niin)).Exec(ctx)
	armyPackSpecialInstructData, _ := repo.Db.ArmyPackagingSpecialInstruct.FindUnique(db.ArmyPackagingSpecialInstruct.Niin.Equals(niin)).Exec(ctx)
	armyFreightData, _ := repo.Db.ArmyFreight.FindUnique(db.ArmyFreight.Niin.Equals(niin)).Exec(ctx)
	armyPackSupplementalInstructData, _ := repo.Db.ArmyPackSupplementalInstruct.FindMany(db.ArmyPackSupplementalInstruct.Niin.Equals(niin)).Exec(ctx)

	data := details.ArmyPackagingAndFreight{
		ArmyPackagingAndFreight:      *armyPackingAndFreightData,
		ArmyPackaging1:               *armyPackaging1Data,
		ArmyPackaging2:               *armyPackaging2Data,
		ArmyPackSpecialInstruct:      *armyPackSpecialInstructData,
		ArmyFreight:                  *armyFreightData,
		ArmyPackSupplementalInstruct: armyPackSupplementalInstructData,
	}
	return data, nil
}

func (repo *ItemDetailedRepositoryImpl) GetSarsscat(ctx context.Context, niin string) (details.Sarsscat, error) {
	sarsscatData, _ := repo.Db.ArmySarsscat.FindUnique(db.ArmySarsscat.Niin.Equals(niin)).Exec(ctx)
	moeRuleData, _ := repo.Db.MoeRule.FindMany(db.MoeRule.Niin.Equals(niin)).Exec(ctx)
	amdfFreightData, _ := repo.Db.AmdfFreight.FindUnique(db.AmdfFreight.Niin.Equals(niin)).Exec(ctx)

	data := details.Sarsscat{
		ArmySarsscat: *sarsscatData,
		MoeRule:      moeRuleData,
		AmdfFreight:  *amdfFreightData,
	}

	return data, nil

}

//func (repo *ItemDetailedRepositoryImpl) GetIdentification(ctx context.Context, niin string) (details.Identification, error) {
//	data, err := repo.Db.Identification.FindFirst(db.Identification.Niin.Equals(niin)).Exec(ctx)
//	if err != nil {
//		return details.Identification{}, err
//	}
//	return details.Identification{
//		// Map fields from data to details.Identification
//	}, nil
//}
//
//func (repo *ItemDetailedRepositoryImpl) GetManagement(ctx context.Context, niin string) (details.Management, error) {
//	data, err := repo.Db.Management.FindFirst(db.Management.Niin.Equals(niin)).Exec(ctx)
//	if err != nil {
//		return details.Management{}, err
//	}
//	return details.Management{
//		// Map fields from data to details.Management
//	}, nil
//}
//
//func (repo *ItemDetailedRepositoryImpl) GetReference(ctx context.Context, niin string) (details.Reference, error) {
//	data, err := repo.Db.Reference.FindFirst(db.Reference.Niin.Equals(niin)).Exec(ctx)
//	if err != nil {
//		return details.Reference{}, err
//	}
//	return details.Reference{
//		// Map fields from data to details.Reference
//	}, nil
//}
//
//func (repo *ItemDetailedRepositoryImpl) GetFreight(ctx context.Context, niin string) (details.Freight, error) {
//	data, err := repo.Db.Freight.FindFirst(db.Freight.Niin.Equals(niin)).Exec(ctx)
//	if err != nil {
//		return details.Freight{}, err
//	}
//	return details.Freight{
//		// Map fields from data to details.Freight
//	}, nil
//}
//
//func (repo *ItemDetailedRepositoryImpl) GetPackaging(ctx context.Context, niin string) (details.Packaging, error) {
//	data, err := repo.Db.Packaging.FindFirst(db.Packaging.Niin.Equals(niin)).Exec(ctx)
//	if err != nil {
//		return details.Packaging{}, err
//	}
//	return details.Packaging{
//		// Map fields from data to details.Packaging
//	}, nil
//}
//
//func (repo *ItemDetailedRepositoryImpl) GetCharacteristics(ctx context.Context, niin string) (details.Characteristics, error) {
//	data, err := repo.Db.Characteristics.FindFirst(db.Characteristics.Niin.Equals(niin)).Exec(ctx)
//	if err != nil {
//		return details.Characteristics{}, err
//	}
//	return details.Characteristics{
//		// Map fields from data to details.Characteristics
//	}, nil
//}
//
//func (repo *ItemDetailedRepositoryImpl) GetDisposition(ctx context.Context, niin string) (details.Disposition, error) {
//	data, err := repo.Db.Disposition.FindFirst(db.Disposition.Niin.Equals(niin)).Exec(ctx)
//	if err != nil {
//		return details.Disposition{}, err
//	}
//	return details.Disposition{
//		// Map fields from data to details.Disposition
//	}, nil
//}
